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

// ContactInput represents the DTO for creating/updating contacts
type ContactInput struct {
	Firstname          string   `json:"firstname" validate:"required,min=1,max=100,safe_string"`
	Lastname           string   `json:"lastname" validate:"max=100,safe_string"`
	Nickname           string   `json:"nickname" validate:"max=50,safe_string"`
	Gender             string   `json:"gender" validate:"omitempty,oneof=male female other prefer_not_to_say"`
	Email              string   `json:"email" validate:"omitempty,email"`
	Phone              string   `json:"phone" validate:"omitempty,phone"`
	Birthday           string   `json:"birthday" validate:"omitempty,birthday"`
	Address            string   `json:"address" validate:"max=500,safe_string"`
	HowWeMet           string   `json:"how_we_met" validate:"max=1000,safe_string"`
	FoodPreference     string   `json:"food_preference" validate:"max=500,safe_string"`
	WorkInformation    string   `json:"work_information" validate:"max=1000,safe_string"`
	ContactInformation string   `json:"contact_information" validate:"max=1000,safe_string"`
	Circles            []string `json:"circles" validate:"unique_circles"`
}

// PasswordResetRequestInput captures email for initiating password reset
type PasswordResetRequestInput struct {
	Email string `json:"email" validate:"required,email"`
}

// PasswordResetConfirmInput carries token and new password for reset flow
type PasswordResetConfirmInput struct {
	Token    string `json:"token" validate:"required,min=16"`
	Password string `json:"password" validate:"required,min=8,strong_password"`
}

// ChangePasswordInput is used by authenticated users to rotate credentials
type ChangePasswordInput struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8,strong_password"`
}

// RelationshipInput represents the DTO for creating/updating relationships
// ContactID is not included as it comes from the URL parameter
type RelationshipInput struct {
	Name             string `json:"name" validate:"required,min=1,max=100,safe_string"`
	Type             string `json:"type" validate:"required,min=1,max=50,safe_string"`
	Gender           string `json:"gender" validate:"omitempty,oneof=male female other prefer_not_to_say"`
	Birthday         string `json:"birthday" validate:"omitempty,birthday"`
	RelatedContactID *uint  `json:"related_contact_id"`
}
