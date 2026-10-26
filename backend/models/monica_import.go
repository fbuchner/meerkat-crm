package models

// MonicaEntityCounts mirrors the per-entity totals of a Monica account
// (kept separate from the monica package so models stays dependency-free).
type MonicaEntityCounts struct {
	Contacts   int `json:"contacts"`
	Activities int `json:"activities"`
	Notes      int `json:"notes"`
	Reminders  int `json:"reminders"`
	Calls      int `json:"calls"`
	Tasks      int `json:"tasks"`
	Gifts      int `json:"gifts"`
	Debts      int `json:"debts"`
}

// MonicaConnectRequest starts a Monica import by validating the instance
// URL and API token. The token is held in memory only and never persisted.
type MonicaConnectRequest struct {
	BaseURL  string `json:"base_url" validate:"required,max=500"`
	APIToken string `json:"api_token" validate:"required,max=4000"`
}

// MonicaConnectResponse reports what the Monica account contains.
type MonicaConnectResponse struct {
	SessionID             string             `json:"session_id"`
	Totals                MonicaEntityCounts `json:"totals"`
	EstimatedFetchSeconds int                `json:"estimated_fetch_seconds"`
}

// MonicaFetchRequest starts the background snapshot fetch.
type MonicaFetchRequest struct {
	SessionID string `json:"session_id" validate:"required"`
	// IncludeRelationships requires one extra API request per contact
	// (Monica has no account-wide relationships endpoint), so it is opt-in.
	IncludeRelationships bool `json:"include_relationships"`
	// IncludeExtras imports calls, tasks, gifts and debts as notes.
	IncludeExtras bool `json:"include_extras"`
	// CollapseReciprocalRelationships imports only one row per relationship
	// pair. Monica stores every relationship on both contacts, whereas a
	// Meerkat relationship is directed and already surfaces on the other
	// contact as an incoming relationship — so importing both halves shows
	// the same relationship twice on each contact. Off keeps both halves.
	CollapseReciprocalRelationships bool `json:"collapse_reciprocal_relationships"`
}

// Monica import phases, in order of progression.
const (
	MonicaPhaseConnecting            = "connecting"
	MonicaPhaseFetchingContacts      = "fetching_contacts"
	MonicaPhaseFetchingActivities    = "fetching_activities"
	MonicaPhaseFetchingNotes         = "fetching_notes"
	MonicaPhaseFetchingReminders     = "fetching_reminders"
	MonicaPhaseFetchingExtras        = "fetching_extras"
	MonicaPhaseFetchingRelationships = "fetching_relationships"
	MonicaPhaseBuildingPreview       = "building_preview"
	MonicaPhaseReady                 = "ready"
	MonicaPhaseImporting             = "importing"
	MonicaPhaseImportingPhotos       = "importing_photos"
	MonicaPhaseDone                  = "done"
	MonicaPhaseFailed                = "failed"
)

// MonicaImportStatus is the polling payload for fetch and import progress.
type MonicaImportStatus struct {
	SessionID  string              `json:"session_id"`
	Phase      string              `json:"phase"`
	PhaseDone  int                 `json:"phase_done"`
	PhaseTotal int                 `json:"phase_total"`
	Error      string              `json:"error,omitempty"`
	Result     *MonicaImportResult `json:"result,omitempty"` // set from phase "importing_photos" onwards
}

// MonicaRowRelatedCounts summarizes the related entities that would be
// imported alongside one contact.
type MonicaRowRelatedCounts struct {
	Activities    int `json:"activities"`
	Notes         int `json:"notes"`
	Reminders     int `json:"reminders"`
	Relationships int `json:"relationships"`
	ExtraNotes    int `json:"extra_notes"` // calls/tasks/gifts/debts converted to notes
}

// MonicaImportRowPreview extends the generic import preview row with
// Monica-specific context.
type MonicaImportRowPreview struct {
	ImportRowPreview
	Related  MonicaRowRelatedCounts `json:"related"`
	HasPhoto bool                   `json:"has_photo"`
}

// MonicaPreviewResponse contains the full preview of a fetched snapshot.
type MonicaPreviewResponse struct {
	SessionID      string                   `json:"session_id"`
	Rows           []MonicaImportRowPreview `json:"rows"`
	TotalRows      int                      `json:"total_rows"`
	ValidRows      int                      `json:"valid_rows"`
	DuplicateCount int                      `json:"duplicate_count"`
	ErrorCount     int                      `json:"error_count"`
	Totals         MonicaRowRelatedCounts   `json:"totals"`
}

// MonicaConfirmRequest executes the import with per-contact actions.
type MonicaConfirmRequest struct {
	SessionID string            `json:"session_id" validate:"required"`
	Actions   []RowImportAction `json:"actions" validate:"required,dive"`
}

// MonicaImportResult extends the generic import result with counts for the
// related entities and the asynchronous photo phase.
type MonicaImportResult struct {
	ImportResult
	ActivitiesCreated    int `json:"activities_created"`
	NotesCreated         int `json:"notes_created"`
	RemindersCreated     int `json:"reminders_created"`
	RelationshipsCreated int `json:"relationships_created"`
	// RelationshipsCollapsed counts reciprocal halves skipped because the
	// other direction was imported instead.
	RelationshipsCollapsed int `json:"relationships_collapsed"`
	PhotosQueued           int `json:"photos_queued"`
	PhotosSaved            int `json:"photos_saved"`
	PhotosFailed           int `json:"photos_failed"`
	// Issues lists every Monica record that was skipped or failed, or was
	// imported without part of its data, so the summary counts can be
	// traced back to concrete records.
	Issues []MonicaImportIssue `json:"issues"`
}

// Monica import issue statuses.
const (
	MonicaIssueSkipped = "skipped"
	MonicaIssueFailed  = "failed"
)

// Monica import issue reasons; the frontend translates them.
const (
	// MonicaReasonNotSelected: the user chose "skip" for the contact.
	MonicaReasonNotSelected = "not_selected"
	// MonicaReasonInvalid: the contact failed validation (Detail lists why).
	MonicaReasonInvalid = "invalid"
	// MonicaReasonNoMatch: "update" was chosen but no existing contact matched.
	MonicaReasonNoMatch = "no_match"
	// MonicaReasonContactNotImported: the record belongs to a contact that
	// was not imported.
	MonicaReasonContactNotImported = "contact_not_imported"
	// MonicaReasonAlreadyExists: an identical record is already in Meerkat
	// (e.g. from an earlier import run).
	MonicaReasonAlreadyExists = "already_exists"
	// MonicaReasonIncomplete: the record lacks data Meerkat needs (a name,
	// a usable date, content).
	MonicaReasonIncomplete = "incomplete"
	// MonicaReasonExpired: a one-time reminder whose date has passed.
	MonicaReasonExpired = "expired"
	// MonicaReasonUnsupportedFrequency: a reminder interval Meerkat lacks.
	MonicaReasonUnsupportedFrequency = "unsupported_frequency"
	// MonicaReasonApproximateBirthday: Monica only knows an age, not a date;
	// the birthday of a relationship cannot hold that.
	MonicaReasonApproximateBirthday = "approximate_birthday"
	// MonicaReasonBirthdayOnRelationship: a partial person's birthday
	// reminder, left out because the birthday is stored on the relationship,
	// which Meerkat already announces.
	MonicaReasonBirthdayOnRelationship = "birthday_on_relationship"
	// MonicaReasonPhotoFailed: the contact's photo could not be downloaded or stored.
	MonicaReasonPhotoFailed = "photo_failed"
	// MonicaReasonError: the database rejected the record (Detail has the error).
	MonicaReasonError = "error"
)

// MonicaImportIssue describes one Monica record that was not (fully) imported.
type MonicaImportIssue struct {
	Status  string `json:"status"`            // MonicaIssueSkipped or MonicaIssueFailed
	Entity  string `json:"entity"`            // contact, relationship, activity, note, reminder, call, task, gift, debt, birthday
	Contact string `json:"contact,omitempty"` // the Monica contact the record belongs to
	Label   string `json:"label,omitempty"`   // what the record is: a title, name or excerpt
	Reason  string `json:"reason"`            // one of the MonicaReason* codes
	Detail  string `json:"detail,omitempty"`  // validation or database error text
	// Count is how many records this row stands for when several were
	// rolled up into it (records of a contact that was not imported); their
	// individual labels are then dropped.
	Count int `json:"count,omitempty"`
	// Cause is the raw error behind a failure, for the server log only.
	Cause string `json:"-"`
}
