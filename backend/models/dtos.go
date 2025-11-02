package models

import "time"

// ActivityInput represents the DTO for creating/updating activities
type ActivityInput struct {
	Title       string    `json:"title" validate:"required,min=1,max=200,safe_string"`
	Description string    `json:"description" validate:"max=2000,safe_string"`
	Location    string    `json:"location" validate:"max=300,safe_string"`
	Date        time.Time `json:"date" validate:"required"`
	ContactIDs  []uint    `json:"contact_ids"` // Accept an array of contact IDs for many-to-many association
}

// NoteInput represents the DTO for creating/updating notes
type NoteInput struct {
	Content   string    `json:"content" validate:"required,min=1,max=5000,safe_string"`
	Date      time.Time `json:"date" validate:"required"`
	ContactID *uint     `json:"contact_id" validate:"required"`
}
