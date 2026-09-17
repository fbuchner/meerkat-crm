package services

import (
	"fmt"
	"meerkat/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetUpcomingBirthdays_ExcludesDeceasedContact verifies that a contact
// flagged as deceased is excluded from upcoming birthday reminders, mirroring
// the existing archived-contact suppression behavior.
func TestGetUpcomingBirthdays_ExcludesDeceasedContact(t *testing.T) {
	db, _ := setupRouter()

	user := models.User{Username: "deceased-user", Password: "password123", Email: "deceased@example.com"}
	require.NoError(t, db.Create(&user).Error)

	today := time.Now()

	deceased := models.Contact{
		UserID:     user.ID,
		Firstname:  "Deceased",
		Birthday:   fmt.Sprintf("2000-%02d-%02d", today.Month(), today.Day()),
		IsDeceased: true,
	}
	require.NoError(t, db.Create(&deceased).Error)

	alive := models.Contact{
		UserID:    user.ID,
		Firstname: "Alive",
		Birthday:  fmt.Sprintf("2000-%02d-%02d", today.Month(), today.Day()),
	}
	require.NoError(t, db.Create(&alive).Error)

	birthdays, err := GetUpcomingBirthdays(db, user.ID, today)
	require.NoError(t, err)

	names := make([]string, 0, len(birthdays))
	for _, b := range birthdays {
		names = append(names, b.Name)
	}
	assert.Contains(t, names, "Alive")
	assert.NotContains(t, names, "Deceased")
}

// TestGetUpcomingBirthdays_ExcludesRelationshipOfDeceasedContact verifies that
// a relationship's own birthday is suppressed when its parent contact is
// deceased, mirroring the existing archived-parent suppression behavior.
func TestGetUpcomingBirthdays_ExcludesRelationshipOfDeceasedContact(t *testing.T) {
	db, _ := setupRouter()

	user := models.User{Username: "deceased-rel-user", Password: "password123", Email: "deceased-rel@example.com"}
	require.NoError(t, db.Create(&user).Error)

	today := time.Now()

	deceasedParent := models.Contact{
		UserID:     user.ID,
		Firstname:  "DeceasedParent",
		IsDeceased: true,
	}
	require.NoError(t, db.Create(&deceasedParent).Error)

	aliveParent := models.Contact{
		UserID:    user.ID,
		Firstname: "AliveParent",
	}
	require.NoError(t, db.Create(&aliveParent).Error)

	deceasedParentRelationship := models.Relationship{
		UserID:    user.ID,
		Name:      "ChildOfDeceased",
		Type:      "Child",
		ContactID: deceasedParent.ID,
		Birthday:  fmt.Sprintf("2010-%02d-%02d", today.Month(), today.Day()),
	}
	require.NoError(t, db.Create(&deceasedParentRelationship).Error)

	aliveParentRelationship := models.Relationship{
		UserID:    user.ID,
		Name:      "ChildOfAlive",
		Type:      "Child",
		ContactID: aliveParent.ID,
		Birthday:  fmt.Sprintf("2010-%02d-%02d", today.Month(), today.Day()),
	}
	require.NoError(t, db.Create(&aliveParentRelationship).Error)

	birthdays, err := GetUpcomingBirthdays(db, user.ID, today)
	require.NoError(t, err)

	names := make([]string, 0, len(birthdays))
	for _, b := range birthdays {
		names = append(names, b.Name)
	}
	assert.Contains(t, names, "ChildOfAlive")
	assert.NotContains(t, names, "ChildOfDeceased")
}
