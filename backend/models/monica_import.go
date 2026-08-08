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
	PhotosQueued         int `json:"photos_queued"`
	PhotosSaved          int `json:"photos_saved"`
	PhotosFailed         int `json:"photos_failed"`
}
