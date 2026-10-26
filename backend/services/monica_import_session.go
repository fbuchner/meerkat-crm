package services

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"meerkat/carddav"
	"meerkat/config"
	apperrors "meerkat/errors"
	"meerkat/models"
	"meerkat/monica"

	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

const (
	// monicaSessionExpiry is longer than the CSV/VCF wizard expiry because the
	// rate-limited fetch phase alone can take several minutes. The expiry
	// slides on every status/preview poll.
	monicaSessionExpiry = 60 * time.Minute
	// monicaSessionMaxLifetime bounds how long the sliding expiry can be
	// extended. The session holds the API token in memory, so a wizard left
	// open (or kept alive by a forgotten browser tab) must still age out.
	monicaSessionMaxLifetime = 6 * time.Hour
	// MaxMonicaContacts caps the account size since the full snapshot is held
	// in memory for the duration of the wizard.
	MaxMonicaContacts = 5000
	// MaxMonicaEntities caps the whole snapshot, not just contacts: an account
	// with few contacts but hundreds of thousands of activities or notes would
	// otherwise be fetched unbounded into memory.
	MaxMonicaEntities = 100000
	// monicaConnectTimeout bounds the nine serialized probe requests Connect
	// makes. It must stay below the frontend's connect timeout: a Connect that
	// outlives the browser's wait would store a session holding the API token
	// that the user can neither use nor cancel.
	monicaConnectTimeout = 150 * time.Second
)

// monicaImportSession holds all server-side state of one Monica import wizard
// run, including the API client (and thus the token) — in memory only, wiped
// when the session is deleted or expires.
type monicaImportSession struct {
	id                   string
	userID               uint
	client               *monica.Client
	counts               monica.EntityCounts
	includeRelationships bool
	includeExtras        bool
	collapseReciprocal   bool
	cancel               context.CancelFunc

	mu         sync.Mutex
	phase      string
	phaseDone  int
	phaseTotal int
	errMsg     string
	snapshot   *monica.Snapshot
	mapped     []MonicaMappedContact
	previews   []models.MonicaImportRowPreview
	result     *models.MonicaImportResult
	expiresAt  time.Time
	hardExpiry time.Time
}

func (s *monicaImportSession) setPhase(phase string, done, total int) {
	s.mu.Lock()
	s.phase = phase
	s.phaseDone = done
	s.phaseTotal = total
	s.mu.Unlock()
}

func (s *monicaImportSession) setProgress(done, total int) {
	s.mu.Lock()
	s.phaseDone = done
	s.phaseTotal = total
	s.mu.Unlock()
}

func (s *monicaImportSession) fail(msg string) {
	s.mu.Lock()
	s.phase = models.MonicaPhaseFailed
	s.errMsg = msg
	s.mu.Unlock()
}

// MonicaImportManager owns the lifecycle of Monica import sessions, mirroring
// ImportSessionManager for the file-based imports.
type MonicaImportManager struct {
	mu       sync.RWMutex
	sessions map[string]*monicaImportSession
}

// NewMonicaImportManager creates an empty manager.
func NewMonicaImportManager() *MonicaImportManager {
	return &MonicaImportManager{sessions: make(map[string]*monicaImportSession)}
}

// CleanupExpired removes expired sessions and cancels their running fetches.
// Safe to call from a goroutine.
func (m *MonicaImportManager) CleanupExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for id, s := range m.sessions {
		s.mu.Lock()
		expired := now.After(s.expiresAt)
		s.mu.Unlock()
		if expired {
			if s.cancel != nil {
				s.cancel()
			}
			delete(m.sessions, id)
		}
	}
}

// get retrieves a session, enforcing ownership and sliding expiry.
func (m *MonicaImportManager) get(sessionID string, userID uint) (*monicaImportSession, *apperrors.AppError) {
	m.mu.RLock()
	s, exists := m.sessions[sessionID]
	m.mu.RUnlock()
	if !exists {
		return nil, apperrors.ErrNotFound("Import session expired or not found")
	}
	if s.userID != userID {
		return nil, apperrors.ErrUnauthorized("Session does not belong to current user")
	}
	s.mu.Lock()
	now := time.Now()
	expired := now.After(s.expiresAt) || now.After(s.hardExpiry)
	if !expired {
		s.expiresAt = now.Add(monicaSessionExpiry)
		if s.expiresAt.After(s.hardExpiry) {
			s.expiresAt = s.hardExpiry
		}
	}
	s.mu.Unlock()
	if expired {
		m.Delete(sessionID)
		return nil, apperrors.ErrNotFound("Import session expired")
	}
	return s, nil
}

// Delete removes a session and cancels any running fetch, dropping the API
// token from memory.
func (m *MonicaImportManager) Delete(sessionID string) {
	m.mu.Lock()
	s, exists := m.sessions[sessionID]
	delete(m.sessions, sessionID)
	m.mu.Unlock()
	if exists && s.cancel != nil {
		s.cancel()
	}
}

// userSafeMonicaError converts client errors to messages that never leak the
// token or internal details.
func userSafeMonicaError(err error) string {
	switch {
	case errors.Is(err, monica.ErrUnauthorized):
		return "Monica rejected the API token"
	case errors.Is(err, monica.ErrInvalidURL):
		return "The Monica URL is invalid"
	case errors.Is(err, monica.ErrPrivateAddress):
		return "The Monica URL resolves to a private address, which this server blocks"
	case errors.Is(err, monica.ErrInvalidData):
		return "The server did not respond like a Monica API — check the URL"
	default:
		return "The Monica instance could not be reached"
	}
}

func monicaAppError(err error) *apperrors.AppError {
	switch {
	case errors.Is(err, monica.ErrUnauthorized):
		return apperrors.ErrInvalidInput("api_token", userSafeMonicaError(err))
	case errors.Is(err, monica.ErrInvalidURL), errors.Is(err, monica.ErrPrivateAddress), errors.Is(err, monica.ErrInvalidData):
		return apperrors.ErrInvalidInput("base_url", userSafeMonicaError(err))
	default:
		return apperrors.ErrExternal("monica", userSafeMonicaError(err)).WithError(err)
	}
}

// Connect validates the Monica URL and token, counts the account's entities,
// and creates a session. Runs synchronously (roughly one request per second
// for nine requests).
func (m *MonicaImportManager) Connect(ctx context.Context, userID uint, req models.MonicaConnectRequest, blockPrivate bool) (*models.MonicaConnectResponse, *apperrors.AppError) {
	client, err := monica.NewClient(req.BaseURL, req.APIToken, blockPrivate)
	if err != nil {
		return nil, monicaAppError(err)
	}
	ctx, cancel := context.WithTimeout(ctx, monicaConnectTimeout)
	defer cancel()
	if err := client.TestConnection(ctx); err != nil {
		return nil, monicaAppError(err)
	}
	counts, err := client.CountEntities(ctx)
	if err != nil {
		return nil, monicaAppError(err)
	}
	if counts.Contacts > MaxMonicaContacts {
		return nil, apperrors.ErrInvalidInput("base_url",
			fmt.Sprintf("This Monica account has %d contacts; the import supports at most %d", counts.Contacts, MaxMonicaContacts))
	}
	if total := counts.Total(); total > MaxMonicaEntities {
		return nil, apperrors.ErrInvalidInput("base_url",
			fmt.Sprintf("This Monica account has %d records; the import supports at most %d", total, MaxMonicaEntities))
	}

	sessionID := generateSessionID()
	now := time.Now()
	session := &monicaImportSession{
		id:         sessionID,
		userID:     userID,
		client:     client,
		counts:     counts,
		phase:      models.MonicaPhaseConnecting,
		expiresAt:  now.Add(monicaSessionExpiry),
		hardExpiry: now.Add(monicaSessionMaxLifetime),
	}
	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	return &models.MonicaConnectResponse{
		SessionID:             sessionID,
		Totals:                models.MonicaEntityCounts(counts),
		EstimatedFetchSeconds: estimateFetchSeconds(counts),
	}, nil
}

// estimateFetchSeconds approximates the fetch duration at ~1 request/second:
// one request per 100 records per entity, plus one per contact when
// relationships are included (reported as the worst case).
func estimateFetchSeconds(counts monica.EntityCounts) int {
	pages := 0
	for _, total := range []int{counts.Contacts, counts.Activities, counts.Notes, counts.Reminders, counts.Calls, counts.Tasks, counts.Gifts, counts.Debts} {
		pages += (total + 99) / 100
	}
	return pages + counts.Contacts
}

// StartFetch launches the background snapshot fetch. Returns a conflict error
// if the session is already fetching or done.
func (m *MonicaImportManager) StartFetch(db *gorm.DB, userID uint, req models.MonicaFetchRequest, log *zerolog.Logger) *apperrors.AppError {
	s, appErr := m.get(req.SessionID, userID)
	if appErr != nil {
		return appErr
	}

	s.mu.Lock()
	if s.phase != models.MonicaPhaseConnecting && s.phase != models.MonicaPhaseFailed {
		s.mu.Unlock()
		return apperrors.ErrConflict("A fetch is already running for this session")
	}
	s.phase = models.MonicaPhaseFetchingContacts
	s.phaseDone = 0
	s.phaseTotal = s.counts.Contacts
	s.errMsg = ""
	s.includeRelationships = req.IncludeRelationships
	s.includeExtras = req.IncludeExtras
	s.collapseReciprocal = req.CollapseReciprocalRelationships
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.mu.Unlock()

	go m.runFetch(ctx, db, s, log)
	return nil
}

// runFetch pulls the full account snapshot, then builds the preview.
func (m *MonicaImportManager) runFetch(ctx context.Context, db *gorm.DB, s *monicaImportSession, log *zerolog.Logger) {
	snapshot := &monica.Snapshot{Relationships: map[int][]monica.Relationship{}}
	fail := func(stage string, err error) {
		if ctx.Err() != nil {
			return // session was cancelled or expired; nobody is polling
		}
		log.Warn().Err(err).Str("stage", stage).Str("session_id", s.id).Msg("Monica fetch failed")
		s.fail(userSafeMonicaError(err))
	}

	s.setPhase(models.MonicaPhaseFetchingContacts, 0, s.counts.Contacts)
	allContacts, err := s.client.FetchAllContacts(ctx, s.setProgress)
	if err != nil {
		fail("contacts", err)
		return
	}
	// Partial contacts are name-only stubs Monica creates behind
	// relationships; they become relationship names, not contact rows.
	for _, mc := range allContacts {
		if mc.IsPartial {
			snapshot.PartialContacts = append(snapshot.PartialContacts, mc)
		} else {
			snapshot.Contacts = append(snapshot.Contacts, mc)
		}
	}

	s.setPhase(models.MonicaPhaseFetchingActivities, 0, s.counts.Activities)
	if snapshot.Activities, err = s.client.FetchAllActivities(ctx, s.setProgress); err != nil {
		fail("activities", err)
		return
	}

	s.setPhase(models.MonicaPhaseFetchingNotes, 0, s.counts.Notes)
	if snapshot.Notes, err = s.client.FetchAllNotes(ctx, s.setProgress); err != nil {
		fail("notes", err)
		return
	}

	s.setPhase(models.MonicaPhaseFetchingReminders, 0, s.counts.Reminders)
	if snapshot.Reminders, err = s.client.FetchAllReminders(ctx, s.setProgress); err != nil {
		fail("reminders", err)
		return
	}

	if s.includeExtras {
		extrasTotal := s.counts.Calls + s.counts.Tasks + s.counts.Gifts + s.counts.Debts
		s.setPhase(models.MonicaPhaseFetchingExtras, 0, extrasTotal)
		fetched := 0
		offsetProgress := func(done, total int) { s.setProgress(fetched+done, extrasTotal) }
		if snapshot.Calls, err = s.client.FetchAllCalls(ctx, offsetProgress); err != nil {
			fail("calls", err)
			return
		}
		fetched += len(snapshot.Calls)
		if snapshot.Tasks, err = s.client.FetchAllTasks(ctx, offsetProgress); err != nil {
			fail("tasks", err)
			return
		}
		fetched += len(snapshot.Tasks)
		if snapshot.Gifts, err = s.client.FetchAllGifts(ctx, offsetProgress); err != nil {
			fail("gifts", err)
			return
		}
		fetched += len(snapshot.Gifts)
		if snapshot.Debts, err = s.client.FetchAllDebts(ctx, offsetProgress); err != nil {
			fail("debts", err)
			return
		}
	}

	if s.includeRelationships {
		s.setPhase(models.MonicaPhaseFetchingRelationships, 0, len(snapshot.Contacts))
		for i, mc := range snapshot.Contacts {
			rels, relErr := s.client.FetchContactRelationships(ctx, mc.ID)
			if relErr != nil {
				fail("relationships", relErr)
				return
			}
			if len(rels) > 0 {
				snapshot.Relationships[mc.ID] = rels
			}
			s.setProgress(i+1, len(snapshot.Contacts))
		}
	}

	s.setPhase(models.MonicaPhaseBuildingPreview, 0, len(snapshot.Contacts))
	mapped, previews, previewErr := m.buildPreview(db, s, snapshot)
	if previewErr != nil {
		if ctx.Err() != nil {
			return
		}
		log.Error().Err(previewErr).Str("session_id", s.id).Msg("Monica preview build failed")
		s.fail("The imported data could not be compared against your contacts")
		return
	}

	s.mu.Lock()
	s.snapshot = snapshot
	s.mapped = mapped
	s.previews = previews
	s.phase = models.MonicaPhaseReady
	s.phaseDone = len(previews)
	s.phaseTotal = len(previews)
	s.mu.Unlock()

	log.Info().
		Str("session_id", s.id).
		Int("contacts", len(snapshot.Contacts)).
		Int("activities", len(snapshot.Activities)).
		Int("notes", len(snapshot.Notes)).
		Msg("Monica snapshot fetched")
}

// buildPreview maps contacts and computes validation, duplicate matches, and
// per-contact related-entity counts.
func (m *MonicaImportManager) buildPreview(db *gorm.DB, s *monicaImportSession, snapshot *monica.Snapshot) ([]MonicaMappedContact, []models.MonicaImportRowPreview, error) {
	// One preloaded index instead of a duplicate query (and a full-table
	// phone scan) per imported contact.
	duplicates, err := NewDuplicateIndex(db, s.userID)
	if err != nil {
		return nil, nil, err
	}

	related := map[int]models.MonicaRowRelatedCounts{}
	for _, activity := range snapshot.Activities {
		for _, attendee := range activity.Attendees.Contacts {
			counts := related[attendee.ID]
			counts.Activities++
			related[attendee.ID] = counts
		}
	}
	for _, note := range snapshot.Notes {
		if note.Contact != nil {
			counts := related[note.Contact.ID]
			counts.Notes++
			related[note.Contact.ID] = counts
		}
	}
	// A reminder of someone who is not a contact of their own (a child's
	// birthday) is imported onto the contact they belong to; see
	// importReminders.
	people := monicaPeople(snapshot)
	partialOwners := monicaPartialOwners(snapshot, people)
	birthdays := birthdayMonthDayByMonicaID(people)
	now := time.Now()
	for _, reminder := range snapshot.Reminders {
		if reminder.Contact != nil {
			ownerID := reminder.Contact.ID
			if owners := partialOwners[ownerID]; len(owners) > 0 {
				// Birthday reminders of partial people are left out in
				// favor of the relationship birthday.
				if mapped, _, skip := mapMonicaReminder(reminder, now, birthdays[ownerID]); skip == "" && !*mapped.ReoccurFromCompletion {
					continue
				}
				ownerID = owners[0]
			}
			counts := related[ownerID]
			counts.Reminders++
			related[ownerID] = counts
		}
	}
	// Counted the way the import will actually create them, so the review
	// step does not promise relationships that collapsing will drop. Every
	// contact is assumed imported here; rows the user skips are unknown at
	// preview time, as for the other related-entity counts.
	var edges reciprocalEdgeIndex
	if s.collapseReciprocal {
		edges = buildReciprocalEdgeIndex(snapshot)
	}
	for monicaID, rels := range snapshot.Relationships {
		counts := related[monicaID]
		for _, rel := range rels {
			if rel.OfContact == nil || rel.OfContact.ID == monicaID {
				continue
			}
			if s.collapseReciprocal && edges.isRedundantHalf(monicaID, rel.OfContact.ID) {
				continue
			}
			counts.Relationships++
		}
		related[monicaID] = counts
	}
	addExtra := func(ref *monica.ContactRef) {
		if ref != nil {
			counts := related[ref.ID]
			counts.ExtraNotes++
			related[ref.ID] = counts
		}
	}
	for _, call := range snapshot.Calls {
		addExtra(call.Contact)
	}
	for _, task := range snapshot.Tasks {
		addExtra(task.Contact)
	}
	for _, gift := range snapshot.Gifts {
		addExtra(gift.Contact)
	}
	for _, debt := range snapshot.Debts {
		addExtra(debt.Contact)
	}

	mapped := make([]MonicaMappedContact, 0, len(snapshot.Contacts))
	previews := make([]models.MonicaImportRowPreview, 0, len(snapshot.Contacts))
	for rowIdx, mc := range snapshot.Contacts {
		mappedContact := MapMonicaContact(mc)
		mapped = append(mapped, mappedContact)

		preview := models.MonicaImportRowPreview{
			ImportRowPreview: models.ImportRowPreview{
				RowIndex:         rowIdx,
				ParsedContact:    ContactToPreviewMap(&mappedContact.Contact),
				ValidationErrors: ValidateImportedContact(&mappedContact.Contact),
				SuggestedAction:  "add",
			},
			Related:  related[mc.ID],
			HasPhoto: mappedContact.AvatarURL != "",
		}
		if len(preview.ValidationErrors) > 0 {
			preview.SuggestedAction = "skip"
		} else if duplicate := duplicates.Detect(mappedContact.Contact.Firstname, mappedContact.Contact.Lastname, mappedContact.Contact.Email, mappedContact.Contact.Phone); duplicate != nil {
			preview.DuplicateMatch = duplicate
			preview.SuggestedAction = "update"
		}
		previews = append(previews, preview)
		s.setProgress(rowIdx+1, len(snapshot.Contacts))
	}
	return mapped, previews, nil
}

// Status returns the current phase and progress for polling.
func (m *MonicaImportManager) Status(userID uint, sessionID string) (*models.MonicaImportStatus, *apperrors.AppError) {
	s, appErr := m.get(sessionID, userID)
	if appErr != nil {
		return nil, appErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	status := &models.MonicaImportStatus{
		SessionID:  sessionID,
		Phase:      s.phase,
		PhaseDone:  s.phaseDone,
		PhaseTotal: s.phaseTotal,
		Error:      s.errMsg,
	}
	if s.result != nil {
		resultCopy := *s.result
		// The issue list can run to thousands of rows; send it once, with
		// the final result, instead of on every progress poll.
		if s.phase != models.MonicaPhaseDone {
			resultCopy.Issues = nil
		}
		status.Result = &resultCopy
	}
	return status, nil
}

// Preview returns the full preview once the snapshot is ready.
func (m *MonicaImportManager) Preview(userID uint, sessionID string) (*models.MonicaPreviewResponse, *apperrors.AppError) {
	s, appErr := m.get(sessionID, userID)
	if appErr != nil {
		return nil, appErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.phase != models.MonicaPhaseReady {
		return nil, apperrors.ErrInvalidInput("session", "The Monica data is not fetched yet")
	}

	resp := &models.MonicaPreviewResponse{
		SessionID: sessionID,
		Rows:      s.previews,
		TotalRows: len(s.previews),
	}
	for _, row := range s.previews {
		if len(row.ValidationErrors) > 0 {
			resp.ErrorCount++
		} else {
			resp.ValidRows++
		}
		if row.DuplicateMatch != nil {
			resp.DuplicateCount++
		}
		resp.Totals.Activities += row.Related.Activities
		resp.Totals.Notes += row.Related.Notes
		resp.Totals.Reminders += row.Related.Reminders
		resp.Totals.Relationships += row.Related.Relationships
		resp.Totals.ExtraNotes += row.Related.ExtraNotes
	}
	return resp, nil
}

// avatarTask defers a contact photo download until after the import commits.
type avatarTask struct {
	contactID   uint // Meerkat contact
	contactName string
	monicaID    int // Monica contact, needed to reach the photos endpoint
	avatarURL   string
	source      string // avatarSourcePhoto or avatarSourceGravatar
}

// Confirm executes the import in one transaction: first the contacts pass
// (building the Monica→Meerkat ID map), then all related entities resolved
// through it. Avatars are downloaded asynchronously afterwards; the caller
// polls Status until phase "done".
func (m *MonicaImportManager) Confirm(db *gorm.DB, userID uint, req models.MonicaConfirmRequest, cfg *config.Config, log *zerolog.Logger) (*models.MonicaImportResult, *apperrors.AppError) {
	s, appErr := m.get(req.SessionID, userID)
	if appErr != nil {
		return nil, appErr
	}

	s.mu.Lock()
	if s.phase != models.MonicaPhaseReady {
		s.mu.Unlock()
		return nil, apperrors.ErrInvalidInput("session", "The Monica data is not fetched yet")
	}
	s.phase = models.MonicaPhaseImporting
	snapshot := s.snapshot
	mapped := s.mapped
	previews := s.previews
	s.mu.Unlock()

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		s.setPhase(models.MonicaPhaseReady, 0, 0)
		return nil, apperrors.ErrDatabase("Failed to load user").WithError(err)
	}

	actionMap := buildActionMap(req.Actions)
	result := models.MonicaImportResult{ImportResult: models.ImportResult{Errors: []string{}}, Issues: []models.MonicaImportIssue{}}
	idMap := make(map[int]uint) // Monica contact ID → Meerkat contact ID
	var avatarTasks []avatarTask

	txErr := db.Transaction(func(tx *gorm.DB) error {
		// Pass 1: contacts (same skip/add/update semantics as CSV/VCF import).
		for rowIdx := range previews {
			preview := &previews[rowIdx]
			contactData := mapped[rowIdx]
			action := actionMap[preview.RowIndex]
			if action == "" {
				action = "skip"
			}
			result.TotalProcessed++
			contactName := strings.TrimSpace(contactData.Contact.Firstname + " " + contactData.Contact.Lastname)
			contactIssue := func(status, reason string, cause error) {
				addMonicaFailure(&result, status, "contact", contactName, "", reason, cause)
			}

			switch action {
			case "skip":
				result.Skipped++
				if len(preview.ValidationErrors) > 0 {
					addMonicaIssue(&result, models.MonicaIssueSkipped, "contact", contactName, "", models.MonicaReasonInvalid, strings.Join(preview.ValidationErrors, "; "))
				} else {
					contactIssue(models.MonicaIssueSkipped, models.MonicaReasonNotSelected, nil)
				}

			case "add":
				contact := contactData.Contact
				contact.UserID = userID
				if err := tx.Create(&contact).Error; err != nil {
					contactIssue(models.MonicaIssueFailed, models.MonicaReasonError, err)
					result.Skipped++
					continue
				}
				result.Created++
				idMap[contactData.MonicaID] = contact.ID
				if contactData.AvatarURL != "" {
					avatarTasks = append(avatarTasks, avatarTask{contactID: contact.ID, contactName: contactName, monicaID: contactData.MonicaID,
						avatarURL: contactData.AvatarURL, source: contactData.AvatarSource})
				}

			case "update":
				if preview.DuplicateMatch == nil {
					contactIssue(models.MonicaIssueFailed, models.MonicaReasonNoMatch, nil)
					result.Skipped++
					continue
				}
				var existing models.Contact
				if err := tx.Where("user_id = ?", userID).First(&existing, preview.DuplicateMatch.ExistingContactID).Error; err != nil {
					contactIssue(models.MonicaIssueFailed, models.MonicaReasonNoMatch, err)
					result.Skipped++
					continue
				}
				if err := CreateMergeNote(tx, userID, existing.ID, &existing, preview.ParsedContact, "Monica"); err != nil {
					log.Warn().Err(err).Uint("contact_id", existing.ID).Msg("Failed to create merge note")
				}
				incoming := contactData.Contact
				MergeImportedContact(&existing, &incoming)
				// Custom fields are additive: never overwrite values the user
				// already maintains in Meerkat.
				for key, value := range incoming.CustomFields {
					if existing.CustomFields == nil {
						existing.CustomFields = map[string]string{}
					}
					if _, taken := existing.CustomFields[key]; !taken {
						existing.CustomFields[key] = value
					}
				}
				if err := tx.Save(&existing).Error; err != nil {
					contactIssue(models.MonicaIssueFailed, models.MonicaReasonError, err)
					result.Skipped++
					continue
				}
				result.Updated++
				idMap[contactData.MonicaID] = existing.ID
				if existing.Photo == "" && contactData.AvatarURL != "" {
					avatarTasks = append(avatarTasks, avatarTask{contactID: existing.ID, contactName: contactName, monicaID: contactData.MonicaID,
						avatarURL: contactData.AvatarURL, source: contactData.AvatarSource})
				}
			}
		}

		// Pass 2: related entities, linked through idMap. Row-level failures
		// are recorded in result.Errors and do not abort the import; only a
		// failure to load the dedupe keys does, since importing without them
		// would silently duplicate everything.
		if err := m.importRelationships(tx, userID, snapshot, idMap, s.collapseReciprocal, &result); err != nil {
			return err
		}
		if err := m.importActivities(tx, userID, user.Language, snapshot, idMap, &result); err != nil {
			return err
		}
		noteKeys, err := loadNoteKeys(tx, userID)
		if err != nil {
			return err
		}
		m.importNotes(tx, userID, snapshot, idMap, noteKeys, &result)
		if err := m.importReminders(tx, userID, snapshot, idMap, &result); err != nil {
			return err
		}
		m.importExtraNotes(tx, userID, user.Language, snapshot, idMap, noteKeys, &result)
		result.Issues = rollUpMonicaIssues(result.Issues)

		// Make the custom fields created by the import visible in the UI.
		// This is cosmetic: a failure here must not roll back thousands of
		// successfully imported records.
		if err := mergeMonicaCustomFieldNames(tx, &user, mapped, idMap); err != nil {
			log.Warn().Err(err).Uint("user_id", userID).Msg("Failed to merge Monica custom field names")
			result.Errors = append(result.Errors, "Imported custom fields could not be added to your custom field list")
		}
		return nil
	})
	if txErr != nil {
		s.setPhase(models.MonicaPhaseReady, 0, 0)
		return nil, apperrors.ErrDatabase("Import failed").WithError(txErr)
	}

	result.PhotosQueued = len(avatarTasks)
	s.mu.Lock()
	resultCopy := result
	s.result = &resultCopy
	s.snapshot = nil // free the snapshot; only photo work remains
	s.mapped = nil
	s.previews = nil
	if len(avatarTasks) == 0 {
		s.phase = models.MonicaPhaseDone
	} else {
		s.phase = models.MonicaPhaseImportingPhotos
		s.phaseDone = 0
		s.phaseTotal = len(avatarTasks)
	}
	s.mu.Unlock()

	failed := 0
	for _, issue := range result.Issues {
		if issue.Status == models.MonicaIssueFailed {
			failed++
			log.Warn().Str("session_id", req.SessionID).Str("entity", issue.Entity).Str("reason", issue.Reason).
				Str("error", issue.Cause).Msg("Monica record not imported")
		}
	}
	log.Info().
		Str("session_id", req.SessionID).
		Int("created", result.Created).
		Int("updated", result.Updated).
		Int("skipped", result.Skipped).
		Int("activities", result.ActivitiesCreated).
		Int("notes", result.NotesCreated).
		Int("reminders", result.RemindersCreated).
		Int("relationships", result.RelationshipsCreated).
		Int("photos_queued", result.PhotosQueued).
		Int("errors", len(result.Errors)).
		Int("issues", len(result.Issues)).
		Int("failed", failed).
		Msg("Monica import completed")

	if len(avatarTasks) > 0 {
		go m.processAvatars(db, s, avatarTasks, cfg, log)
	}

	return &result, nil
}

// processAvatars downloads and stores contact photos after the transaction,
// reporting progress through the session for status polling.
func (m *MonicaImportManager) processAvatars(db *gorm.DB, s *monicaImportSession, tasks []avatarTask, cfg *config.Config, log *zerolog.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(len(tasks)+60)*time.Second*2)
	defer cancel()

	for i, task := range tasks {
		saved := false
		// Photos hosted by Monica are only reachable through the API; every
		// other source (gravatar) is a public URL fetched directly.
		var photoData []byte
		var mediaType string
		var err error
		if task.source == avatarSourcePhoto {
			photoData, mediaType, err = s.client.FetchContactPhoto(ctx, task.monicaID, task.avatarURL)
		} else {
			photoData, mediaType, err = s.client.FetchAvatar(ctx, task.avatarURL)
		}
		// Only fetch errors are shown: storage errors can carry server paths.
		failure := ""
		if err != nil {
			failure = userSafeMonicaError(err)
			log.Warn().Err(err).Uint("contact_id", task.contactID).Str("source", task.source).Msg("Failed to fetch Monica avatar")
		} else {
			photoPath, thumbnailData, saveErr := carddav.SaveContactPhoto(photoData, mediaType, cfg.ProfilePhotoDir)
			if saveErr != nil {
				log.Warn().Err(saveErr).Uint("contact_id", task.contactID).Msg("Failed to save Monica avatar")
			} else if updateErr := db.Model(&models.Contact{}).Where("id = ?", task.contactID).Updates(map[string]interface{}{
				"photo":           photoPath,
				"photo_thumbnail": thumbnailData,
			}).Error; updateErr != nil {
				log.Warn().Err(updateErr).Uint("contact_id", task.contactID).Msg("Failed to update contact with avatar")
			} else {
				saved = true
			}
		}

		s.mu.Lock()
		if s.result != nil {
			if saved {
				s.result.PhotosSaved++
			} else {
				s.result.PhotosFailed++
				addMonicaIssue(s.result, models.MonicaIssueFailed, "photo", task.contactName, "", models.MonicaReasonPhotoFailed, failure)
			}
		}
		s.phaseDone = i + 1
		s.mu.Unlock()
	}

	s.mu.Lock()
	s.phase = models.MonicaPhaseDone
	s.mu.Unlock()
}

// importKeySet holds the dedupe keys of one entity type, preloaded once per
// import. Probing the database per row instead would serialize tens of
// thousands of statements inside the single write transaction, holding
// SQLite's write lock — and blocking the UI and CardDAV — for minutes.
type importKeySet map[string]bool

// addIfNew reports whether the key was absent, recording it if so.
func (s importKeySet) addIfNew(key string) bool {
	if s[key] {
		return false
	}
	s[key] = true
	return true
}

// timeKey renders a timestamp for a dedupe key, normalizing the location so
// a value read back from SQLite matches a freshly mapped one.
func timeKey(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

// textKey hashes free text so long note bodies do not sit in memory twice.
func textKey(text string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(text)))
}

// reciprocalEdgeIndex records which directed relationship edges the snapshot
// contains, so the importer can tell one half of a Monica pair from a
// genuinely one-sided relationship.
type reciprocalEdgeIndex map[[2]int]bool

func buildReciprocalEdgeIndex(snapshot *monica.Snapshot) reciprocalEdgeIndex {
	idx := reciprocalEdgeIndex{}
	for ownerID, rels := range snapshot.Relationships {
		for _, rel := range rels {
			if rel.OfContact != nil {
				idx[[2]int{ownerID, rel.OfContact.ID}] = true
			}
		}
	}
	return idx
}

// isRedundantHalf reports whether the edge owner→other is the half of a
// bidirectional Monica pair that can be dropped.
//
// Monica writes every relationship to both contacts; Meerkat's is directed
// and already renders on the other contact as an incoming relationship, so
// importing both halves shows each relationship twice. The surviving half is
// the one owned by the lower Monica contact id — an arbitrary but stable
// choice, which matters because snapshot.Relationships is a map and its
// iteration order changes between runs.
//
// The reciprocal edge must actually exist: Monica data with only one
// direction recorded is one-sided, and dropping it would lose it entirely.
func (idx reciprocalEdgeIndex) isRedundantHalf(owner, other int) bool {
	return other < owner && idx[[2]int{other, owner}]
}

func relationshipKey(contactID uint, name, relType string) string {
	return fmt.Sprintf("%d\x1f%s\x1f%s", contactID, name, relType)
}

func loadRelationshipKeys(tx *gorm.DB, userID uint) (importKeySet, error) {
	var rows []struct {
		ContactID uint
		Name      string
		Type      string
	}
	if err := tx.Model(&models.Relationship{}).Where("user_id = ?", userID).
		Select("contact_id", "name", "type").Scan(&rows).Error; err != nil {
		return nil, err
	}
	keys := make(importKeySet, len(rows))
	for _, r := range rows {
		keys[relationshipKey(r.ContactID, r.Name, r.Type)] = true
	}
	return keys, nil
}

// importRelationships creates relationship rows for imported contacts,
// deduplicated within the batch and against existing rows.
func (m *MonicaImportManager) importRelationships(tx *gorm.DB, userID uint, snapshot *monica.Snapshot, idMap map[int]uint, collapseReciprocal bool, result *models.MonicaImportResult) error {
	seen, err := loadRelationshipKeys(tx, userID)
	if err != nil {
		return err
	}

	// Includes partial contacts and people only known from the relationship
	// itself: a child that is not a contact of its own still has a gender
	// and birthday in Monica, which belong on the relationship row.
	people := monicaPeople(snapshot)

	var edges reciprocalEdgeIndex
	if collapseReciprocal {
		edges = buildReciprocalEdgeIndex(snapshot)
	}

	for monicaID, rels := range snapshot.Relationships {
		ownerName := monicaPersonName(people[monicaID])
		contactID, imported := idMap[monicaID]
		for _, rel := range rels {
			// Monica should only return edges where the fetched contact is
			// the subject; guard against linking a contact to itself.
			if rel.OfContact != nil && rel.OfContact.ID == monicaID {
				continue
			}
			var related *monica.Contact
			if rel.OfContact != nil {
				related = people[rel.OfContact.ID]
			}
			relationship, otherMonicaID, ok := MapMonicaRelationship(rel, related)
			if !ok {
				addMonicaIssue(result, models.MonicaIssueSkipped, "relationship", ownerName, strings.TrimSpace(rel.RelationshipType.Name), models.MonicaReasonIncomplete, "")
				continue
			}
			label := relationship.Name + " (" + relationship.Type + ")"
			if !imported {
				addMonicaIssue(result, models.MonicaIssueSkipped, "relationship", ownerName, label, models.MonicaReasonContactNotImported, "")
				continue
			}
			// Only collapse when the other contact is imported too: without
			// it there is no linked contact to surface the incoming side, so
			// dropping this half would lose the relationship altogether.
			if collapseReciprocal && edges.isRedundantHalf(monicaID, otherMonicaID) {
				if _, otherImported := idMap[otherMonicaID]; otherImported {
					result.RelationshipsCollapsed++
					continue
				}
			}
			if !seen.addIfNew(relationshipKey(contactID, relationship.Name, relationship.Type)) {
				addMonicaIssue(result, models.MonicaIssueSkipped, "relationship", ownerName, label, models.MonicaReasonAlreadyExists, "")
				continue
			}

			relationship.UserID = userID
			relationship.ContactID = contactID
			if relatedID, linked := idMap[otherMonicaID]; linked {
				relationship.RelatedContactID = &relatedID
			}
			if err := tx.Create(&relationship).Error; err != nil {
				addMonicaFailure(result, models.MonicaIssueFailed, "relationship", ownerName, label, models.MonicaReasonError, err)
				continue
			}
			result.RelationshipsCreated++
			// A linked contact carries its own birthday; only an unlinked
			// person loses an age-only birthday here.
			if relationship.RelatedContactID == nil && related != nil && related.Information.Dates.Birthdate.IsAgeBased {
				addMonicaIssue(result, models.MonicaIssueSkipped, "birthday", ownerName, relationship.Name, models.MonicaReasonApproximateBirthday, "")
			}
		}
	}
	return nil
}

// activityKey identifies an activity for dedupe purposes. Title and date
// alone are far too coarse: two "Lunch" entries on the same day with
// different attendees are distinct activities, and an unrelated pre-existing
// "Meeting" (from calendar sync, say) must not block Monica's "Meeting" on
// the same date. Description and the attendee set complete the identity.
func activityKey(title string, date time.Time, description string, contactIDs []uint) string {
	ids := append([]uint(nil), contactIDs...)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\x1f")
	b.WriteString(timeKey(date))
	b.WriteString("\x1f")
	b.WriteString(textKey(description))
	for _, id := range ids {
		b.WriteString("\x1f")
		b.WriteString(strconv.FormatUint(uint64(id), 10))
	}
	return b.String()
}

func loadActivityKeys(tx *gorm.DB, userID uint) (importKeySet, error) {
	var rows []struct {
		ID          uint
		Title       string
		Description string
		Date        time.Time
	}
	if err := tx.Model(&models.Activity{}).Where("user_id = ?", userID).
		Select("id", "title", "description", "date").Scan(&rows).Error; err != nil {
		return nil, err
	}
	keys := make(importKeySet, len(rows))
	if len(rows) == 0 {
		return keys, nil
	}

	// Joined rather than filtered by an IN list of activity ids, which would
	// blow past SQLite's bound-variable limit on large accounts.
	var links []struct {
		ActivityID uint
		ContactID  uint
	}
	if err := tx.Table("activity_contacts").
		Joins("JOIN activities ON activities.id = activity_contacts.activity_id").
		Where("activities.user_id = ? AND activities.deleted_at IS NULL", userID).
		Select("activity_contacts.activity_id", "activity_contacts.contact_id").
		Scan(&links).Error; err != nil {
		return nil, err
	}
	attendees := make(map[uint][]uint, len(rows))
	for _, l := range links {
		attendees[l.ActivityID] = append(attendees[l.ActivityID], l.ContactID)
	}
	for _, r := range rows {
		keys[activityKey(r.Title, r.Date, r.Description, attendees[r.ID])] = true
	}
	return keys, nil
}

// importActivities creates activities whose attendees resolved to at least
// one imported contact and links them through the join table.
func (m *MonicaImportManager) importActivities(tx *gorm.DB, userID uint, lang string, snapshot *monica.Snapshot, idMap map[int]uint, result *models.MonicaImportResult) error {
	seen, err := loadActivityKeys(tx, userID)
	if err != nil {
		return err
	}

	for _, ma := range snapshot.Activities {
		attendeeNames := make([]string, 0, len(ma.Attendees.Contacts))
		for _, attendee := range ma.Attendees.Contacts {
			if name := monicaRefName(&attendee); name != "" {
				attendeeNames = append(attendeeNames, name)
			}
		}
		attendees := strings.Join(attendeeNames, ", ")
		activity, attendeeMonicaIDs, ok := MapMonicaActivity(ma, lang)
		if !ok {
			addMonicaIssue(result, models.MonicaIssueSkipped, "activity", attendees, monicaExcerpt(ma.Summary), models.MonicaReasonIncomplete, "")
			continue
		}
		var contactIDs []uint
		for _, monicaID := range attendeeMonicaIDs {
			if id, imported := idMap[monicaID]; imported {
				contactIDs = append(contactIDs, id)
			}
		}
		if len(contactIDs) == 0 {
			addMonicaIssue(result, models.MonicaIssueSkipped, "activity", attendees, activity.Title, models.MonicaReasonContactNotImported, "")
			continue
		}

		if !seen.addIfNew(activityKey(activity.Title, activity.Date, activity.Description, contactIDs)) {
			addMonicaIssue(result, models.MonicaIssueSkipped, "activity", attendees, activity.Title, models.MonicaReasonAlreadyExists, "")
			continue
		}

		activity.UserID = userID
		if err := tx.Create(&activity).Error; err != nil {
			addMonicaFailure(result, models.MonicaIssueFailed, "activity", attendees, activity.Title, models.MonicaReasonError, err)
			continue
		}
		var contacts []models.Contact
		if err := tx.Where("user_id = ? AND id IN ?", userID, contactIDs).Find(&contacts).Error; err == nil && len(contacts) > 0 {
			if err := tx.Model(&activity).Association("Contacts").Append(&contacts); err != nil {
				addMonicaFailure(result, models.MonicaIssueFailed, "activity", attendees, activity.Title, models.MonicaReasonError, fmt.Errorf("failed to link contacts: %w", err))
			}
		}
		result.ActivitiesCreated++
	}
	return nil
}

func noteKey(contactID uint, content string, date time.Time) string {
	return fmt.Sprintf("%d\x1f%s\x1f%s", contactID, textKey(content), timeKey(date))
}

func loadNoteKeys(tx *gorm.DB, userID uint) (importKeySet, error) {
	var rows []struct {
		ContactID *uint
		Content   string
		Date      time.Time
	}
	if err := tx.Model(&models.Note{}).Where("user_id = ?", userID).
		Select("contact_id", "content", "date").Scan(&rows).Error; err != nil {
		return nil, err
	}
	keys := make(importKeySet, len(rows))
	for _, r := range rows {
		if r.ContactID == nil {
			continue // standalone note; imported notes always have a contact
		}
		keys[noteKey(*r.ContactID, r.Content, r.Date)] = true
	}
	return keys, nil
}

func (m *MonicaImportManager) createImportedNote(tx *gorm.DB, userID uint, entity, contactName string, note models.Note, contactID uint, seen importKeySet, result *models.MonicaImportResult) {
	if !seen.addIfNew(noteKey(contactID, note.Content, note.Date)) {
		addMonicaIssue(result, models.MonicaIssueSkipped, entity, contactName, monicaExcerpt(note.Content), models.MonicaReasonAlreadyExists, "")
		return
	}
	note.UserID = userID
	note.ContactID = &contactID
	if err := tx.Create(&note).Error; err != nil {
		addMonicaFailure(result, models.MonicaIssueFailed, entity, contactName, monicaExcerpt(note.Content), models.MonicaReasonError, err)
		return
	}
	result.NotesCreated++
}

func (m *MonicaImportManager) importNotes(tx *gorm.DB, userID uint, snapshot *monica.Snapshot, idMap map[int]uint, seen importKeySet, result *models.MonicaImportResult) {
	for _, mn := range snapshot.Notes {
		m.importNoteLike(tx, userID, "note", mn.Contact, monicaExcerpt(mn.Body), idMap, seen, result, func() (models.Note, int, bool) {
			return MapMonicaNote(mn)
		})
	}
}

// importNoteLike creates one note (or call/task/gift/debt converted into a
// note), recording why it was left out when it is.
func (m *MonicaImportManager) importNoteLike(tx *gorm.DB, userID uint, entity string, ref *monica.ContactRef, label string, idMap map[int]uint, seen importKeySet, result *models.MonicaImportResult, mapFn func() (models.Note, int, bool)) {
	contactName := monicaRefName(ref)
	note, monicaContactID, ok := mapFn()
	if !ok {
		addMonicaIssue(result, models.MonicaIssueSkipped, entity, contactName, label, models.MonicaReasonIncomplete, "")
		return
	}
	contactID, imported := idMap[monicaContactID]
	if !imported {
		addMonicaIssue(result, models.MonicaIssueSkipped, entity, contactName, label, models.MonicaReasonContactNotImported, "")
		return
	}
	m.createImportedNote(tx, userID, entity, contactName, note, contactID, seen, result)
}

func reminderKey(contactID uint, message string, remindAt time.Time) string {
	return fmt.Sprintf("%d\x1f%s\x1f%s", contactID, message, timeKey(remindAt))
}

func loadReminderKeys(tx *gorm.DB, userID uint) (importKeySet, error) {
	var rows []struct {
		ContactID *uint
		Message   string
		RemindAt  time.Time
	}
	if err := tx.Model(&models.Reminder{}).Where("user_id = ?", userID).
		Select("contact_id", "message", "remind_at").Scan(&rows).Error; err != nil {
		return nil, err
	}
	keys := make(importKeySet, len(rows))
	for _, r := range rows {
		if r.ContactID == nil {
			continue
		}
		keys[reminderKey(*r.ContactID, r.Message, r.RemindAt)] = true
	}
	return keys, nil
}

func (m *MonicaImportManager) importReminders(tx *gorm.DB, userID uint, snapshot *monica.Snapshot, idMap map[int]uint, result *models.MonicaImportResult) error {
	seen, err := loadReminderKeys(tx, userID)
	if err != nil {
		return err
	}
	people := monicaPeople(snapshot)
	birthdays := birthdayMonthDayByMonicaID(people)
	partialOwners := monicaPartialOwners(snapshot, people)
	now := time.Now()
	for _, mr := range snapshot.Reminders {
		birthdayMonthDay, personName := "", monicaRefName(mr.Contact)
		if mr.Contact != nil {
			birthdayMonthDay = birthdays[mr.Contact.ID]
			if name := monicaPersonName(people[mr.Contact.ID]); name != "" {
				personName = name
			}
		}
		label := monicaExcerpt(mr.Title)
		reminder, monicaContactID, skipReason := mapMonicaReminder(mr, now, birthdayMonthDay)
		if skipReason != "" {
			addMonicaIssue(result, models.MonicaIssueSkipped, "reminder", personName, label, skipReason, "")
			continue
		}
		contactID, imported := idMap[monicaContactID]
		movedToOwner := false
		if !imported {
			// A reminder of someone who is not a contact of their own (a
			// child, typically) goes to the contact they belong to.
			for _, ownerID := range partialOwners[monicaContactID] {
				if ownerContactID, ownerImported := idMap[ownerID]; ownerImported {
					contactID, imported, movedToOwner = ownerContactID, true, true
					break
				}
			}
		}
		// MapMonicaReminder pins exactly the birthday reminders to the
		// calendar. A partial person's birthday is imported onto the
		// relationship, which Meerkat already announces; a reminder on top
		// would announce it twice. A child with only an approximate age has
		// no birthday to match, so its reminder is not recognized as one and
		// still moves to the parent — the only place it can survive.
		if movedToOwner && !*reminder.ReoccurFromCompletion {
			addMonicaIssue(result, models.MonicaIssueSkipped, "reminder", personName, label, models.MonicaReasonBirthdayOnRelationship, "")
			continue
		}
		if movedToOwner {
			reminder.Message = monicaMessageAbout(reminder.Message, people[monicaContactID])
		}
		if !imported {
			addMonicaIssue(result, models.MonicaIssueSkipped, "reminder", personName, label, models.MonicaReasonContactNotImported, "")
			continue
		}
		if !seen.addIfNew(reminderKey(contactID, reminder.Message, reminder.RemindAt)) {
			addMonicaIssue(result, models.MonicaIssueSkipped, "reminder", personName, label, models.MonicaReasonAlreadyExists, "")
			continue
		}
		reminder.UserID = userID
		reminder.ContactID = &contactID
		if err := tx.Create(&reminder).Error; err != nil {
			addMonicaFailure(result, models.MonicaIssueFailed, "reminder", personName, label, models.MonicaReasonError, err)
			continue
		}
		result.RemindersCreated++
	}
	return nil
}

// monicaMessageAbout makes a reminder moved onto another contact name the
// person it is about, unless its text already does (Monica's generated
// birthday reminders do).
func monicaMessageAbout(message string, person *monica.Contact) string {
	name := monicaPersonName(person)
	if name == "" {
		return message
	}
	if first := strings.TrimSpace(person.FirstName); first != "" && containsWords(message, first) {
		return message
	}
	return truncateRunes(name+": "+message, 500)
}

// containsWords reports whether every word of phrase appears as a whole word
// in text, ignoring case — so "Al" does not match "Call".
func containsWords(text, phrase string) bool {
	split := func(s string) []string {
		return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		})
	}
	words := split(text)
	for _, want := range split(phrase) {
		if !slices.Contains(words, want) {
			return false
		}
	}
	return true
}

// importExtraNotes converts calls, tasks, gifts and debts into dated notes.
func (m *MonicaImportManager) importExtraNotes(tx *gorm.DB, userID uint, lang string, snapshot *monica.Snapshot, idMap map[int]uint, seen importKeySet, result *models.MonicaImportResult) {
	for _, call := range snapshot.Calls {
		m.importNoteLike(tx, userID, "call", call.Contact, monicaExcerpt(call.Content), idMap, seen, result, func() (models.Note, int, bool) {
			return MapMonicaCall(call, lang)
		})
	}
	for _, task := range snapshot.Tasks {
		m.importNoteLike(tx, userID, "task", task.Contact, monicaExcerpt(task.Title), idMap, seen, result, func() (models.Note, int, bool) {
			return MapMonicaTask(task, lang)
		})
	}
	for _, gift := range snapshot.Gifts {
		m.importNoteLike(tx, userID, "gift", gift.Contact, monicaExcerpt(gift.Name), idMap, seen, result, func() (models.Note, int, bool) {
			return MapMonicaGift(gift, lang)
		})
	}
	for _, debt := range snapshot.Debts {
		label := strings.TrimSpace(debt.AmountWithCurrency)
		if label == "" {
			label = monicaExcerpt(debt.Reason)
		}
		m.importNoteLike(tx, userID, "debt", debt.Contact, label, idMap, seen, result, func() (models.Note, int, bool) {
			return MapMonicaDebt(debt, lang)
		})
	}
}

// addMonicaIssue records one record that did not make it into Meerkat (or
// made it without part of its data) for the import report.
func addMonicaIssue(result *models.MonicaImportResult, status, entity, contactName, label, reason, detail string) {
	result.Issues = append(result.Issues, models.MonicaImportIssue{
		Status:  status,
		Entity:  entity,
		Contact: contactName,
		Label:   label,
		Reason:  reason,
		Detail:  detail,
	})
}

// addMonicaFailure records a record the database (or a lookup) rejected. The
// raw error stays server-side — it is logged, never sent — since driver
// errors can carry SQL and schema details; the reason code says enough.
func addMonicaFailure(result *models.MonicaImportResult, status, entity, contactName, label, reason string, cause error) {
	addMonicaIssue(result, status, entity, contactName, label, reason, "")
	if cause != nil {
		result.Issues[len(result.Issues)-1].Cause = cause.Error()
	}
}

// rollUpMonicaIssues folds the records of a contact that was not imported
// into one row per contact and record type: the contact's own row already
// says why, and listing each of its notes one by one buries the rest.
func rollUpMonicaIssues(issues []models.MonicaImportIssue) []models.MonicaImportIssue {
	type groupKey struct{ entity, contact string }
	index := map[groupKey]int{}
	out := make([]models.MonicaImportIssue, 0, len(issues))
	for _, issue := range issues {
		if issue.Reason != models.MonicaReasonContactNotImported {
			out = append(out, issue)
			continue
		}
		key := groupKey{issue.Entity, issue.Contact}
		if i, seen := index[key]; seen {
			out[i].Count++
			out[i].Label = ""
			continue
		}
		issue.Count = 1
		index[key] = len(out)
		out = append(out, issue)
	}
	return out
}

// monicaExcerpt shortens free text to a single line fit for the report.
func monicaExcerpt(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if len([]rune(text)) > 80 {
		return truncateRunes(text, 79) + "…"
	}
	return text
}

// mergeMonicaCustomFieldNames adds custom field names used by imported
// contacts to the user's field list so they render in the contact UI.
func mergeMonicaCustomFieldNames(tx *gorm.DB, user *models.User, mapped []MonicaMappedContact, idMap map[int]uint) error {
	existing := map[string]bool{}
	for _, name := range user.CustomFieldNames {
		existing[name] = true
	}
	added := false
	for _, contactData := range mapped {
		if _, imported := idMap[contactData.MonicaID]; !imported {
			continue
		}
		for key := range contactData.Contact.CustomFields {
			if !existing[key] {
				user.CustomFieldNames = append(user.CustomFieldNames, key)
				existing[key] = true
				added = true
			}
		}
	}
	if !added {
		return nil
	}
	// Struct-based update so the JSON serializer on CustomFieldNames applies.
	return tx.Model(user).Select("CustomFieldNames").Updates(user).Error
}
