package controllers

import (
	"meerkat/models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExportData(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.GET("/export", ExportData)

	// Create test data
	contact1 := models.Contact{
		UserID:             user.ID,
		Firstname:          "Alice",
		Lastname:           "Johnson",
		Email:              "alice@example.com",
		Phone:              "123-456-7890",
		Birthday:           "1990-01-15",
		Address:            "123 Main St",
		HowWeMet:           "Work conference",
		FoodPreference:     "Vegetarian",
		WorkInformation:    "Software Engineer",
		ContactInformation: "Prefers email",
		Circles:            []string{"Friends", "Work"},
	}
	db.Create(&contact1)

	contact2 := models.Contact{
		UserID:    user.ID,
		Firstname: "Bob",
		Lastname:  "Smith",
		Email:     "bob@example.com",
		Circles:   []string{"Family"},
	}
	db.Create(&contact2)

	// Create a relationship
	relationship := models.Relationship{
		UserID:    user.ID,
		ContactID: contact1.ID,
		Name:      "Charlie",
		Type:      "Friend",
		Birthday:  "1985-05-20",
	}
	db.Create(&relationship)

	// Create an activity
	activityDate := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
	activity := models.Activity{
		UserID:      user.ID,
		Title:       "Coffee Meeting",
		Description: "Catch up over coffee",
		Location:    "Local Cafe",
		Date:        activityDate,
		Contacts:    []models.Contact{contact1},
	}
	db.Create(&activity)

	// Create a note
	noteDate := time.Date(2024, 7, 10, 10, 0, 0, 0, time.UTC)
	note := models.Note{
		UserID:    user.ID,
		ContactID: &contact1.ID,
		Content:   "Remember to follow up about the project",
		Date:      noteDate,
	}
	db.Create(&note)

	// Create a reminder
	byMail := false
	reoccur := true
	reminderDate := time.Date(2024, 8, 1, 9, 0, 0, 0, time.UTC)
	reminder := models.Reminder{
		UserID:                user.ID,
		ContactID:             &contact1.ID,
		Message:               "Birthday reminder",
		RemindAt:              reminderDate,
		Recurrence:            "yearly",
		ByMail:                &byMail,
		ReoccurFromCompletion: &reoccur,
		Completed:             false,
	}
	db.Create(&reminder)

	// Make the request
	req, _ := http.NewRequest("GET", "/export", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check headers
	contentType := w.Header().Get("Content-Type")
	assert.Contains(t, contentType, "text/csv")

	contentDisposition := w.Header().Get("Content-Disposition")
	assert.Contains(t, contentDisposition, "attachment")
	assert.Contains(t, contentDisposition, "meerkat-export")
	assert.Contains(t, contentDisposition, ".csv")

	// Check body content
	body := w.Body.String()

	// Verify contacts section
	assert.Contains(t, body, "=== CONTACTS ===")
	assert.Contains(t, body, "Alice")
	assert.Contains(t, body, "Johnson")
	assert.Contains(t, body, "alice@example.com")
	assert.Contains(t, body, "Bob")
	assert.Contains(t, body, "Smith")
	assert.Contains(t, body, "Friends; Work")

	// Verify relationships section
	assert.Contains(t, body, "=== RELATIONSHIPS ===")
	assert.Contains(t, body, "Charlie")
	assert.Contains(t, body, "Friend")

	// Verify activities section
	assert.Contains(t, body, "=== ACTIVITIES ===")
	assert.Contains(t, body, "Coffee Meeting")
	assert.Contains(t, body, "Catch up over coffee")
	assert.Contains(t, body, "Local Cafe")

	// Verify notes section
	assert.Contains(t, body, "=== NOTES ===")
	assert.Contains(t, body, "Remember to follow up about the project")

	// Verify reminders section
	assert.Contains(t, body, "=== REMINDERS ===")
	assert.Contains(t, body, "Birthday reminder")
	assert.Contains(t, body, "yearly")
}

func TestExportDataEmpty(t *testing.T) {
	_, router := setupRouter()

	router.GET("/export", ExportData)

	// Make the request with no data
	req, _ := http.NewRequest("GET", "/export", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check headers
	contentType := w.Header().Get("Content-Type")
	assert.Contains(t, contentType, "text/csv")

	// Check body still contains section headers
	body := w.Body.String()
	assert.Contains(t, body, "=== CONTACTS ===")
	assert.Contains(t, body, "=== RELATIONSHIPS ===")
	assert.Contains(t, body, "=== ACTIVITIES ===")
	assert.Contains(t, body, "=== NOTES ===")
	assert.Contains(t, body, "=== REMINDERS ===")
}

func TestExportDataUserScoping(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	// Create a second user
	otherUser := models.User{Username: "other", Password: "password456", Email: "other@example.com"}
	db.Create(&otherUser)

	router.GET("/export", ExportData)

	// Create contact for the first user
	contact1 := models.Contact{
		UserID:    user.ID,
		Firstname: "UserContact",
		Lastname:  "One",
	}
	db.Create(&contact1)

	// Create contact for the second user (should NOT appear in export)
	contact2 := models.Contact{
		UserID:    otherUser.ID,
		Firstname: "OtherUserContact",
		Lastname:  "Two",
	}
	db.Create(&contact2)

	// Make the request
	req, _ := http.NewRequest("GET", "/export", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()

	// Verify only the first user's contact is in the export
	assert.Contains(t, body, "UserContact")
	assert.True(t, strings.Contains(body, "UserContact"))
	assert.False(t, strings.Contains(body, "OtherUserContact"))
}
