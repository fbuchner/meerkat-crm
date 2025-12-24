package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"meerkat/models"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateReminder(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.POST("/contacts/:id/reminders", CreateReminder)

	// Create a contact for the reminder
	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "Tom",
		Lastname:  "Smith",
	}
	db.Create(&contact)

	// Create a new reminder
	newReminder := models.Reminder{
		UserID:     user.ID,
		Message:    "Catch-up with Tom",
		ByMail:     false,
		RemindAt:   time.Now().Add(24 * time.Hour), // Tomorrow
		Recurrence: "Once",
		Contact:    contact,
	}

	jsonValue, _ := json.Marshal(newReminder)
	req, _ := http.NewRequest("POST", "/contacts/"+strconv.Itoa(int(contact.ID))+"/reminders", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, "Reminder created successfully", responseBody["message"])
}

func TestGetReminder(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.GET("/reminders/:id", GetReminder)

	// Create a contact
	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "Emily",
		Lastname:  "Johnson",
	}

	// Create a reminder
	reminder := models.Reminder{
		UserID:     user.ID,
		Message:    "Catch-up",
		ByMail:     false,
		RemindAt:   time.Now().Add(24 * 7 * time.Hour), // In 1 week
		Recurrence: "Monthly",
		Contact:    contact,
	}
	db.Create(&reminder)

	// Fetch the reminder by ID
	req, _ := http.NewRequest("GET", "/reminders/"+strconv.Itoa(int(reminder.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody models.Reminder
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, reminder.Message, responseBody.Message) // Ensure response matches
}

func TestUpdateReminder(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.PUT("/reminders/:id", UpdateReminder)

	// Create a contact
	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "Jamie",
		Lastname:  "Smith",
	}
	db.Create(&contact)

	// Create a reminder
	reminder := models.Reminder{
		UserID:                user.ID,
		Message:               "Catch-up",
		ByMail:                false,
		RemindAt:              time.Now().Add(24 * 4 * 8 * time.Hour),
		Recurrence:            "Once",
		ReoccurFromCompletion: false,
		Contact:               contact,
	}
	db.Create(&reminder)

	fmt.Println("ID is" + strconv.Itoa(int(reminder.ID)))

	// Create updated reminder data
	updatedReminder := models.Reminder{
		UserID:                user.ID,
		Message:               "Catch-up with Jamie",
		ByMail:                true,
		RemindAt:              time.Now().Add(24 * 4 * 3 * time.Hour),
		Recurrence:            "Monthly",
		ReoccurFromCompletion: true,
	}
	jsonValue, _ := json.Marshal(updatedReminder)

	req, _ := http.NewRequest("PUT", "/reminders/"+strconv.Itoa(int(reminder.ID)), bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, "Reminder updated successfully", responseBody["message"])
}

func TestDeleteReminder(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.DELETE("/reminders/:id", DeleteReminder)

	// Create a reminder
	reminder := models.Reminder{
		UserID:                user.ID,
		Message:               "Wish happy birthday to Joan",
		ByMail:                true,
		RemindAt:              time.Date(2025, 05, 22, 12, 0, 0, 0, time.UTC), // Fixed date
		Recurrence:            "Yearly",
		ReoccurFromCompletion: false,
	}
	db.Create(&reminder)

	// Delete the reminder
	req, _ := http.NewRequest("DELETE", "/reminders/"+strconv.Itoa(int(reminder.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]string
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, "Reminder deleted", responseBody["message"])

	// Verify the reminder has been deleted
	var deletedReminder models.Reminder
	result := db.First(&deletedReminder, reminder.ID)
	assert.Error(t, result.Error) // Should return an error, as it has been deleted
}

func TestGetRemindersForContact(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.GET("/contacts/:id/reminders", GetRemindersForContact)

	// Create a contact
	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "Emily",
		Lastname:  "Johnson",
	}
	db.Create(&contact)

	// Create reminders for this contact
	reminder1 := models.Reminder{
		UserID:                user.ID,
		Message:               "Catch-up with Emily",
		ByMail:                false,
		RemindAt:              time.Now().Add(48 * time.Hour), // 2 days from now
		Recurrence:            "Quarterly",
		ReoccurFromCompletion: true,
		ContactID:             &contact.ID,
	}
	reminder2 := models.Reminder{
		UserID:                user.ID,
		Message:               "Book flight tickets",
		ByMail:                true,
		RemindAt:              time.Date(2025, 8, 4, 12, 0, 0, 0, time.UTC), // Fixed date
		Recurrence:            "Yearly",
		ReoccurFromCompletion: false,
		ContactID:             &contact.ID,
	}
	db.Create(&reminder1)
	db.Create(&reminder2)

	// Fetch reminders for the contact
	req, _ := http.NewRequest("GET", "/contacts/"+strconv.Itoa(int(contact.ID))+"/reminders", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Len(t, responseBody["reminders"], 2) // Should return both reminders for the contact
}
