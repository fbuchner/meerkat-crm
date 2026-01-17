package services

import (
	"fmt"
	"meerkat/models"
	"slices"
	"time"

	"gorm.io/gorm"
)

// GetUpcomingBirthdays fetches upcoming birthdays for a specific user
// Returns birthdays sorted by days until birthday, with smart limits
func GetUpcomingBirthdays(db *gorm.DB, userID uint) ([]models.Birthday, error) {
	now := time.Now()
	currentDay := now.Format("02")
	currentMonth := now.Format("01")
	nextMonth := now.AddDate(0, 1, 0).Format("01")

	var birthdays []models.Birthday

	// Query upcoming contact birthdays
	// Birthday format is now YYYY-MM-DD or --MM-DD (ISO 8601)
	// Month is at position LENGTH-4 (2 chars), Day is at position LENGTH-1 (2 chars)
	var contacts []models.Contact
	contactQuery := db.Model(&models.Contact{}).
		Where("user_id = ?", userID).
		Where("birthday IS NOT NULL AND birthday != ''").
		Where(
			db.Where("SUBSTR(birthday, LENGTH(birthday) - 4, 2) = ? AND SUBSTR(birthday, LENGTH(birthday) - 1, 2) >= ?", currentMonth, currentDay).
				Or("SUBSTR(birthday, LENGTH(birthday) - 4, 2) = ?", nextMonth),
		)

	if err := contactQuery.Find(&contacts).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve upcoming birthdays: %w", err)
	}

	// Convert contacts to Birthday DTOs
	for _, contact := range contacts {
		name := contact.Firstname
		if contact.Nickname != "" {
			name = contact.Nickname
		}
		if contact.Lastname != "" {
			name += " " + contact.Lastname
		}

		birthdays = append(birthdays, models.Birthday{
			Type:         "contact",
			Name:         name,
			Birthday:     contact.Birthday,
			ThumbnailURL: contact.PhotoThumbnail,
			ContactID:    contact.ID,
		})
	}

	// Query upcoming relationship birthdays (only those without their own contact)
	// Birthday format is now YYYY-MM-DD or --MM-DD (ISO 8601)
	var relationships []models.Relationship
	relationshipQuery := db.Model(&models.Relationship{}).
		Preload("RelatedContact").
		Where("user_id = ?", userID).
		Where("related_contact_id IS NULL").
		Where("birthday IS NOT NULL AND birthday != ''").
		Where(
			db.Where("SUBSTR(birthday, LENGTH(birthday) - 4, 2) = ? AND SUBSTR(birthday, LENGTH(birthday) - 1, 2) >= ?", currentMonth, currentDay).
				Or("SUBSTR(birthday, LENGTH(birthday) - 4, 2) = ?", nextMonth),
		)

	if err := relationshipQuery.Find(&relationships).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve relationship birthdays: %w", err)
	}

	// Get parent contacts for relationships
	contactIDs := make([]uint, 0, len(relationships))
	for _, rel := range relationships {
		contactIDs = append(contactIDs, rel.ContactID)
	}

	parentContacts := make(map[uint]models.Contact)
	if len(contactIDs) > 0 {
		var parentContactList []models.Contact
		if err := db.Where("id IN ?", contactIDs).Find(&parentContactList).Error; err != nil {
			return nil, fmt.Errorf("failed to retrieve parent contacts: %w", err)
		}
		for _, pc := range parentContactList {
			parentContacts[pc.ID] = pc
		}
	}

	// Convert relationships to Birthday DTOs
	for _, rel := range relationships {
		parentContact := parentContacts[rel.ContactID]
		parentName := parentContact.Firstname
		if parentContact.Nickname != "" {
			parentName = parentContact.Nickname
		}
		if parentContact.Lastname != "" {
			parentName += " " + parentContact.Lastname
		}

		birthdays = append(birthdays, models.Birthday{
			Type:                  "relationship",
			Name:                  rel.Name,
			Birthday:              rel.Birthday,
			ThumbnailURL:          parentContact.PhotoThumbnail,
			ContactID:             rel.ContactID,
			RelationshipType:      rel.Type,
			AssociatedContactName: parentName,
		})
	}

	// Sort by days until birthday
	slices.SortFunc(birthdays, func(a, b models.Birthday) int {
		daysA := DaysUntilBirthday(a.Birthday, now)
		daysB := DaysUntilBirthday(b.Birthday, now)
		return daysA - daysB
	})

	// Apply limit: max 5, but include all birthdays within 2 weeks
	const maxResults = 5
	const twoWeeksDays = 14

	resultCount := 0
	for i, b := range birthdays {
		days := DaysUntilBirthday(b.Birthday, now)
		if days <= twoWeeksDays {
			resultCount = i + 1
		} else if resultCount < maxResults {
			resultCount = i + 1
		} else {
			break
		}
	}

	if resultCount < len(birthdays) {
		birthdays = birthdays[:resultCount]
	}

	return birthdays, nil
}

// DaysUntilBirthday calculates the number of days until a birthday from a given date
// Birthday format is YYYY-MM-DD or --MM-DD (ISO 8601)
func DaysUntilBirthday(birthday string, now time.Time) int {
	if len(birthday) < 7 {
		return 999
	}

	// Extract month and day from end of string (works for both YYYY-MM-DD and --MM-DD)
	// Month is at position len-5 to len-3, Day is at position len-2 to len
	length := len(birthday)
	month := birthday[length-5 : length-3]
	day := birthday[length-2 : length]

	d, err1 := time.Parse("02", day)
	m, err2 := time.Parse("01", month)
	if err1 != nil || err2 != nil {
		return 999
	}

	birthdayThisYear := time.Date(now.Year(), m.Month(), d.Day(), 0, 0, 0, 0, now.Location())
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if birthdayThisYear.Before(today) {
		birthdayThisYear = birthdayThisYear.AddDate(1, 0, 0)
	}

	return int(birthdayThisYear.Sub(today).Hours() / 24)
}
