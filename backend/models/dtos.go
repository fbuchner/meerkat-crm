package models

import (
	"mycorrhizal/contactmodel"
	"time"
)

// ActivityInput represents the DTO for creating/updating activities
type ActivityInput struct {
	Title       string    `json:"title" validate:"required,min=1,max=200"`
	Description string    `json:"description" validate:"max=2000"`
	Location    string    `json:"location" validate:"max=300"`
	Date        time.Time `json:"date" validate:"required"`
	ContactIDs  []uint    `json:"contact_ids"` // Accept an array of contact IDs for many-to-many association
	// Type/ExternalRef are WP-84's Interaction fields (activity.go) -- open/
	// conventional, no oneof validation, matching Activity.Type's own doc
	// comment on why it's deliberately unvalidated.
	Type        string `json:"type,omitempty"`
	ExternalRef string `json:"external_ref,omitempty"`
}

// CircleInput is the DTO for creating/updating a Circle (circle.go). Only
// Name is editable here -- membership lifecycle lives in its own
// AddCircleMember/RemoveCircleMember endpoints, not folded into update.
type CircleInput struct {
	Name string `json:"name" validate:"required,min=1,max=200"`
}

// CircleMemberInput is the DTO for POST /circles/:id/members.
type CircleMemberInput struct {
	MemberVCardUID string `json:"member_vcard_uid" validate:"required,uuid4"`
}

// HouseholdInput is the DTO for creating/updating a Household (household.go).
// Membership lifecycle lives in its own AddHouseholdMember/
// RemoveHouseholdMember endpoints, not folded into update — the same split as
// CircleInput. Type is a closed switch (drives the suggestion engine), so it
// carries the same `oneof` validation as the Household model field itself.
type HouseholdInput struct {
	Name    string                `json:"name" validate:"required,min=1,max=200"`
	Type    string                `json:"type" validate:"required,oneof=family_unit roommates other"`
	Address *contactmodel.Address `json:"address,omitempty"`
}

// HouseholdMemberInput is the DTO for POST /households/:id/members. Role is
// conventional, not validated (HouseholdRole* constants) — deliberately open,
// mirroring HouseholdMember.Role's own doc comment.
type HouseholdMemberInput struct {
	MemberVCardUID string `json:"member_vcard_uid" validate:"required,uuid4"`
	Role           string `json:"role,omitempty"`
	Since          string `json:"since,omitempty"`
	Until          string `json:"until,omitempty"`
}

// TagInput is the DTO for creating/updating a Tag (tag.go). Only Name is
// editable here -- tagging lifecycle lives in its own AddContactTag/
// RemoveContactTag endpoints, not folded into update.
type TagInput struct {
	Name string `json:"name" validate:"required,min=1,max=200"`
}

// ContactTagInput is the DTO for POST /tags/:id/contacts.
type ContactTagInput struct {
	ContactVCardUID string `json:"contact_vcard_uid" validate:"required,uuid4"`
}

// LifeEventInput is the DTO for creating/updating a LifeEvent (life_event.go).
type LifeEventInput struct {
	EntityID         string                    `json:"entity_id" validate:"required,uuid4"`
	Type             string                    `json:"type,omitempty"`
	Date             *contactmodel.PartialDate `json:"date,omitempty"`
	Description      string                    `json:"description,omitempty" validate:"max=2000"`
	Source           string                    `json:"source,omitempty" validate:"omitempty,oneof=user imported ai-suggested"`
	RelatedEntityIDs []string                  `json:"related_entity_ids,omitempty"`
	Remind           bool                      `json:"remind,omitempty"`
}

// PreferenceInput is the DTO for creating/updating a Preference
// (preference.go, §91.9). Category is deliberately not `oneof`-validated —
// it is an open classifier (see Preference's doc comment). Sensitivity
// defaults to normal server-side when omitted (mirroring
// FieldValue/RelationshipEdge's own defaults), not validated as required.
type PreferenceInput struct {
	EntityID      string     `json:"entity_id" validate:"required,uuid4"`
	Category      string     `json:"category" validate:"required,max=100"`
	Key           string     `json:"key,omitempty" validate:"omitempty,max=100"`
	Value         string     `json:"value" validate:"required,max=1000"`
	Source        string     `json:"source,omitempty" validate:"omitempty,oneof=conversation_note user ai-suggested external"`
	Confidence    *float64   `json:"confidence,omitempty"`
	LastConfirmed *time.Time `json:"last_confirmed,omitempty"`
	Sensitivity   string     `json:"sensitivity,omitempty" validate:"omitempty,oneof=normal private secret"`
}

// CalendarSubscriptionInput is the DTO for creating/updating a calendar subscription.
// Credentials are optional (public/unprotected calendars). On update, an empty
// password keeps the stored one; set ClearPassword to remove it.
type CalendarSubscriptionInput struct {
	Name          string `json:"name" validate:"required,min=1,max=100"`
	URL           string `json:"url" validate:"required,url,max=2048"`
	Username      string `json:"username" validate:"omitempty,max=200"`
	Password      string `json:"password" validate:"omitempty,max=500"`
	ClearPassword bool   `json:"clear_password"`
	SyncEnabled   *bool  `json:"sync_enabled"`
	PastDays      *int   `json:"past_days" validate:"omitempty,min=0,max=3650"`
	FutureDays    *int   `json:"future_days" validate:"omitempty,min=0,max=3650"`
}

// CalendarSubscriptionResponse is the DTO returned for a calendar subscription (no password)
type CalendarSubscriptionResponse struct {
	ID             uint       `json:"id"`
	Name           string     `json:"name"`
	URL            string     `json:"url"`
	Username       string     `json:"username"`
	HasPassword    bool       `json:"has_password"`
	SyncEnabled    bool       `json:"sync_enabled"`
	PastDays       int        `json:"past_days"`
	FutureDays     int        `json:"future_days"`
	LastSyncedAt   *time.Time `json:"last_synced_at"`
	LastSyncStatus string     `json:"last_sync_status"`
	LastSyncError  string     `json:"last_sync_error"`
	CreatedAt      time.Time  `json:"created_at"`
}

// ContactSubscriptionInput is the DTO for creating/updating a CardDAV
// contact subscription (WP-73b). Credentials are optional (some servers
// allow anonymous/public read). On update, an empty password keeps the
// stored one; set ClearPassword to remove it. Mirrors
// CalendarSubscriptionInput's shape, minus PastDays/FutureDays (no contacts
// analog).
type ContactSubscriptionInput struct {
	Name          string `json:"name" validate:"required,min=1,max=100"`
	URL           string `json:"url" validate:"required,url,max=2048"`
	Username      string `json:"username" validate:"omitempty,max=200"`
	Password      string `json:"password" validate:"omitempty,max=500"`
	ClearPassword bool   `json:"clear_password"`
	SyncEnabled   *bool  `json:"sync_enabled"`
}

// ContactSubscriptionResponse is the DTO returned for a contact subscription
// (no password, no sync token).
type ContactSubscriptionResponse struct {
	ID             uint       `json:"id"`
	Name           string     `json:"name"`
	URL            string     `json:"url"`
	Username       string     `json:"username"`
	HasPassword    bool       `json:"has_password"`
	SyncEnabled    bool       `json:"sync_enabled"`
	LastSyncedAt   *time.Time `json:"last_synced_at"`
	LastSyncStatus string     `json:"last_sync_status"`
	LastSyncError  string     `json:"last_sync_error"`
	CreatedAt      time.Time  `json:"created_at"`
}

// NoteInput represents the DTO for creating/updating notes
type NoteInput struct {
	Content   string    `json:"content" validate:"required,min=1,max=5000"`
	Date      time.Time `json:"date" validate:"required"`
	ContactID *uint     `json:"contact_id" validate:"omitempty,gt=0"`
}

// CustomFieldNamesInput represents the DTO for updating user's custom field definitions
type CustomFieldNamesInput struct {
	Names []string `json:"names" validate:"dive,max=100"`
}

// represents the DTO for updating which extended contact fields are visible in the UI. A nil/absent list means "use the default set"
type EnabledContactFieldsInput struct {
	Fields []string `json:"fields" validate:"dive,max=50"`
}

// UserRegistrationInput represents the DTO for user registration
// This DTO intentionally excludes IsAdmin to prevent mass assignment attacks
type UserRegistrationInput struct {
	Username string `json:"username" validate:"required,min=1,max=50,no_at_sign"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,strong_password"`
	Language string `json:"language" validate:"omitempty,oneof=en de it es fr"`
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

// ThinContactInput creates a not-yet-existing "thin" Contact inline as one
// endpoint of a RelationshipEdge — the graph-model equivalent of the legacy
// Relationship{Name, Gender, Birthday, RelatedContactID: nil} case (the
// legacy model and its one-time migration tool are both gone as of §3d
// WP5; this comment is kept only as historical context for the field
// mapping's origin).
type ThinContactInput struct {
	Name     string `json:"name" validate:"required,min=1,max=100"`
	Gender   string `json:"gender" validate:"omitempty,oneof=male female other prefer_not_to_say"`
	Birthday string `json:"birthday" validate:"omitempty,birthday"`
}

// RelationshipEdgeInput is the DTO for creating/updating a RelationshipEdge.
// Each endpoint is either an existing Contact (SourceID/TargetID, a
// Contact.VCardUID) or a not-yet-existing person described inline
// (SourceThin/TargetThin) — exactly one of the pair per endpoint, enforced
// in the controller (not via validate tags — this codebase has no existing
// precedent for required_without/excluded_with, and a controller-level
// check gives a much more precise per-field error).
//
// Directional/Source/Confidence/Status are deliberately absent: the server
// always derives Directional from Type (!IsSymmetricRelationType), and
// always sets Source/Confidence/Status itself — a client can never
// mass-assign a fabricated provenance or self-confirm a status. A
// user-created edge is always Source: user-confirmed, Confidence: 1.0,
// Status: confirmed. Status only ever changes via AcceptRelationshipEdge.
type RelationshipEdgeInput struct {
	SourceID   string            `json:"source_id,omitempty" validate:"omitempty,uuid4"`
	SourceThin *ThinContactInput `json:"source_thin,omitempty"`
	TargetID   string            `json:"target_id,omitempty" validate:"omitempty,uuid4"`
	TargetThin *ThinContactInput `json:"target_thin,omitempty"`

	Type        string                 `json:"type" validate:"required,relation_type"`
	Sensitivity string                 `json:"sensitivity,omitempty" validate:"omitempty,oneof=normal private secret"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ContactResponse represents the DTO returned from GET /contacts with photo thumbnail
type ContactResponse struct {
	Contact
	PhotoThumbnail string `json:"photo_thumbnail"`
}

// Birthday represents an upcoming contact birthday
type Birthday struct {
	Type           string `json:"type"`                      // always "contact"
	Name           string `json:"name"`                      // Unified display name
	Birthday       string `json:"birthday"`                  // Birthday in YYYY-MM-DD format
	PhotoThumbnail string `json:"photo_thumbnail,omitempty"` // Profile picture thumbnail (base64)
	ContactID      uint   `json:"contact_id"`                // Contact ID
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
	ID         uint      `json:"id"`
	Username   string    `json:"username"`
	Email      string    `json:"email"`
	Language   string    `json:"language"`
	DateFormat string    `json:"date_format"`
	IsAdmin    bool      `json:"is_admin"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// /users/me payload: standard user fields plus caller's UI preferences (custom field names and enabled contact fields)
type CurrentUserResponse struct {
	AdminUserResponse
	CustomFieldNames     []string `json:"custom_field_names"`
	EnabledContactFields []string `json:"enabled_contact_fields"`
}

// AdminUserUpdateInput - DTO for admin updating a user
type AdminUserUpdateInput struct {
	Username *string `json:"username" validate:"omitempty,min=1,max=50,no_at_sign"`
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

// DefaultApiTokenExpiryDays is used when a token is created without an explicit
// lifetime. Tokens are long-lived bearer credentials, so they get a bounded
// life by default rather than living forever until someone remembers to revoke.
const DefaultApiTokenExpiryDays = 90

// MaxApiTokenExpiryDays caps how far out a token may be dated.
const MaxApiTokenExpiryDays = 365

// DefaultApiTokenScope is used when a token is created without an explicit scope.
const DefaultApiTokenScope = "full"

// ApiTokenInput represents the DTO for creating an API token
type ApiTokenInput struct {
	Name string `json:"name" validate:"required,min=1,max=100"`
	// ExpiresInDays is optional; omitting it applies DefaultApiTokenExpiryDays.
	// There is deliberately no "never expires" option.
	ExpiresInDays *int `json:"expires_in_days" validate:"omitempty,min=1,max=365"`
	// Scope is optional; omitting it applies DefaultApiTokenScope ("full").
	// "carddav" restricts the token to the CardDAV Basic-Auth path only.
	Scope string `json:"scope" validate:"omitempty,oneof=full carddav"`
}

// ApiTokenResponse represents the DTO returned for an API token
type ApiTokenResponse struct {
	ID         uint       `json:"id"`
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	Scope      string     `json:"scope"`
}

// ApiTokenCreateResponse is returned on token creation and includes the plaintext token
type ApiTokenCreateResponse struct {
	ApiTokenResponse
	Token string `json:"token"`
}

// WebhookInput is the DTO for creating/updating a webhook
type WebhookInput struct {
	Name     string   `json:"name" validate:"required,min=1,max=200"`
	URL      string   `json:"url" validate:"required,http_url"`
	Events   []string `json:"events" validate:"required,min=1,dive,oneof=contact.created contact.updated contact.deleted note.created note.updated note.deleted activity.created activity.updated activity.deleted reminder.triggered birthday.occurred"`
	IsActive bool     `json:"is_active"`
}

// WebhookResponse is the DTO returned for a webhook (no secret)
type WebhookResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// WebhookCreateResponse is the DTO returned once after creation — includes the plaintext secret
type WebhookCreateResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	Secret    string    `json:"secret"`
}

// WebhookDeliveryResponse is the DTO returned for a webhook delivery record
type WebhookDeliveryResponse struct {
	ID          uint       `json:"id"`
	WebhookID   uint       `json:"webhook_id"`
	EventType   string     `json:"event_type"`
	StatusCode  *int       `json:"status_code"`
	Error       *string    `json:"error"`
	Attempts    int        `json:"attempts"`
	NextRetryAt *time.Time `json:"next_retry_at"`
	CreatedAt   time.Time  `json:"created_at"`
}
