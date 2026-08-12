// Package monica implements a read-only client for the Monica CRM REST API
// (https://www.monicahq.com/api), used by the Monica import assistant.
// Wire structs only declare the fields the importer consumes and keep
// everything optional/tolerant, since payloads vary slightly across Monica
// versions.
package monica

// Meta is the pagination block returned by Monica list endpoints.
type Meta struct {
	CurrentPage int `json:"current_page"`
	LastPage    int `json:"last_page"`
	Total       int `json:"total"`
}

// Paged is the envelope of every Monica list response.
type Paged[T any] struct {
	Data []T  `json:"data"`
	Meta Meta `json:"meta"`
}

// SpecialDate is Monica's flexible date type for birthdays and similar dates:
// the date may be exact, year-unknown (month/day only), or derived from an age.
type SpecialDate struct {
	Date          *string `json:"date"` // RFC3339 or YYYY-MM-DD, nil if unknown
	IsAgeBased    bool    `json:"is_age_based"`
	IsYearUnknown bool    `json:"is_year_unknown"`
}

// Tag is a Monica tag (mapped to a Meerkat circle).
type Tag struct {
	Name string `json:"name"`
}

// Address is a Monica postal address.
type Address struct {
	Name       string `json:"name"` // label, e.g. "home"
	Street     string `json:"street"`
	City       string `json:"city"`
	Province   string `json:"province"`
	PostalCode string `json:"postal_code"`
	Country    *struct {
		Name string `json:"name"`
	} `json:"country"`
}

// ContactField is a Monica contact information entry (email, phone, social
// handle, ...); the type is fully denormalized into contact_field_type.
type ContactField struct {
	Content          string `json:"content"`
	ContactFieldType struct {
		Name     string  `json:"name"`     // "Email", "Phone", "Twitter", ...
		Protocol *string `json:"protocol"` // "mailto:", "tel:", or nil
		Type     *string `json:"type"`     // "email", "phone", or nil
	} `json:"contact_field_type"`
}

// ContactRef is the minimal contact reference embedded in related entities.
type ContactRef struct {
	ID           int    `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	CompleteName string `json:"complete_name"`
}

// Contact is a Monica contact as returned by GET /contacts?with=contactfields.
type Contact struct {
	ID              int    `json:"id"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Nickname        string `json:"nickname"`
	Gender          string `json:"gender"`      // display name, e.g. "Man"
	GenderType      string `json:"gender_type"` // "M", "F", "O"
	IsPartial       bool   `json:"is_partial"`
	IsDead          bool   `json:"is_dead"`
	IsStarred       bool   `json:"is_starred"`
	Description     string `json:"description"`
	FoodPreferences string `json:"food_preferences"`
	Information     struct {
		Dates struct {
			Birthdate    SpecialDate `json:"birthdate"`
			DeceasedDate SpecialDate `json:"deceased_date"`
		} `json:"dates"`
		Career struct {
			Job     *string `json:"job"`
			Company *string `json:"company"`
		} `json:"career"`
		Avatar struct {
			URL    *string `json:"url"`
			Source *string `json:"source"` // "default", "photo", "gravatar", ...
		} `json:"avatar"`
		HowYouMet struct {
			GeneralInformation *string `json:"general_information"`
		} `json:"how_you_met"`
	} `json:"information"`
	Tags          []Tag          `json:"tags"`
	Addresses     []Address      `json:"addresses"`
	ContactFields []ContactField `json:"contactFields"`
}

// Photo is one of a contact's stored photos. Monica serves the image bytes
// inline as a base64 data URL
type Photo struct {
	ID          int    `json:"id"`
	NewFilename string `json:"new_filename"` // e.g. "photos/<hash>.jpg"
	MimeType    string `json:"mime_type"`
	DataURL     string `json:"dataUrl"`
}

// Activity is a shared event linked to one or more contacts.
type Activity struct {
	ID           int    `json:"id"`
	Summary      string `json:"summary"`
	Description  string `json:"description"`
	HappenedAt   string `json:"happened_at"` // YYYY-MM-DD
	ActivityType *struct {
		Name string `json:"name"`
	} `json:"activity_type"`
	Attendees struct {
		Contacts []ContactRef `json:"contacts"`
	} `json:"attendees"`
}

// Note is a free-form note attached to a contact.
type Note struct {
	Body        string      `json:"body"`
	IsFavorited bool        `json:"is_favorited"`
	CreatedAt   string      `json:"created_at"`
	Contact     *ContactRef `json:"contact"`
}

// Reminder is a one-time or recurring reminder for a contact.
type Reminder struct {
	Title            string      `json:"title"`
	Description      string      `json:"description"`
	FrequencyType    string      `json:"frequency_type"` // one_time, week, month, year
	FrequencyNumber  int         `json:"frequency_number"`
	InitialDate      *string     `json:"initial_date"`
	NextExpectedDate *string     `json:"next_expected_date"`
	Contact          *ContactRef `json:"contact"`
}

// Call is a logged phone call with a contact.
type Call struct {
	Content  string      `json:"content"`
	CalledAt string      `json:"called_at"`
	Contact  *ContactRef `json:"contact"`
}

// Task is a to-do item, optionally linked to a contact.
type Task struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Completed   bool        `json:"completed"`
	CompletedAt *string     `json:"completed_at"`
	CreatedAt   string      `json:"created_at"`
	Contact     *ContactRef `json:"contact"`
}

// Gift is a gift idea or given/received gift for a contact.
type Gift struct {
	Name    string  `json:"name"`
	Comment string  `json:"comment"`
	URL     string  `json:"url"`
	Amount  *string `json:"amount"`
	Status  string  `json:"status"` // "idea", "offered", "received" (newer versions)
	Date    *string `json:"date"`
	// Older Monica versions use booleans instead of status.
	IsAnIdea        *bool       `json:"is_an_idea"`
	HasBeenOffered  *bool       `json:"has_been_offered"`
	HasBeenReceived *bool       `json:"has_been_received"`
	CreatedAt       string      `json:"created_at"`
	Contact         *ContactRef `json:"contact"`
}

// Debt is money owed to or by a contact.
type Debt struct {
	InDebt             string      `json:"in_debt"` // "yes" = user owes the contact
	Status             string      `json:"status"`
	Amount             float64     `json:"amount"`
	AmountWithCurrency string      `json:"amount_with_currency"`
	Reason             string      `json:"reason"`
	CreatedAt          string      `json:"created_at"`
	Contact            *ContactRef `json:"contact"`
}

// Relationship links two Monica contacts. "contact_is" IS the
// relationship_type of "of_contact" (e.g. John is the father of Jane).
type Relationship struct {
	RelationshipType struct {
		Name string `json:"name"`
	} `json:"relationship_type"`
	ContactIs *ContactRef `json:"contact_is"`
	OfContact *ContactRef `json:"of_contact"`
}

// EntityCounts holds per-entity totals reported by Monica, used for the
// pre-fetch summary and progress estimates.
type EntityCounts struct {
	Contacts   int `json:"contacts"`
	Activities int `json:"activities"`
	Notes      int `json:"notes"`
	Reminders  int `json:"reminders"`
	Calls      int `json:"calls"`
	Tasks      int `json:"tasks"`
	Gifts      int `json:"gifts"`
	Debts      int `json:"debts"`
}

// Total is the number of records a full fetch would hold in memory.
func (c EntityCounts) Total() int {
	return c.Contacts + c.Activities + c.Notes + c.Reminders +
		c.Calls + c.Tasks + c.Gifts + c.Debts
}

// Snapshot is the complete in-memory copy of a Monica account fetched by the
// import assistant. Relationships are keyed by Monica contact ID.
type Snapshot struct {
	Contacts      []Contact
	Activities    []Activity
	Notes         []Note
	Reminders     []Reminder
	Calls         []Call
	Tasks         []Task
	Gifts         []Gift
	Debts         []Debt
	Relationships map[int][]Relationship
}
