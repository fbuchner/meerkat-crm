package models

import "time"

// ImportableContactFields defines the valid target fields for import
var ImportableContactFields = []string{
	"firstname", "lastname", "nickname", "gender", "email", "phone",
	"birthday", "address", "how_we_met", "food_preference",
	"work_information", "contact_information", "circles",
}

// ColumnMapping represents how a CSV column maps to a contact field
type ColumnMapping struct {
	CSVColumn    string `json:"csv_column" validate:"required"`
	ContactField string `json:"contact_field"` // Empty means "ignore this column"
}

// ImportUploadResponse is returned after CSV upload
type ImportUploadResponse struct {
	SessionID         string          `json:"session_id"`
	Headers           []string        `json:"headers"`
	SuggestedMappings []ColumnMapping `json:"suggested_mappings"`
	RowCount          int             `json:"row_count"`
	SampleData        [][]string      `json:"sample_data"` // First few rows for preview
}

// ImportPreviewRequest is sent to request a preview with mappings
type ImportPreviewRequest struct {
	SessionID string          `json:"session_id" validate:"required"`
	Mappings  []ColumnMapping `json:"mappings" validate:"required,dive"`
}

// DuplicateMatch describes a potential duplicate contact
type DuplicateMatch struct {
	ExistingContactID uint   `json:"existing_contact_id"`
	ExistingFirstname string `json:"existing_firstname"`
	ExistingLastname  string `json:"existing_lastname"`
	ExistingEmail     string `json:"existing_email"`
	ExistingPhone     string `json:"existing_phone"`
	MatchReason       string `json:"match_reason"` // "email", "name", or "phone"
}

// ImportRowPreview represents one row in the import preview
type ImportRowPreview struct {
	RowIndex         int                    `json:"row_index"`
	ParsedContact    map[string]interface{} `json:"parsed_contact"`    // Parsed field values
	ValidationErrors []string               `json:"validation_errors"` // Any validation issues
	DuplicateMatch   *DuplicateMatch        `json:"duplicate_match"`   // Potential duplicate, if any
	SuggestedAction  string                 `json:"suggested_action"`  // "add", "skip", or "update"
}

// ImportPreviewResponse contains the full preview data
type ImportPreviewResponse struct {
	SessionID      string             `json:"session_id"`
	Rows           []ImportRowPreview `json:"rows"`
	TotalRows      int                `json:"total_rows"`
	ValidRows      int                `json:"valid_rows"`
	DuplicateCount int                `json:"duplicate_count"`
	ErrorCount     int                `json:"error_count"`
}

// RowImportAction specifies what to do with each row
type RowImportAction struct {
	RowIndex int    `json:"row_index" validate:"min=0"`
	Action   string `json:"action" validate:"required,oneof=skip add update"`
}

// ImportConfirmRequest is sent to execute the import
type ImportConfirmRequest struct {
	SessionID string            `json:"session_id" validate:"required"`
	Actions   []RowImportAction `json:"actions" validate:"required,dive"`
}

// ImportResult summarizes what happened during import
type ImportResult struct {
	TotalProcessed int      `json:"total_processed"`
	Created        int      `json:"created"`
	Updated        int      `json:"updated"`
	Skipped        int      `json:"skipped"`
	Errors         []string `json:"errors"`
}

// ImportSession stores temporary import data server-side (not persisted to DB)
type ImportSession struct {
	ID        string
	UserID    uint
	Headers   []string
	Rows      [][]string
	CreatedAt time.Time
	ExpiresAt time.Time
	// Cached preview data (set after PreviewImport is called)
	Mappings      []ColumnMapping
	PreviewRows   []ImportRowPreview
	PreviewCached bool
}
