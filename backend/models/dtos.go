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
	ContactID *uint     `json:"contact_id" validate:"omitempty,gt=0"`
}

// ContactInput represents the DTO for creating/updating contacts
type ContactInput struct {
	Firstname          string            `json:"firstname" validate:"required,min=1,max=100,safe_string"`
	Lastname           string            `json:"lastname" validate:"max=100,safe_string"`
	Nickname           string            `json:"nickname" validate:"max=50,safe_string"`
	Gender             string            `json:"gender" validate:"omitempty,oneof=male female other prefer_not_to_say"`
	Email              string            `json:"email" validate:"omitempty,email"`
	Phone              string            `json:"phone" validate:"omitempty,phone"`
	Birthday           string            `json:"birthday" validate:"omitempty,birthday"`
	Address            string            `json:"address" validate:"max=500,safe_string"`
	HowWeMet           string            `json:"how_we_met" validate:"max=1000,safe_string"`
	FoodPreference     string            `json:"food_preference" validate:"max=500,safe_string"`
	WorkInformation    string            `json:"work_information" validate:"max=1000,safe_string"`
	ContactInformation string            `json:"contact_information" validate:"max=1000,safe_string"`
	Circles            []string          `json:"circles" validate:"unique_circles"`
	CustomFields       map[string]string `json:"custom_fields"`
}

// CustomFieldNamesInput represents the DTO for updating user's custom field definitions
type CustomFieldNamesInput struct {
	Names []string `json:"names" validate:"dive,max=100,safe_string"`
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

// ContactResponse represents the DTO returned from GET /contacts with photo thumbnail
type ContactResponse struct {
	Contact
	PhotoThumbnail string `json:"photo_thumbnail"`
}

// Birthday represents a unified birthday entry for contacts and relationships
type Birthday struct {
	Type                  string `json:"type"`                              // "contact" or "relationship"
	Name                  string `json:"name"`                              // Unified display name
	Birthday              string `json:"birthday"`                          // Birthday in DD.MM.YYYY or DD.MM. format
	PhotoThumbnail        string `json:"photo_thumbnail,omitempty"`         // Profile picture thumbnail (base64)
	ContactID             uint   `json:"contact_id"`                        // Contact ID (the person or parent contact for relationships)
	RelationshipType      string `json:"relationship_type,omitempty"`       // Relationship type (empty for contacts)
	AssociatedContactName string `json:"associated_contact_name,omitempty"` // Parent contact name (for relationships)
}

// GraphNode represents a node in the network visualization (contact or activity)
type GraphNode struct {
	ID             string   `json:"id"`                        // "c-{contactID}" or "a-{activityID}"
	Type           string   `json:"type"`                      // "contact" or "activity"
	Label          string   `json:"label"`                     // Display name or activity title
	PhotoThumbnail string   `json:"photo_thumbnail,omitempty"` // Profile picture for contacts (base64)
	Circles        []string `json:"circles,omitempty"`         // Circles for contacts
}

// GraphEdge represents an edge in the network visualization
type GraphEdge struct {
	ID     string `json:"id"`     // Unique edge ID
	Source string `json:"source"` // Source node ID
	Target string `json:"target"` // Target node ID
	Type   string `json:"type"`   // "relationship" or "activity"
	Label  string `json:"label"`  // Relationship type or activity title
}

// GraphResponse is the API response for the network graph
type GraphResponse struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// AdminUserResponse - user data returned to admin (no password)
type AdminUserResponse struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Language  string    `json:"language"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AdminUserUpdateInput - DTO for admin updating a user
type AdminUserUpdateInput struct {
	Username *string `json:"username" validate:"omitempty,min=3,max=50,safe_string,no_at_sign"`
	Email    *string `json:"email" validate:"omitempty,email"`
	Password *string `json:"password" validate:"omitempty,min=8,strong_password"`
	IsAdmin  *bool   `json:"is_admin"`
}

// AdminUsersListResponse - paginated list of users
type AdminUsersListResponse struct {
	Users      []AdminUserResponse `json:"users"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
	TotalPages int                 `json:"total_pages"`
}
