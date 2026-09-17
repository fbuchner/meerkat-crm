package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	mcarddav "meerkat/carddav"
	"meerkat/config"
	"meerkat/logger"
	"meerkat/models"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/carddav"
	"gorm.io/gorm"
)

// Sentinel errors for contact sync failures, mapped to API errors in the controller.
var (
	ErrContactSyncInvalidURL     = errors.New("CardDAV URL is invalid")
	ErrContactSyncUnauthorized   = errors.New("CardDAV authentication failed")
	ErrContactSyncForbidden      = errors.New("CardDAV server refused the request")
	ErrContactSyncNotFound       = errors.New("CardDAV resource not found")
	ErrContactSyncUnreachable    = errors.New("CardDAV server could not be reached")
	ErrContactSyncInvalidData    = errors.New("CardDAV server returned data that could not be parsed")
	ErrContactSyncTooLarge       = errors.New("CardDAV response exceeds the size limit")
	ErrContactSyncPrivateAddress = errors.New("CardDAV URL resolves to a private or loopback address")
	ErrContactSyncTooMany        = errors.New("address book exceeds the per-sync contact limit")
	ErrContactSyncNoAddressBooks = errors.New("no address books found on the server")

	// errContactSyncUnsupported triggers fallbacks (e.g. PROPFIND listing when
	// the server lacks the sync-collection REPORT).
	errContactSyncUnsupported = errors.New("server does not support the requested CardDAV operation")
)

const (
	maxContactResponseBytes = 20 * 1024 * 1024
	maxContactsPerSync      = 5000
	multiGetBatchSize       = 50
	contactRequestTimeout   = 60 * time.Second
	contactSyncRunTimeout   = 10 * time.Minute
)

// summarizes one sync run of a CardDAV connection.
type ContactSyncStats struct {
	PulledCreated int `json:"pulled_created"`
	PulledUpdated int `json:"pulled_updated"`
	PulledDeleted int `json:"pulled_deleted"`
	PushedCreated int `json:"pushed_created"`
	PushedUpdated int `json:"pushed_updated"`
	PushedDeleted int `json:"pushed_deleted"`
	Conflicts     int `json:"conflicts"`
	Skipped       int `json:"skipped"`
	Errors        int `json:"errors"`
}

// syncs the user's contacts with an external CardDAV address book (Nextcloud, Radicale, sabre/dav), in the direction configured
type ContactSyncService struct {
	client *http.Client
}

// builds a sync service. blockPrivateURLs enables SSRF protection
func NewContactSyncService(blockPrivateURLs bool) *ContactSyncService {
	transport := &http.Transport{}
	if blockPrivateURLs {
		transport.DialContext = newPrivateBlockingDialContext(ErrContactSyncUnreachable, ErrContactSyncPrivateAddress)
	}
	return &ContactSyncService{
		client: &http.Client{
			Timeout:   contactRequestTimeout,
			Transport: &contactRoundTripper{base: transport},
		},
	}
}

// maps HTTP error responses to sentinel errors, caps response body sizes.
type contactRoundTripper struct {
	base http.RoundTripper
}

func (t *contactRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		resp.Body.Close()
		return nil, ErrContactSyncUnauthorized
	case http.StatusForbidden:
		resp.Body.Close()
		return nil, ErrContactSyncForbidden
	case http.StatusNotFound, http.StatusGone:
		resp.Body.Close()
		return nil, ErrContactSyncNotFound
	case http.StatusMethodNotAllowed, http.StatusNotImplemented:
		resp.Body.Close()
		return nil, errContactSyncUnsupported
	}
	resp.Body = &limitedBody{inner: resp.Body, remaining: maxContactResponseBytes, errTooLarge: ErrContactSyncTooLarge}
	return resp, nil
}

func isContactSyncSentinel(err error) bool {
	return errors.Is(err, ErrContactSyncUnauthorized) || errors.Is(err, ErrContactSyncNotFound) ||
		errors.Is(err, ErrContactSyncTooLarge) || errors.Is(err, ErrContactSyncPrivateAddress) ||
		errors.Is(err, ErrContactSyncForbidden) || errors.Is(err, errContactSyncUnsupported)
}

// validates a CardDAV base or address book URL (inline credentials are stripped)
func NormalizeCardDAVURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, ErrContactSyncInvalidURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, ErrContactSyncInvalidURL
	}
	if parsed.Hostname() == "" {
		return nil, ErrContactSyncInvalidURL
	}
	parsed.Fragment = ""
	parsed.User = nil
	return parsed, nil
}

func (s *ContactSyncService) davClient(baseURL, username, password string) (*carddav.Client, error) {
	var httpClient webdav.HTTPClient = s.client
	if username != "" || password != "" {
		httpClient = webdav.HTTPClientWithBasicAuth(s.client, username, password)
	}
	client, err := carddav.NewClient(httpClient, baseURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrContactSyncInvalidURL, err)
	}
	return client, nil
}

// lists the address books available on a CardDAV server (first principal discovery at the given URL,
// then via /.well-known/carddav, last resort probes whether the URL itself is an address book)
func (s *ContactSyncService) Discover(ctx context.Context, baseURL, username, password string) ([]models.DiscoveredAddressBook, error) {
	parsed, err := NormalizeCardDAVURL(baseURL)
	if err != nil {
		return nil, err
	}

	client, err := s.davClient(parsed.String(), username, password)
	if err != nil {
		return nil, err
	}

	principal, principalErr := client.FindCurrentUserPrincipal(ctx)
	if principalErr != nil {
		if errors.Is(principalErr, ErrContactSyncUnauthorized) || errors.Is(principalErr, ErrContactSyncPrivateAddress) {
			return nil, principalErr
		}
		if resolved, ok := s.resolveWellKnown(ctx, parsed, username, password); ok {
			if wkClient, wkErr := s.davClient(resolved, username, password); wkErr == nil {
				if p, err := wkClient.FindCurrentUserPrincipal(ctx); err == nil {
					client, principal, principalErr = wkClient, p, nil
				}
			}
		}
	}
	if principalErr != nil {
		// The URL may point directly at an address book collection.
		if book, ok := s.probeAddressBook(ctx, client, parsed); ok {
			return []models.DiscoveredAddressBook{book}, nil
		}
		return nil, fmt.Errorf("%w: principal discovery failed: %v", ErrContactSyncUnreachable, principalErr)
	}

	homeSet, err := client.FindAddressBookHomeSet(ctx, principal)
	if err != nil {
		return nil, s.wrapDiscoveryError(err)
	}
	books, err := client.FindAddressBooks(ctx, homeSet)
	if err != nil {
		return nil, s.wrapDiscoveryError(err)
	}
	if len(books) == 0 {
		return nil, ErrContactSyncNoAddressBooks
	}

	all := make([]models.DiscoveredAddressBook, 0, len(books))
	for _, book := range books {
		name := book.Name
		if name == "" {
			name = strings.Trim(lastPathSegment(book.Path), "/")
		}
		all = append(all, models.DiscoveredAddressBook{
			Path:        book.Path,
			Name:        name,
			Description: book.Description,
		})
	}

	discovered := make([]models.DiscoveredAddressBook, 0, len(all))
	for _, book := range all {
		if !isGeneratedAddressBook(book.Path) {
			discovered = append(discovered, book)
		}
	}
	if len(discovered) == 0 {
		// Everything was filtered out. Offering the server-managed books is
		// still better than refusing to connect: the filter is a heuristic and
		// must not be able to lock a user out of a setup we did not foresee.
		return all, nil
	}
	return discovered, nil
}

// generatedAddressBookPrefixes are the collection-name prefixes Nextcloud uses
// for address books it manages itself: "Accounts" (a read-only directory of
// every user on the instance) and per-app books such as "Recently contacted".
// Neither is a meaningful sync target, and offering them invites a user to pick
// a collection that cannot accept writes.
var generatedAddressBookPrefixes = []string{"z-server-generated--", "z-app-generated--"}

func isGeneratedAddressBook(path string) bool {
	segment := strings.ToLower(strings.Trim(lastPathSegment(path), "/"))
	for _, prefix := range generatedAddressBookPrefixes {
		if strings.HasPrefix(segment, prefix) {
			return true
		}
	}
	return false
}

func (s *ContactSyncService) wrapDiscoveryError(err error) error {
	if isContactSyncSentinel(err) {
		return err
	}
	return fmt.Errorf("%w: %v", ErrContactSyncUnreachable, err)
}

// resolveWellKnown resolves the /.well-known/carddav endpoint of the server,
// following the redirect manually (Go's client rewrites non-GET methods on
// 301/302, which breaks PROPFIND-based discovery through redirects).
func (s *ContactSyncService) resolveWellKnown(ctx context.Context, base *url.URL, username, password string) (string, bool) {
	wellKnown := *base
	wellKnown.Path = "/.well-known/carddav"
	wellKnown.RawQuery = ""

	noRedirect := &http.Client{
		Timeout:   s.client.Timeout,
		Transport: s.client.Transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wellKnown.String(), nil)
	if err != nil {
		return "", false
	}
	if username != "" || password != "" {
		req.SetBasicAuth(username, password)
	}
	resp, err := noRedirect.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		location := resp.Header.Get("Location")
		if location == "" {
			return "", false
		}
		resolved, err := wellKnown.Parse(location)
		if err != nil {
			return "", false
		}
		// The caller attaches the user's Basic credentials to whatever this
		// resolves to, and Location is under the remote server's control. Only
		// follow it within the same host, and never down from https to http —
		// otherwise a hostile or compromised server can harvest the password by
		// redirecting discovery at itself in the clear.
		if !strings.EqualFold(resolved.Hostname(), base.Hostname()) {
			return "", false
		}
		switch resolved.Scheme {
		case "https":
		case "http":
			if base.Scheme == "https" {
				return "", false // no downgrade
			}
		default:
			return "", false
		}
		resolved.User = nil
		return resolved.String(), true
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return wellKnown.String(), true
	}
	return "", false
}

// probeAddressBook checks whether the URL is itself a CardDAV address book.
func (s *ContactSyncService) probeAddressBook(ctx context.Context, client *carddav.Client, parsed *url.URL) (models.DiscoveredAddressBook, bool) {
	if err := client.HasSupport(ctx); err != nil {
		return models.DiscoveredAddressBook{}, false
	}
	info, err := client.Stat(ctx, parsed.Path)
	if err != nil || !info.IsDir {
		return models.DiscoveredAddressBook{}, false
	}
	return models.DiscoveredAddressBook{
		Path: parsed.Path,
		Name: strings.Trim(lastPathSegment(parsed.Path), "/"),
	}, true
}

func lastPathSegment(p string) string {
	trimmed := strings.TrimSuffix(p, "/")
	if idx := strings.LastIndex(trimmed, "/"); idx >= 0 {
		return trimmed[idx+1:]
	}
	return trimmed
}

// canonicalVCardHash fingerprints a vCard independent of property order (a
// vcard.Card is a map, so encoding order is nondeterministic) and of volatile
// properties (REV, PRODID). Used to detect local contact changes without
// relying on UpdatedAt, which unrelated saves also bump.
func canonicalVCardHash(card vcard.Card) string {
	var buf bytes.Buffer
	if err := vcard.NewEncoder(&buf).Encode(card); err != nil {
		return ""
	}
	lines := strings.Split(buf.String(), "\r\n")
	kept := lines[:0]
	for _, line := range lines {
		upper := strings.ToUpper(line)
		if line == "" || strings.HasPrefix(upper, "REV:") || strings.HasPrefix(upper, "REV;") ||
			strings.HasPrefix(upper, "PRODID:") || strings.HasPrefix(upper, "PRODID;") {
			continue
		}
		kept = append(kept, line)
	}
	sort.Strings(kept)
	sum := sha256.Sum256([]byte(strings.Join(kept, "\n")))
	return hex.EncodeToString(sum[:])
}

// contactConnectionLocks serializes runs of the same connection, mirroring the
// calendar sync service. The scheduled job and a manual sync from the UI would
// otherwise race — main.go kicks off a sync on startup, exactly when a user is
// likely to press "Sync now" — and two concurrent runs double-push contacts and
// collide on the carddav_contact_links unique indexes.
var contactConnectionLocks sync.Map // connection ID → *sync.Mutex

// SyncConnection syncs the connection and updates its last-sync bookkeeping.
func (s *ContactSyncService) SyncConnection(ctx context.Context, db *gorm.DB, cfg config.Config, conn *models.CardDAVConnection) (ContactSyncStats, error) {
	lock, _ := contactConnectionLocks.LoadOrStore(conn.ID, &sync.Mutex{})
	mutex := lock.(*sync.Mutex)
	mutex.Lock()
	defer mutex.Unlock()

	run, err := s.syncConnection(ctx, db, cfg, conn)

	var stats ContactSyncStats
	var itemErr error
	if run != nil {
		stats, itemErr = run.stats, run.firstErr
	}

	now := time.Now().UTC()
	conn.LastSyncedAt = &now
	switch {
	case err != nil:
		conn.LastSyncStatus = models.CardDAVSyncStatusError
		conn.LastSyncError = clampRunes(err.Error(), 500)
	case stats.Errors > 0:
		conn.LastSyncStatus = models.CardDAVSyncStatusPartial
		conn.LastSyncError = clampRunes(fmt.Sprintf("%d contacts failed to sync: %v", stats.Errors, itemErr), 500)
	default:
		conn.LastSyncStatus = models.CardDAVSyncStatusSuccess
		conn.LastSyncError = ""
	}
	// The counts are persisted because the run that produced them is started in
	// the background: the request that kicked it off is long gone by the time
	// the UI polls for the outcome.
	if encoded, marshalErr := json.Marshal(stats); marshalErr == nil {
		conn.LastSyncStats = string(encoded)
	}

	if saveErr := db.Model(&models.CardDAVConnection{}).Where("id = ?", conn.ID).Updates(map[string]interface{}{
		"sync_token":       conn.SyncToken,
		"last_synced_at":   conn.LastSyncedAt,
		"last_sync_status": conn.LastSyncStatus,
		"last_sync_error":  conn.LastSyncError,
		"last_sync_stats":  conn.LastSyncStats,
	}).Error; saveErr != nil {
		logger.Error().Err(saveErr).Uint("connection_id", conn.ID).Msg("Failed to update contact sync status")
	}

	return stats, err
}

// activeContactSyncs marks connections whose sync was started through the API
// and has not finished yet. It backs the "syncing" state the UI polls for; a
// process restart clears it, which is correct, since no run survives a restart.
var activeContactSyncs sync.Map // connection ID → struct{}

// IsContactSyncRunning reports whether an API-initiated sync is in flight.
func IsContactSyncRunning(connectionID uint) bool {
	_, running := activeContactSyncs.Load(connectionID)
	return running
}

// StartContactSync runs a sync in the background, reporting whether it started.
// It returns false when one is already in flight for the connection, so a
// second press of "Sync now" cannot queue a duplicate run.
//
// A first sync of a large address book takes minutes — one multi-get per 50
// contacts plus a PUT per local contact — which does not fit in an HTTP
// request, so callers start the run here and poll the connection for its
// outcome. The run takes its own context and its own copy of the connection for
// exactly that reason: it must outlive the request that started it.
func StartContactSync(db *gorm.DB, cfg config.Config, conn models.CardDAVConnection) bool {
	if _, alreadyRunning := activeContactSyncs.LoadOrStore(conn.ID, struct{}{}); alreadyRunning {
		return false
	}

	go func() {
		defer activeContactSyncs.Delete(conn.ID)

		// The timeout bounds the CardDAV traffic. The database session gets a
		// background context of its own so the run does not inherit (now or
		// after some future change to the db middleware) a request context that
		// is already cancelled by the time this goroutine writes anything.
		ctx, cancel := context.WithTimeout(context.Background(), contactSyncRunTimeout)
		defer cancel()
		db := db.WithContext(context.Background())

		service := NewContactSyncService(cfg.CardDAVBlockPrivateURLs)
		stats, err := service.SyncConnection(ctx, db, cfg, &conn)
		if err != nil {
			// The failure is already recorded on the connection for the UI.
			logger.Warn().Err(err).Uint("connection_id", conn.ID).Uint("user_id", conn.UserID).
				Msg("Requested contact sync failed")
			return
		}
		logger.Info().Uint("connection_id", conn.ID).
			Int("pulled_created", stats.PulledCreated).Int("pulled_updated", stats.PulledUpdated).Int("pulled_deleted", stats.PulledDeleted).
			Int("pushed_created", stats.PushedCreated).Int("pushed_updated", stats.PushedUpdated).Int("pushed_deleted", stats.PushedDeleted).
			Int("conflicts", stats.Conflicts).Int("errors", stats.Errors).
			Msg("Requested contact sync completed")
	}()
	return true
}

// remoteObject is one address object on the server, possibly without its card
// (sync-collection REPORTs return etags only; cards are multi-getted on demand).
type remoteObject struct {
	Path string
	ETag string
	Card vcard.Card
}

// remoteState is the server-side view of the address book for one run. In
// token mode only changed objects appear and deletions are explicit; in a full
// listing absence implies deletion.
type remoteState struct {
	objects      map[string]*remoteObject
	deletedPaths map[string]bool
	newSyncToken string
	fullListing  bool
}

// contactSyncRun carries the state of one sync run.
type contactSyncRun struct {
	ctx      context.Context
	db       *gorm.DB
	cfg      config.Config
	conn     *models.CardDAVConnection
	client   *carddav.Client
	remote   *remoteState
	stats    ContactSyncStats
	firstErr error

	links       []models.CardDAVContactLink
	linkByPath  map[string]*models.CardDAVContactLink
	linkByUID   map[string]*models.CardDAVContactLink
	contactByID map[uint]*models.Contact
	// unlinked live contacts, keyed for match & merge
	unlinkedByUID map[string]*models.Contact

	// localHashes memoizes canonical vCard hashes by contact ID for this run.
	// Building a card reads the contact's photo off disk and base64-encodes it,
	// so the change-detection and push phases must not each pay for it.
	localHashes map[uint]string
}

// localHash returns the contact's canonical vCard hash, building the card at
// most once per run. Callers that mutate a contact must refresh the entry via
// setLocalHash or drop it with invalidateLocalHash.
func (r *contactSyncRun) localHash(contact *models.Contact) string {
	if hash, ok := r.localHashes[contact.ID]; ok {
		return hash
	}
	hash := canonicalVCardHash(mcarddav.ContactToVCard(contact, r.cfg.ProfilePhotoDir))
	r.setLocalHash(contact.ID, hash)
	return hash
}

func (r *contactSyncRun) setLocalHash(contactID uint, hash string) {
	if contactID != 0 {
		r.localHashes[contactID] = hash
	}
}

func (r *contactSyncRun) invalidateLocalHash(contactID uint) {
	delete(r.localHashes, contactID)
}

func (r *contactSyncRun) pullEnabled() bool {
	return r.conn.Direction != models.CardDAVDirectionPush
}

func (r *contactSyncRun) pushEnabled() bool {
	return r.conn.Direction != models.CardDAVDirectionPull
}

func (r *contactSyncRun) recordItemError(err error, msg string, fields map[string]string) {
	r.stats.Errors++
	if r.firstErr == nil {
		r.firstErr = err
	}
	event := logger.Warn().Err(err).Uint("connection_id", r.conn.ID)
	for k, v := range fields {
		event = event.Str(k, v)
	}
	event.Msg(msg)
}

func (s *ContactSyncService) syncConnection(ctx context.Context, db *gorm.DB, cfg config.Config, conn *models.CardDAVConnection) (*contactSyncRun, error) {
	// Normalizing here also strips any inline credentials from rows stored
	// before NormalizeCardDAVURL did so.
	parsed, err := NormalizeCardDAVURL(conn.BaseURL)
	if err != nil {
		return nil, err
	}

	password, err := DecryptCredential(cfg.JWTSecretKey, conn.PasswordEncrypted)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrContactSyncUnauthorized, err)
	}

	client, err := s.davClient(parsed.String(), conn.Username, password)
	if err != nil {
		return nil, err
	}

	run := &contactSyncRun{
		ctx: ctx, db: db, cfg: cfg, conn: conn, client: client,
		localHashes: map[uint]string{},
	}

	if err := run.loadLocalState(); err != nil {
		return run, err
	}
	if err := run.fetchRemoteState(); err != nil {
		return run, err
	}
	if err := run.fetchNeededCards(); err != nil {
		return run, err
	}

	run.reconcileRemoteObjects()
	run.reconcileRemoteDeletions()
	run.reconcileLocalChanges()
	run.pushLocalCreations()

	// Only advance the sync token when every item landed. The token tells the
	// server "I have everything up to here"; moving it past an object that
	// failed would mean the next incremental REPORT never offers that object
	// again and a transient failure silently becomes permanent data loss.
	// Keeping the old token costs one redundant full-ish delta next run.
	if run.stats.Errors == 0 {
		conn.SyncToken = run.remote.newSyncToken
	}
	return run, nil
}

func (r *contactSyncRun) loadLocalState() error {
	if err := r.db.Where("connection_id = ?", r.conn.ID).Find(&r.links).Error; err != nil {
		return fmt.Errorf("failed to load contact links: %w", err)
	}

	// Soft-deleted contacts are loaded too: they are the tombstones that tell
	// the reconciler a contact was deleted locally rather than never existing.
	// They do not count towards the sync limit, which is about live contacts —
	// otherwise a user who has deleted enough contacts could never sync again.
	var contacts []models.Contact
	if err := r.db.Unscoped().Preload("Relationships.RelatedContact").Where("user_id = ?", r.conn.UserID).Find(&contacts).Error; err != nil {
		return fmt.Errorf("failed to load contacts: %w", err)
	}

	r.contactByID = make(map[uint]*models.Contact, len(contacts))
	live := 0
	for i := range contacts {
		r.contactByID[contacts[i].ID] = &contacts[i]
		if !contacts[i].DeletedAt.Valid {
			live++
		}
	}
	if live > maxContactsPerSync {
		return ErrContactSyncTooMany
	}

	r.linkByPath = make(map[string]*models.CardDAVContactLink, len(r.links))
	r.linkByUID = make(map[string]*models.CardDAVContactLink, len(r.links))
	linkedContactIDs := make(map[uint]bool, len(r.links))
	for i := range r.links {
		link := &r.links[i]
		r.linkByPath[link.RemotePath] = link
		r.linkByUID[link.RemoteUID] = link
		linkedContactIDs[link.ContactID] = true
	}

	r.unlinkedByUID = make(map[string]*models.Contact)
	for _, contact := range r.contactByID {
		if !linkedContactIDs[contact.ID] && !contact.DeletedAt.Valid && contact.VCardUID != "" {
			r.unlinkedByUID[contact.VCardUID] = contact
		}
	}
	return nil
}

// fetchRemoteState determines what changed on the server, preferring the
// sync-collection REPORT (with stored token) and falling back to a PROPFIND
// depth-1 etag listing for servers without sync-collection support.
func (r *contactSyncRun) fetchRemoteState() error {
	state := &remoteState{objects: map[string]*remoteObject{}, deletedPaths: map[string]bool{}}

	tryTokens := []string{}
	if r.conn.SyncToken != "" {
		tryTokens = append(tryTokens, r.conn.SyncToken)
	}
	tryTokens = append(tryTokens, "")

	var lastErr error
	for _, token := range tryTokens {
		resp, err := r.client.SyncCollection(r.ctx, r.conn.AddressBookPath, &carddav.SyncQuery{SyncToken: token})
		if err != nil {
			// A rejected token (expired, server restart) surfaces as an
			// arbitrary error; retry with a fresh full sync before giving up.
			lastErr = err
			if errors.Is(err, ErrContactSyncUnauthorized) || errors.Is(err, ErrContactSyncPrivateAddress) ||
				errors.Is(err, ErrContactSyncTooLarge) {
				return err
			}
			continue
		}
		for i := range resp.Updated {
			obj := resp.Updated[i]
			state.objects[obj.Path] = &remoteObject{Path: obj.Path, ETag: obj.ETag, Card: obj.Card}
		}
		for _, deletedPath := range resp.Deleted {
			state.deletedPaths[deletedPath] = true
		}
		state.newSyncToken = resp.SyncToken
		state.fullListing = token == ""
		if state.fullListing && len(state.objects) > maxContactsPerSync {
			return ErrContactSyncTooMany
		}
		r.remote = state
		return nil
	}

	// No sync-collection support: fall back to a full addressbook-query
	// listing, which RFC 6352 requires every CardDAV server to support.
	logger.Debug().Err(lastErr).Uint("connection_id", r.conn.ID).
		Msg("sync-collection REPORT unavailable, falling back to addressbook-query listing")
	// FilterAllOf with no prop filters matches every object (an empty anyof
	// matches none, per go-webdav's Filter semantics).
	objects, err := r.client.QueryAddressBook(r.ctx, r.conn.AddressBookPath, &carddav.AddressBookQuery{
		DataRequest: carddav.AddressDataRequest{AllProp: true},
		FilterTest:  carddav.FilterAllOf,
	})
	if err != nil {
		if isContactSyncSentinel(err) {
			return err
		}
		return fmt.Errorf("%w: %v", ErrContactSyncUnreachable, err)
	}
	for i := range objects {
		obj := objects[i]
		state.objects[obj.Path] = &remoteObject{Path: obj.Path, ETag: obj.ETag, Card: obj.Card}
	}
	if len(state.objects) > maxContactsPerSync {
		return ErrContactSyncTooMany
	}
	state.fullListing = true
	state.newSyncToken = ""
	r.remote = state
	return nil
}

// fetchNeededCards multi-gets vCard data for remote objects the reconciler
// will actually read: new or changed objects, and only when pulling.
func (r *contactSyncRun) fetchNeededCards() error {
	if !r.pullEnabled() {
		return nil
	}

	var needed []string
	for path, obj := range r.remote.objects {
		if obj.Card != nil {
			continue
		}
		link, linked := r.linkByPath[path]
		if linked && link.RemoteETag == obj.ETag {
			continue // unchanged
		}
		needed = append(needed, path)
	}
	sort.Strings(needed)

	for start := 0; start < len(needed); start += multiGetBatchSize {
		end := start + multiGetBatchSize
		if end > len(needed) {
			end = len(needed)
		}
		objects, err := r.client.MultiGetAddressBook(r.ctx, r.conn.AddressBookPath, &carddav.AddressBookMultiGet{
			Paths:       needed[start:end],
			DataRequest: carddav.AddressDataRequest{AllProp: true},
		})
		if err != nil {
			if isContactSyncSentinel(err) {
				return err
			}
			return fmt.Errorf("%w: %v", ErrContactSyncUnreachable, err)
		}
		for i := range objects {
			obj := objects[i]
			if existing, ok := r.remote.objects[obj.Path]; ok {
				existing.Card = obj.Card
				if obj.ETag != "" {
					existing.ETag = obj.ETag
				}
			}
		}
	}
	return nil
}

func remoteUID(obj *remoteObject) string {
	if obj.Card != nil {
		if uid := obj.Card.Value(vcard.FieldUID); uid != "" {
			return uid
		}
	}
	return strings.TrimSuffix(lastPathSegment(obj.Path), ".vcf")
}

// reconcileRemoteObjects walks the server-side objects and applies creations,
// updates and conflicts according to the sync direction.
func (r *contactSyncRun) reconcileRemoteObjects() {
	paths := make([]string, 0, len(r.remote.objects))
	for path := range r.remote.objects {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		obj := r.remote.objects[path]
		link, linked := r.linkByPath[path]

		if !linked {
			// The object may be a known contact moved to a new path.
			if uidLink, ok := r.linkByUID[remoteUID(obj)]; ok {
				oldPath := uidLink.RemotePath
				uidLink.RemotePath = obj.Path
				delete(r.linkByPath, oldPath)
				r.linkByPath[obj.Path] = uidLink
				link, linked = uidLink, true
				// Persist the move now. When the etag is unchanged nothing below
				// writes the link, and a stale RemotePath in the database would
				// send the next push to the old location, duplicating the object
				// on the server.
				if link.ID != 0 {
					r.saveLink(link)
				}
			}
		}

		if linked {
			if link.RemoteETag == obj.ETag {
				continue // unchanged remotely; the local pass handles local edits
			}
			r.applyRemoteChange(obj, link)
			continue
		}

		if !r.pullEnabled() {
			continue // push-only ignores remote additions
		}
		r.mergeRemoteCreation(obj)
	}
}

// applyRemoteChange handles a remotely modified object: pull it, resolve a
// conflict, or restore an edit that raced a local deletion.
func (r *contactSyncRun) applyRemoteChange(obj *remoteObject, link *models.CardDAVContactLink) {
	contact := r.contactByID[link.ContactID]

	if !r.pullEnabled() {
		// Push-only: Meerkat is authoritative.
		if contact == nil || contact.DeletedAt.Valid {
			// Adopt the etag in memory (deliberately not persisted) so that
			// reconcileLocalChanges does not mistake this object for a remote
			// change already handled here and skip the deletion it owns.
			link.RemoteETag = obj.ETag
			return // the local pass deletes the remote object
		}
		if r.localChanged(contact, link) {
			r.pushContact(contact, link, false) // local state overwrites the remote edit
			return
		}
		// No local edit to assert; adopt the etag so the object isn't reprocessed.
		link.RemoteETag = obj.ETag
		r.saveLink(link)
		return
	}

	if obj.Card == nil {
		r.recordItemError(ErrContactSyncInvalidData, "Remote contact has no vCard data", map[string]string{"path": obj.Path})
		return
	}

	switch {
	case contact == nil:
		// Link points at a vanished row; recreate from remote.
		r.pullCreate(obj, link)
	case contact.DeletedAt.Valid:
		if !r.pushEnabled() {
			// Pull-only: the local deletion is a local-side decision that the
			// server never learns about; never resurrect the contact.
			link.RemoteETag = obj.ETag
			r.saveLink(link)
			r.stats.Skipped++
			return
		}
		// Edit beats delete: restore the contact with the remote data.
		// UpdateColumn skips the Contact hooks, which would run on a model
		// without a primary key here.
		if err := r.db.Unscoped().Model(&models.Contact{}).Where("id = ?", contact.ID).UpdateColumn("deleted_at", nil).Error; err != nil {
			r.recordItemError(err, "Failed to restore deleted contact", map[string]string{"path": obj.Path})
			return
		}
		contact.DeletedAt = gorm.DeletedAt{}
		r.pullUpdate(obj, link, contact)
	case r.localChanged(contact, link):
		// Both sides moved since the recorded merge base: a real conflict.
		r.stats.Conflicts++
		r.resolveByRevision(obj, link, contact)
	default:
		r.pullUpdate(obj, link, contact)
	}
}

// mergeRemoteCreation links a new remote object to an existing local contact
// when possible (by UID, then by name+email), otherwise creates the contact.
func (r *contactSyncRun) mergeRemoteCreation(obj *remoteObject) {
	if obj.Card == nil {
		r.recordItemError(ErrContactSyncInvalidData, "Remote contact has no vCard data", map[string]string{"path": obj.Path})
		return
	}
	uid := remoteUID(obj)

	if contact, ok := r.unlinkedByUID[uid]; ok {
		delete(r.unlinkedByUID, uid)
		link := r.newLink(contact, obj)
		r.resolveByRevision(obj, link, contact)
		return
	}

	if contact := r.matchByIdentity(obj.Card); contact != nil {
		delete(r.unlinkedByUID, contact.VCardUID)
		// The remote UID wins so the link key, future pushes, and the built-in
		// CardDAV server all agree on one identity. Persisted immediately: if
		// the conflict below resolves by pushing rather than pulling, nothing
		// else writes the contact row, and a push would otherwise re-send the
		// stale UID while targeting the link's (correct) new remote path —
		// a UID/path mismatch CardDAV servers reject.
		contact.VCardUID = uid
		r.invalidateLocalHash(contact.ID) // UID is part of the card
		if err := r.db.Model(&models.Contact{}).Where("id = ?", contact.ID).UpdateColumn("vcard_uid", uid).Error; err != nil {
			r.recordItemError(err, "Failed to persist matched contact UID", map[string]string{"path": obj.Path})
			return
		}
		link := r.newLink(contact, obj)
		r.resolveByRevision(obj, link, contact)
		return
	}

	r.pullCreate(obj, nil)
}

// matchByIdentity finds an unlinked local contact with the same normalized
// name and primary email as the card. Ambiguous matches return nil: creating
// a duplicate is recoverable, silently merging the wrong people is not.
func (r *contactSyncRun) matchByIdentity(card vcard.Card) *models.Contact {
	imported, _, _, _, _ := mcarddav.VCardToContact(card, nil)
	key := identityKey(imported.Firstname, imported.Lastname, imported.Email)
	if key == "" {
		return nil
	}

	var match *models.Contact
	for _, contact := range r.unlinkedByUID {
		if identityKey(contact.Firstname, contact.Lastname, contact.Email) != key {
			continue
		}
		if match != nil {
			return nil // ambiguous
		}
		match = contact
	}
	return match
}

func identityKey(firstname, lastname, email string) string {
	name := strings.ToLower(strings.TrimSpace(firstname + " " + lastname))
	if name == "" {
		return ""
	}
	return name + "\x1f" + strings.ToLower(strings.TrimSpace(email))
}

func (r *contactSyncRun) newLink(contact *models.Contact, obj *remoteObject) *models.CardDAVContactLink {
	link := &models.CardDAVContactLink{
		ConnectionID: r.conn.ID,
		UserID:       r.conn.UserID,
		ContactID:    contact.ID,
		RemoteUID:    remoteUID(obj),
		RemotePath:   obj.Path,
	}
	r.linkByPath[obj.Path] = link
	r.linkByUID[link.RemoteUID] = link
	return link
}

// pullCreate imports a remote object as a new local contact. When link is
// non-nil the existing link row is reused (contact row vanished).
func (r *contactSyncRun) pullCreate(obj *remoteObject, link *models.CardDAVContactLink) {
	contact, relationships, photoData, mediaType, photoURL := mcarddav.VCardToContact(obj.Card, nil)
	contact.UserID = r.conn.UserID
	if contact.VCardUID == "" {
		contact.VCardUID = remoteUID(obj)
	}
	r.applyPhoto(contact, photoData, mediaType, photoURL)

	var hash string
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(contact).Error; err != nil {
			return err
		}
		if len(relationships) > 0 {
			if err := mcarddav.ResolveAndUpsertRelationships(tx, r.conn.UserID, contact.ID, relationships); err != nil {
				return err
			}
		}
		now := time.Now().UTC()
		if link == nil {
			link = &models.CardDAVContactLink{
				ConnectionID: r.conn.ID,
				UserID:       r.conn.UserID,
				RemoteUID:    remoteUID(obj),
				RemotePath:   obj.Path,
			}
		}
		hash = canonicalVCardHash(mcarddav.ContactToVCard(contact, r.cfg.ProfilePhotoDir))
		link.ContactID = contact.ID
		link.RemoteETag = obj.ETag
		link.LocalHash = hash
		link.SyncedAt = &now
		return tx.Save(link).Error
	})
	if err != nil {
		r.recordItemError(err, "Failed to import remote contact", map[string]string{"path": obj.Path})
		return
	}
	r.contactByID[contact.ID] = contact
	r.setLocalHash(contact.ID, hash)
	r.linkByPath[obj.Path] = link
	r.linkByUID[link.RemoteUID] = link
	r.stats.PulledCreated++
}

// pullUpdate applies a remote card onto an existing local contact.
func (r *contactSyncRun) pullUpdate(obj *remoteObject, link *models.CardDAVContactLink, contact *models.Contact) {
	updated, relationships, photoData, mediaType, photoURL := mcarddav.VCardToContact(obj.Card, contact)
	updated.UserID = r.conn.UserID
	r.applyPhoto(updated, photoData, mediaType, photoURL)
	r.invalidateLocalHash(updated.ID) // the contact just changed

	var hash string
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(updated).Error; err != nil {
			return err
		}
		if len(relationships) > 0 {
			if err := mcarddav.ResolveAndUpsertRelationships(tx, r.conn.UserID, updated.ID, relationships); err != nil {
				return err
			}
		}
		now := time.Now().UTC()
		hash = canonicalVCardHash(mcarddav.ContactToVCard(updated, r.cfg.ProfilePhotoDir))
		link.ContactID = updated.ID
		link.RemoteETag = obj.ETag
		link.LocalHash = hash
		link.SyncedAt = &now
		return tx.Save(link).Error
	})
	if err != nil {
		r.recordItemError(err, "Failed to update contact from remote", map[string]string{"path": obj.Path})
		return
	}
	r.contactByID[updated.ID] = updated
	r.setLocalHash(updated.ID, hash)
	r.stats.PulledUpdated++
}

// resolveByRevision picks a winner for a contact that exists on both sides:
// newest wins, comparing the remote card's REV against the local UpdatedAt.
// Without a parseable REV the remote wins — the external address book is
// usually the hub other devices write to, and VCardExtra round-tripping means
// unmapped remote data is never destroyed.
//
// This is used both for genuine conflicts and for first-time pairings, which
// have no merge base to diff against. Only the caller knows which it is, so
// only the caller counts conflicts.
func (r *contactSyncRun) resolveByRevision(obj *remoteObject, link *models.CardDAVContactLink, contact *models.Contact) {
	localWins := false
	if rev, ok := mcarddav.ParseRevision(obj.Card); ok {
		localWins = contact.UpdatedAt.After(rev)
	}

	if localWins && r.pushEnabled() {
		r.pushContact(contact, link, false)
		return
	}
	r.pullUpdate(obj, link, contact)
}

// applyPhoto stores an embedded or URL-referenced photo on the contact,
// mirroring the CardDAV server's PutAddressObject behavior. Photo failures
// are logged, never fatal.
func (r *contactSyncRun) applyPhoto(contact *models.Contact, photoData []byte, mediaType, photoURL string) {
	if len(photoData) == 0 && photoURL != "" {
		fetched, fetchedType, err := mcarddav.FetchPhotoFromURL(photoURL)
		if err != nil {
			logger.Warn().Err(err).Str("photo_url", photoURL).Msg("Contact sync: failed to fetch photo from URL")
			return
		}
		photoData, mediaType = fetched, fetchedType
	}
	if len(photoData) == 0 {
		return
	}
	photoPath, thumbnail, err := mcarddav.SaveContactPhoto(photoData, mediaType, r.cfg.ProfilePhotoDir)
	if err != nil {
		logger.Warn().Err(err).Msg("Contact sync: failed to save contact photo")
		return
	}
	contact.Photo = photoPath
	contact.PhotoThumbnail = thumbnail
}

// reconcileRemoteDeletions handles objects that disappeared from the server.
func (r *contactSyncRun) reconcileRemoteDeletions() {
	for i := range r.links {
		link := &r.links[i]
		deleted := r.remote.deletedPaths[link.RemotePath]
		if r.remote.fullListing {
			_, present := r.remote.objects[link.RemotePath]
			deleted = deleted || !present
		}
		if !deleted {
			continue
		}

		contact := r.contactByID[link.ContactID]
		localGone := contact == nil || contact.DeletedAt.Valid

		switch {
		case localGone:
			// Deleted on both sides: just drop the link.
			r.dropLink(link)
		case r.pushEnabled() && (!r.pullEnabled() || r.localChanged(contact, link)):
			// Push-only (Meerkat is authoritative) or edit beats delete —
			// re-create the object on the server.
			link.RemotePath = r.objectPath(contact.VCardUID)
			link.RemoteUID = contact.VCardUID
			link.RemoteETag = ""
			r.pushContact(contact, link, false)
		default:
			// Pull is necessarily enabled here: the case above covers every
			// push-only path. Deleted remotely, so soft-delete locally.
			if err := r.db.Transaction(func(tx *gorm.DB) error {
				if err := tx.Delete(&models.Contact{}, contact.ID).Error; err != nil {
					return err
				}
				return tx.Delete(link).Error
			}); err != nil {
				r.recordItemError(err, "Failed to delete contact removed on server", map[string]string{"path": link.RemotePath})
				continue
			}
			r.markLinkDropped(link)
			r.stats.PulledDeleted++
		}
	}
}

// reconcileLocalChanges walks the surviving links for local edits/deletions
// that the remote passes did not already handle.
func (r *contactSyncRun) reconcileLocalChanges() {
	for i := range r.links {
		link := &r.links[i]
		if link.ID == 0 && link.ContactID == 0 {
			continue // dropped earlier this run
		}
		if _, stillLinked := r.linkByPath[link.RemotePath]; !stillLinked {
			continue // dropped earlier this run
		}
		if r.remote.deletedPaths[link.RemotePath] {
			continue // handled by reconcileRemoteDeletions
		}
		if r.remote.fullListing {
			if _, present := r.remote.objects[link.RemotePath]; !present {
				continue // handled by reconcileRemoteDeletions
			}
		}
		if obj, ok := r.remote.objects[link.RemotePath]; ok && obj.ETag != link.RemoteETag {
			continue // handled by reconcileRemoteObjects
		}

		contact := r.contactByID[link.ContactID]
		localGone := contact == nil || contact.DeletedAt.Valid

		if localGone {
			if !r.pushEnabled() {
				// Pull-only: the soft-deleted row is a tombstone that prevents
				// re-import; keep the link so the deletion is remembered.
				continue
			}
			if err := r.client.RemoveAll(r.ctx, link.RemotePath); err != nil && !errors.Is(err, ErrContactSyncNotFound) {
				r.recordItemError(err, "Failed to delete contact on server", map[string]string{"path": link.RemotePath})
				continue
			}
			r.dropLink(link)
			r.stats.PushedDeleted++
			continue
		}

		if r.pushEnabled() && r.localChanged(contact, link) {
			r.pushContact(contact, link, false)
		}
	}
}

// pushLocalCreations uploads live contacts that have no link yet.
func (r *contactSyncRun) pushLocalCreations() {
	if !r.pushEnabled() {
		return
	}
	uids := make([]string, 0, len(r.unlinkedByUID))
	for uid := range r.unlinkedByUID {
		uids = append(uids, uid)
	}
	sort.Strings(uids)

	for _, uid := range uids {
		contact := r.unlinkedByUID[uid]
		link := &models.CardDAVContactLink{
			ConnectionID: r.conn.ID,
			UserID:       r.conn.UserID,
			ContactID:    contact.ID,
			RemoteUID:    contact.VCardUID,
			RemotePath:   r.objectPath(contact.VCardUID),
		}
		r.pushContact(contact, link, true)
	}
}

func (r *contactSyncRun) objectPath(uid string) string {
	base := strings.TrimSuffix(r.conn.AddressBookPath, "/")
	return base + "/" + url.PathEscape(uid) + ".vcf"
}

// localChanged reports whether the contact's vCard representation drifted
// from the state recorded at the last sync.
func (r *contactSyncRun) localChanged(contact *models.Contact, link *models.CardDAVContactLink) bool {
	return r.localHash(contact) != link.LocalHash
}

// pushContact PUTs the contact's vCard to the server and records the new
// link state. isCreation only affects which stat is bumped.
func (r *contactSyncRun) pushContact(contact *models.Contact, link *models.CardDAVContactLink, isCreation bool) {
	card := mcarddav.ContactToVCard(contact, r.cfg.ProfilePhotoDir)

	putObj, err := r.client.PutAddressObject(r.ctx, link.RemotePath, card)
	if err != nil {
		r.recordItemError(err, "Failed to push contact to server", map[string]string{"path": link.RemotePath})
		return
	}

	etag := ""
	if putObj != nil {
		etag = putObj.ETag
		if putObj.Path != "" && putObj.Path != link.RemotePath {
			delete(r.linkByPath, link.RemotePath)
			link.RemotePath = putObj.Path
			r.linkByPath[link.RemotePath] = link
		}
	}
	if etag == "" {
		// Servers may omit the ETag on PUT; fetch it so the next run doesn't
		// mistake our own write for a remote change.
		if info, statErr := r.client.Stat(r.ctx, link.RemotePath); statErr == nil {
			etag = info.ETag
		}
	}

	now := time.Now().UTC()
	link.RemoteETag = etag
	link.LocalHash = canonicalVCardHash(card)
	link.SyncedAt = &now
	if err := r.db.Save(link).Error; err != nil {
		r.recordItemError(err, "Failed to record pushed contact", map[string]string{"path": link.RemotePath})
		return
	}
	r.setLocalHash(contact.ID, link.LocalHash)
	r.linkByPath[link.RemotePath] = link
	r.linkByUID[link.RemoteUID] = link
	if isCreation {
		r.stats.PushedCreated++
	} else {
		r.stats.PushedUpdated++
	}
}

func (r *contactSyncRun) saveLink(link *models.CardDAVContactLink) {
	if err := r.db.Save(link).Error; err != nil {
		r.recordItemError(err, "Failed to save contact link", map[string]string{"path": link.RemotePath})
	}
}

// dropLink deletes the link row and forgets it for the rest of the run.
func (r *contactSyncRun) dropLink(link *models.CardDAVContactLink) {
	if link.ID != 0 {
		if err := r.db.Delete(link).Error; err != nil {
			r.recordItemError(err, "Failed to delete contact link", map[string]string{"path": link.RemotePath})
			return
		}
	}
	r.markLinkDropped(link)
}

// markLinkDropped unregisters a link and zeroes its identity, which is how
// later phases recognize a link this run already disposed of.
func (r *contactSyncRun) markLinkDropped(link *models.CardDAVContactLink) {
	delete(r.linkByPath, link.RemotePath)
	delete(r.linkByUID, link.RemoteUID)
	link.ContactID = 0
	link.ID = 0
}

// SyncAllContactConnections syncs every enabled connection. Failures on one
// connection are recorded on it and do not stop the others.
func SyncAllContactConnections(db *gorm.DB, cfg config.Config) {
	var connections []models.CardDAVConnection
	if err := db.Where("sync_enabled = ?", true).Find(&connections).Error; err != nil {
		logger.Error().Err(err).Msg("Failed to load CardDAV connections for scheduled sync")
		return
	}
	if len(connections) == 0 {
		return
	}

	service := NewContactSyncService(cfg.CardDAVBlockPrivateURLs)
	for i := range connections {
		conn := &connections[i]
		ctx, cancel := context.WithTimeout(context.Background(), contactSyncRunTimeout)
		stats, err := service.SyncConnection(ctx, db, cfg, conn)
		cancel()
		if err != nil {
			logger.Warn().Err(err).Uint("connection_id", conn.ID).Uint("user_id", conn.UserID).
				Msg("Scheduled contact sync failed")
			continue
		}
		logger.Info().Uint("connection_id", conn.ID).
			Int("pulled_created", stats.PulledCreated).Int("pulled_updated", stats.PulledUpdated).Int("pulled_deleted", stats.PulledDeleted).
			Int("pushed_created", stats.PushedCreated).Int("pushed_updated", stats.PushedUpdated).Int("pushed_deleted", stats.PushedDeleted).
			Int("conflicts", stats.Conflicts).Int("errors", stats.Errors).
			Msg("Scheduled contact sync completed")
	}
}

// SyncContactsWithRateLimit wraps SyncAllContactConnections with the shared
// job lock so rapid restarts or multiple instances don't sync repeatedly.
func SyncContactsWithRateLimit(db *gorm.DB, cfg config.Config) {
	minInterval := time.Duration(cfg.CardDAVSyncIntervalHours)*time.Hour - 30*time.Minute
	if minInterval < 30*time.Minute {
		minInterval = 30 * time.Minute
	}

	acquired, err := acquireJobLock(db, models.JobNameContactSync, minInterval)
	if err != nil {
		logger.Error().Err(err).Msg("Error checking contact sync job lock")
		return
	}
	if !acquired {
		logger.Info().Msg("Skipping contact sync job - rate limited")
		return
	}

	SyncAllContactConnections(db, cfg)

	if err := releaseJobLock(db, models.JobNameContactSync, true); err != nil {
		logger.Error().Err(err).Msg("Error releasing contact sync job lock")
	}
}
