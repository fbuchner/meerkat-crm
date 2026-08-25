// backend/models/employment_history.go
package models

import (
	"time"

	"gorm.io/gorm"
)

// EmploymentHistory is a single job entry in a contact's employment timeline.
// An entry with an empty EndDate is considered "current" (still ongoing).
// When a contact is saved directly with scalar Organization/Department/JobTitle/Role
// values (CSV/Monica import, CardDAV sync, or a plain API PUT), Contact.AfterSave
// patches the winning open entry to match — see contact.go.
type EmploymentHistory struct {
	gorm.Model
	UserID       uint    `gorm:"not null;index" json:"-"`
	ContactID    uint    `gorm:"not null;index" json:"contact_id" validate:"required"`
	Contact      Contact `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Organization string  `gorm:"type:text" json:"organization" validate:"required,min=1,max=200"`
	Department   string  `gorm:"type:text" json:"department" validate:"max=200"`
	JobTitle     string  `gorm:"type:text" json:"job_title" validate:"max=200"`
	Role         string  `gorm:"type:text" json:"role" validate:"max=200"`
	StartDate    string  `json:"start_date" validate:"omitempty,partialdate"`
	EndDate      string  `json:"end_date" validate:"omitempty,partialdate"`
	Notes        string  `gorm:"type:text" json:"notes" validate:"max=1000"`
}

// PartialDateStart parses a "YYYY", "YYYY-MM", or "YYYY-MM-DD" string into the
// start instant of that period, so entries with different date precision can
// still be compared chronologically. Returns ok=false for an empty or
// unparseable string.
func PartialDateStart(s string) (t time.Time, ok bool) {
	for _, layout := range []string{"2006-01-02", "2006-01", "2006"} {
		if parsed, err := time.Parse(layout, s); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}
