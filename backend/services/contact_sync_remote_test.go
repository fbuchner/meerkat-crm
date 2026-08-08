package services

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	mcarddav "meerkat/carddav"
	"meerkat/models"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	carddavlib "github.com/emersion/go-webdav/carddav"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// The contact sync scenarios run against two real CardDAV servers:
//
//   - "meerkat": Meerkat's own CardDAV server, in-process. It has no
//     sync-collection support, so it exercises the addressbook-query full
//     listing fallback. Always runs.
//   - "radicale": a real Radicale instance, which does support sync-collection.
//     It exercises token mode, explicit deletions and the multi-get path (the
//     sync-collection REPORT returns etags only, so cards must be fetched
//     separately). Opt-in via MEERKAT_CARDDAV_IT=1.
//
// Assertions and fixtures speak CardDAV over HTTP rather than reaching into a
// backing store, so one set of scenarios covers both.

const (
	radicaleEnvVar   = "MEERKAT_CARDDAV_IT"
	radicaleURLVar   = "MEERKAT_CARDDAV_URL"
	radicaleUserVar  = "MEERKAT_CARDDAV_USER"
	radicalePassVar  = "MEERKAT_CARDDAV_PASS"
	defaultRadicale  = "http://localhost:5232"
	defaultRadUser   = "testuser"
	defaultRadPass   = "testpass"
	testRequestMark  = "X-Meerkat-Test"
	radicaleBookRoot = "/testuser/"
)

var radicaleBookCounter atomic.Uint64

// davRemote drives a CardDAV server over HTTP for test setup and assertions.
// Requests it makes itself carry a marker header and are excluded from
// methodCount, so only traffic from the code under test is counted.
type davRemote struct {
	name      string
	proxyURL  string // what the sync connection points at (recorded)
	directURL string // bypasses the recorder, for fixtures and assertions
	path      string
	user      string
	pass      string

	// syncCollection records whether this server implements the
	// sync-collection REPORT, which selects token mode over full listings.
	syncCollection bool

	// preservesRawCards is true when the server stores and returns exactly the
	// vCard it was sent. Meerkat's own server re-derives cards from its data
	// model instead, so it can never serve a card without a REV, and scenarios
	// that need one only apply to servers where this holds.
	preservesRawCards bool

	// setRevision makes the server report a specific revision time for a
	// contact. How that is achieved is backend-specific.
	setRevision func(t *testing.T, uid string, ts time.Time)

	// afterWrite gives a backend a chance to make a just-written change
	// visible. Meerkat derives its ETag from updated_at at second granularity,
	// so two writes in the same second are otherwise indistinguishable.
	afterWrite func(t *testing.T, uid string)

	mu       sync.Mutex
	requests []string

	// fault, when set, rejects matching requests from the code under test, so
	// scenarios can exercise the per-item error paths against a real server.
	fault func(req *http.Request) bool

	// writeSeq keeps successive writes on distinct ETag-visible timestamps.
	writeSeq atomic.Uint64
}

// failRequests makes the server answer 500 to requests the predicate selects.
// Pass nil to recover. Requests the harness itself issues carry the test marker
// and are never affected, so fixtures and assertions keep working while the
// code under test sees a broken server.
func (r *davRemote) failRequests(fn func(req *http.Request) bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fault = fn
}

func (r *davRemote) shouldFail(req *http.Request) bool {
	if req.Header.Get(testRequestMark) != "" {
		return false
	}
	// Copy the hook out before calling it: a predicate is allowed to block (to
	// hold a request open), and running it under the lock would stall every
	// other request and the assertions that read the recorded ones.
	r.mu.Lock()
	fn := r.fault
	r.mu.Unlock()
	return fn != nil && fn(req)
}

func (r *davRemote) record(req *http.Request) {
	if req.Header.Get(testRequestMark) != "" {
		return
	}
	r.mu.Lock()
	r.requests = append(r.requests, req.Method+" "+req.URL.Path)
	r.mu.Unlock()
}

func (r *davRemote) methodCount(method string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, req := range r.requests {
		if strings.HasPrefix(req, method+" ") {
			count++
		}
	}
	return count
}

func (r *davRemote) addressBookPath() string { return r.path }

func (r *davRemote) objectURL(uid string) string {
	return strings.TrimSuffix(r.directURL, "/") + r.path + url.PathEscape(uid) + ".vcf"
}

func (r *davRemote) do(t *testing.T, method, rawURL string, body io.Reader, contentType string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, rawURL, body)
	require.NoError(t, err)
	req.Header.Set(testRequestMark, "1")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if r.user != "" || r.pass != "" {
		req.SetBasicAuth(r.user, r.pass)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// putContact creates or replaces a contact on the server.
func (r *davRemote) putContact(t *testing.T, uid, firstname, lastname, email string) {
	t.Helper()
	lines := []string{
		"BEGIN:VCARD", "VERSION:3.0",
		"UID:" + uid,
		"FN:" + strings.TrimSpace(firstname+" "+lastname),
		fmt.Sprintf("N:%s;%s;;;", lastname, firstname),
	}
	if email != "" {
		lines = append(lines, "EMAIL;TYPE=HOME:"+email)
	}
	lines = append(lines, "END:VCARD", "")
	r.putRaw(t, uid, strings.Join(lines, "\r\n"))
}

// putRaw uploads a verbatim vCard body, for cases that need exact control
// (e.g. an explicit REV).
func (r *davRemote) putRaw(t *testing.T, uid, body string) {
	t.Helper()
	resp := r.do(t, http.MethodPut, r.objectURL(uid), strings.NewReader(body), "text/vcard; charset=utf-8")
	defer resp.Body.Close()
	require.Less(t, resp.StatusCode, 300, "PUT %s returned %d", uid, resp.StatusCode)
	if r.afterWrite != nil {
		r.afterWrite(t, uid)
	}
}

func (r *davRemote) deleteContact(t *testing.T, uid string) {
	t.Helper()
	resp := r.do(t, http.MethodDelete, r.objectURL(uid), nil, "")
	defer resp.Body.Close()
	require.Less(t, resp.StatusCode, 300, "DELETE %s returned %d", uid, resp.StatusCode)
}

// card fetches one contact, reporting whether it exists.
func (r *davRemote) card(t *testing.T, uid string) (vcard.Card, bool) {
	t.Helper()
	resp := r.do(t, http.MethodGet, r.objectURL(uid), nil, "")
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return nil, false
	}
	require.Less(t, resp.StatusCode, 300, "GET %s returned %d", uid, resp.StatusCode)
	card, err := vcard.NewDecoder(resp.Body).Decode()
	require.NoError(t, err)
	return card, true
}

// setRemoteRevision makes the remote copy of a contact claim it was last
// revised at ts, so a test can place it before or after a local edit.
func (r *davRemote) setRemoteRevision(t *testing.T, uid string, ts time.Time) {
	t.Helper()
	require.NotNil(t, r.setRevision, "backend %s cannot set a revision", r.name)
	r.setRevision(t, uid, ts)
}

// revision reads the REV the server reports for a contact, interpreted exactly
// as the sync code interprets it. A card with no usable REV yields the zero
// time, so tests assert on IsZero rather than on a parse error.
func (r *davRemote) revision(t *testing.T, uid string) time.Time {
	t.Helper()
	card, ok := r.card(t, uid)
	require.True(t, ok, "expected remote contact %s to exist", uid)
	rev, _ := mcarddav.ParseRevision(card)
	return rev
}

// lastname reads the family name of a remote contact, failing if absent.
func (r *davRemote) lastname(t *testing.T, uid string) string {
	t.Helper()
	card, ok := r.card(t, uid)
	require.True(t, ok, "expected remote contact %s to exist", uid)
	require.NotNil(t, card.Name(), "remote contact %s has no N property", uid)
	return card.Name().FamilyName
}

type davMultistatus struct {
	XMLName   xml.Name `xml:"DAV: multistatus"`
	Responses []struct {
		Href string `xml:"DAV: href"`
	} `xml:"DAV: response"`
}

// count returns how many address objects the address book holds.
func (r *davRemote) count(t *testing.T) int {
	t.Helper()
	bookURL := strings.TrimSuffix(r.directURL, "/") + r.path
	req, err := http.NewRequest("PROPFIND", bookURL, strings.NewReader(
		`<?xml version="1.0" encoding="utf-8"?><d:propfind xmlns:d="DAV:"><d:prop><d:getetag/></d:prop></d:propfind>`))
	require.NoError(t, err)
	req.Header.Set(testRequestMark, "1")
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Depth", "1")
	if r.user != "" || r.pass != "" {
		req.SetBasicAuth(r.user, r.pass)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var ms davMultistatus
	require.NoError(t, xml.Unmarshal(payload, &ms), "PROPFIND body: %s", payload)
	count := 0
	for _, response := range ms.Responses {
		if strings.HasSuffix(response.Href, ".vcf") {
			count++
		}
	}
	return count
}

// newMeerkatRemote runs Meerkat's own CardDAV server in-process, backed by a
// second database and user.
func newMeerkatRemote(t *testing.T) *davRemote {
	t.Helper()

	db := newContactSyncDB(t)
	user := models.User{Username: "remote", Password: "password123!A", Email: "remote@example.com"}
	require.NoError(t, db.Create(&user).Error)

	remote := &davRemote{
		name: "meerkat",
		path: "/carddav/addressbooks/" + user.Username + "/contacts/",
	}

	handler := &carddavlib.Handler{
		Backend: mcarddav.NewBackend(db, t.TempDir()),
		Prefix:  "/carddav",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		remote.record(req)
		if remote.shouldFail(req) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		ctx := mcarddav.ContextWithUser(req.Context(), user.ID, user.Username, db, "")
		handler.ServeHTTP(w, req.WithContext(ctx))
	}))
	t.Cleanup(srv.Close)

	remote.proxyURL = srv.URL
	remote.directURL = srv.URL

	// Meerkat's ETag is e-<id>-<unix seconds>, so writes landing in the same
	// second are indistinguishable. Push each write onto its own future second
	// — a fixed offset is not enough, since a test can write the same contact
	// twice within one real second and would then reuse the ETag.
	remote.afterWrite = func(t *testing.T, uid string) {
		t.Helper()
		var contact models.Contact
		if err := db.Where("vcard_uid = ?", uid).First(&contact).Error; err != nil {
			return // not all writes land as a contact row (e.g. rejected cards)
		}
		stampMeerkatContact(t, db, contact.ID, time.Now().Add(time.Duration(5+remote.writeSeq.Add(1))*time.Second))
	}

	// Meerkat derives REV from the contact's UpdatedAt, so moving the timestamp
	// moves the reported revision (and the ETag with it).
	remote.setRevision = func(t *testing.T, uid string, ts time.Time) {
		t.Helper()
		var contact models.Contact
		require.NoError(t, db.Where("vcard_uid = ?", uid).First(&contact).Error)
		stampMeerkatContact(t, db, contact.ID, ts)
	}
	return remote
}

// stampMeerkatContact moves a contact's modification time, keeping the ETag
// (e-<id>-<unix seconds>) consistent with it.
func stampMeerkatContact(t *testing.T, db *gorm.DB, contactID uint, ts time.Time) {
	t.Helper()
	require.NoError(t, db.Model(&models.Contact{}).Where("id = ?", contactID).UpdateColumns(map[string]interface{}{
		"updated_at": ts,
		"etag":       fmt.Sprintf("e-%d-%d", contactID, ts.Unix()),
	}).Error)
}

// makeAddressBook creates the remote's address book collection and schedules
// its removal. The body uses the extended MKCOL element from RFC 5689
// (DAV:mkcol); sabre/dav-based servers such as Nextcloud reject anything else
// with 400, where Radicale is lenient about the root element.
func makeAddressBook(t *testing.T, remote *davRemote, collectionPath, displayName, hint string) {
	t.Helper()

	collectionURL := strings.TrimSuffix(remote.directURL, "/") + collectionPath
	body := `<?xml version="1.0" encoding="utf-8"?>` +
		`<D:mkcol xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:carddav"><D:set><D:prop>` +
		`<D:resourcetype><D:collection/><C:addressbook/></D:resourcetype>` +
		`<D:displayname>` + displayName + `</D:displayname>` +
		`</D:prop></D:set></D:mkcol>`

	resp := remote.do(t, "MKCOL", collectionURL, bytes.NewBufferString(body), "application/xml")
	payload, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.Less(t, resp.StatusCode, 300,
		"MKCOL on %s returned %d: %s (%s)", collectionURL, resp.StatusCode, payload, hint)

	t.Cleanup(func() {
		req, err := http.NewRequest(http.MethodDelete, collectionURL, nil)
		if err != nil {
			return
		}
		req.Header.Set(testRequestMark, "1")
		if remote.user != "" || remote.pass != "" {
			req.SetBasicAuth(remote.user, remote.pass)
		}
		if resp, err := http.DefaultClient.Do(req); err == nil {
			resp.Body.Close()
		}
	})
}

// newRadicaleRemote provisions a fresh address book on a real Radicale server
// and points the connection at it through a recording reverse proxy.
func newRadicaleRemote(t *testing.T) *davRemote {
	t.Helper()

	upstreamURL := envOr(radicaleURLVar, defaultRadicale)
	upstream, err := url.Parse(upstreamURL)
	require.NoError(t, err)

	book := fmt.Sprintf("book-%d/", radicaleBookCounter.Add(1))
	remote := &davRemote{
		name:              "radicale",
		directURL:         upstreamURL,
		path:              radicaleBookRoot + book,
		user:              envOr(radicaleUserVar, defaultRadUser),
		pass:              envOr(radicalePassVar, defaultRadPass),
		syncCollection:    true,
		preservesRawCards: true,
	}

	// Radicale stores cards verbatim, so the revision is whatever the card says.
	remote.setRevision = func(t *testing.T, uid string, ts time.Time) {
		t.Helper()
		card, ok := remote.card(t, uid)
		require.True(t, ok, "cannot set revision on missing contact %s", uid)
		card.SetRevision(ts.UTC())
		var buf bytes.Buffer
		require.NoError(t, vcard.NewEncoder(&buf).Encode(card))
		remote.putRaw(t, uid, buf.String())
	}

	proxy := httputil.NewSingleHostReverseProxy(upstream)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		remote.record(req)
		if remote.shouldFail(req) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		proxy.ServeHTTP(w, req)
	}))
	t.Cleanup(srv.Close)
	remote.proxyURL = srv.URL

	makeAddressBook(t, remote, remote.path, book, "is Radicale running? see docker-compose.carddav-test.yml")
	return remote
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

type remoteBackend struct {
	name   string
	create func(t *testing.T) *davRemote
}

func contactSyncBackends() []remoteBackend {
	backends := []remoteBackend{{name: "meerkat", create: newMeerkatRemote}}
	if os.Getenv(radicaleEnvVar) != "" {
		backends = append(backends, remoteBackend{name: "radicale", create: newRadicaleRemote})
	}
	return backends
}

// runContactSync runs a scenario against every available remote backend.
func runContactSync(t *testing.T, direction string, scenario func(t *testing.T, env *contactSyncEnv)) {
	t.Helper()
	for _, backend := range contactSyncBackends() {
		t.Run(backend.name, func(t *testing.T) {
			scenario(t, setupContactSyncEnv(t, direction, backend.create(t)))
		})
	}
}
