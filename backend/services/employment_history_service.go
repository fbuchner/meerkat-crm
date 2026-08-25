package services

import (
	"meerkat/models"
	"sort"
	"time"

	"gorm.io/gorm"
)

// RecomputeContactEmploymentScalars sets a Contact's denormalized
// Organization/Department/JobTitle/Role fields from its current (open-ended)
// employment history entry with the most recent StartDate, and saves the
// Contact. Call this after any employment history CRUD operation so the
// contact form and vCard export stay in sync with the history just edited.
func RecomputeContactEmploymentScalars(db *gorm.DB, userID, contactID uint) error {
	var openEntries []models.EmploymentHistory
	if err := db.Where("user_id = ? AND contact_id = ? AND end_date = ''", userID, contactID).
		Find(&openEntries).Error; err != nil {
		return err
	}

	var contact models.Contact
	if err := db.Where("user_id = ?", userID).First(&contact, contactID).Error; err != nil {
		return err
	}

	SortEmploymentHistory(openEntries)
	if len(openEntries) > 0 {
		current := openEntries[0]
		contact.Organization = current.Organization
		contact.Department = current.Department
		contact.JobTitle = current.JobTitle
		contact.Role = current.Role
	} else {
		contact.Organization = ""
		contact.Department = ""
		contact.JobTitle = ""
		contact.Role = ""
	}

	return db.Save(&contact).Error
}

// SortEmploymentHistory orders entries most-recent-first: open-ended entries
// (no EndDate) sort above closed ones. Within each group, entries are ordered
// by their most relevant date (EndDate for closed entries, StartDate for open
// ones) descending; entries with no parseable date of that kind sort last
// within their group.
func SortEmploymentHistory(entries []models.EmploymentHistory) {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		aOpen, bOpen := a.EndDate == "", b.EndDate == ""
		if aOpen != bOpen {
			return aOpen
		}
		aKey, aOk := sortKey(a)
		bKey, bOk := sortKey(b)
		if aOk != bOk {
			return aOk
		}
		if aOk && bOk {
			return aKey.After(bKey)
		}
		return false
	})
}

// sortKey returns the date used to order an entry within its open/closed
// group: EndDate for a closed entry, StartDate for an open one.
func sortKey(e models.EmploymentHistory) (time.Time, bool) {
	if e.EndDate != "" {
		return models.PartialDateStart(e.EndDate)
	}
	return models.PartialDateStart(e.StartDate)
}
