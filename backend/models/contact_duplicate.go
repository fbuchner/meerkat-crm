package models

import "time"

// ContactDuplicateDismissal records a pair of contacts the user marked as
// "not a duplicate". ContactAID is always the smaller of the two ids.
type ContactDuplicateDismissal struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	UserID     uint      `gorm:"not null;uniqueIndex:idx_contact_duplicate_dismissals_pair,priority:1" json:"-"`
	ContactAID uint      `gorm:"column:contact_a_id;not null;uniqueIndex:idx_contact_duplicate_dismissals_pair,priority:2" json:"contact_a_id"`
	ContactBID uint      `gorm:"column:contact_b_id;not null;uniqueIndex:idx_contact_duplicate_dismissals_pair,priority:3" json:"contact_b_id"`
}

// DuplicateContactSummary is the slice of a contact shown in a duplicate suggestion.
type DuplicateContactSummary struct {
	ID             uint   `json:"ID"`
	Firstname      string `json:"firstname"`
	Lastname       string `json:"lastname"`
	Nickname       string `json:"nickname"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	Organization   string `json:"organization"`
	PhotoThumbnail string `json:"photo_thumbnail"`
	Archived       bool   `json:"archived"`
}

// DuplicatePair is two contacts that probably describe the same person.
// Reasons lists what they share: "name", "email" and/or "phone".
type DuplicatePair struct {
	Contacts [2]DuplicateContactSummary `json:"contacts"`
	Reasons  []string                   `json:"reasons"`
}

// MergeConflict is a field both contacts fill with different values. Field is
// the contact's JSON field name, "photo", or "custom_fields.<name>".
type MergeConflict struct {
	Field       string `json:"field"`
	TargetValue string `json:"target_value"`
	SourceValue string `json:"source_value"`
}

// MergeMovedCounts is how much of the source contact's data a merge moves over.
type MergeMovedCounts struct {
	Notes               int64 `json:"notes"`
	Activities          int64 `json:"activities"`
	Reminders           int64 `json:"reminders"`
	Relationships       int64 `json:"relationships"`
	EmploymentHistories int64 `json:"employment_histories"`
}

// MergePreview describes what merging Source into Target would do.
type MergePreview struct {
	Target    ContactResponse  `json:"target"`
	Source    ContactResponse  `json:"source"`
	Conflicts []MergeConflict  `json:"conflicts"`
	Moved     MergeMovedCounts `json:"moved"`
}
