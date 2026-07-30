package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mycorrhizal/models"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateReminder(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.POST("/contacts/:id/reminders", withValidated(func() any { return &models.Reminder{} }), CreateReminder)

	// Helper for bool pointers
	boolPtr := func(b bool) *bool { return &b }

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
		ByMail:     boolPtr(false),
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

	// Helper for bool pointers
	boolPtr := func(b bool) *bool { return &b }

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
		ByMail:     boolPtr(false),
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
	router.PUT("/reminders/:id", withValidated(func() any { return &models.Reminder{} }), UpdateReminder)

	// Helper for bool pointers
	boolPtr := func(b bool) *bool { return &b }

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
		ByMail:                boolPtr(false),
		RemindAt:              time.Now().Add(24 * 4 * 8 * time.Hour),
		Recurrence:            "Once",
		ReoccurFromCompletion: boolPtr(false),
		Contact:               contact,
	}
	db.Create(&reminder)

	fmt.Println("ID is" + strconv.Itoa(int(reminder.ID)))

	// Create updated reminder data
	updatedReminder := models.Reminder{
		UserID:                user.ID,
		Message:               "Catch-up with Jamie",
		ByMail:                boolPtr(true),
		RemindAt:              time.Now().Add(24 * 4 * 3 * time.Hour),
		Recurrence:            "Monthly",
		ReoccurFromCompletion: boolPtr(true),
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

	// Helper for bool pointers
	boolPtr := func(b bool) *bool { return &b }

	// Create a contact for the reminder
	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "Joan",
		Lastname:  "Doe",
	}
	db.Create(&contact)

	// Create a reminder
	reminder := models.Reminder{
		UserID:                user.ID,
		Message:               "Wish happy birthday to Joan",
		ByMail:                boolPtr(true),
		RemindAt:              time.Date(2025, 05, 22, 12, 0, 0, 0, time.UTC), // Fixed date
		Recurrence:            "yearly",
		ReoccurFromCompletion: boolPtr(false),
		ContactID:             &contact.ID,
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

	// Helper for bool pointers
	boolPtr := func(b bool) *bool { return &b }

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
		ByMail:                boolPtr(false),
		RemindAt:              time.Now().Add(48 * time.Hour), // 2 days from now
		Recurrence:            "Quarterly",
		ReoccurFromCompletion: boolPtr(true),
		ContactID:             &contact.ID,
	}
	reminder2 := models.Reminder{
		UserID:                user.ID,
		Message:               "Book flight tickets",
		ByMail:                boolPtr(true),
		RemindAt:              time.Date(2025, 8, 4, 12, 0, 0, 0, time.UTC), // Fixed date
		Recurrence:            "Yearly",
		ReoccurFromCompletion: boolPtr(false),
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

func TestGetAllReminders(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.GET("/reminders", GetAllReminders)

	// Helper for bool pointers
	boolPtr := func(b bool) *bool { return &b }

	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "Amy",
		Lastname:  "Lee",
	}
	db.Create(&contact)

	later := models.Reminder{
		UserID:     user.ID,
		Message:    "Later reminder",
		ByMail:     boolPtr(false),
		RemindAt:   time.Now().Add(72 * time.Hour),
		Recurrence: "once",
		ContactID:  &contact.ID,
	}
	sooner := models.Reminder{
		UserID:     user.ID,
		Message:    "Sooner reminder",
		ByMail:     boolPtr(false),
		RemindAt:   time.Now().Add(24 * time.Hour),
		Recurrence: "once",
		ContactID:  &contact.ID,
	}
	db.Create(&later)
	db.Create(&sooner)

	req, _ := http.NewRequest("GET", "/reminders", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	reminders, ok := responseBody["reminders"].([]any)
	if !ok {
		t.Fatalf("expected reminders array in response")
	}
	assert.Len(t, reminders, 2)
	// Ordered ascending by remind_at: sooner first
	assert.Equal(t, "Sooner reminder", reminders[0].(map[string]any)["message"])
	assert.Equal(t, "Later reminder", reminders[1].(map[string]any)["message"])
}

func TestGetUpcomingReminders_ReturnsAtLeastFiveWhenFewDueSoon(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.GET("/reminders/upcoming", GetUpcomingReminders)

	boolPtr := func(b bool) *bool { return &b }

	contact := models.Contact{UserID: user.ID, Firstname: "Amy", Lastname: "Lee"}
	db.Create(&contact)

	makeReminder := func(msg string, in time.Duration, completed bool) models.Reminder {
		r := models.Reminder{
			UserID:     user.ID,
			Message:    msg,
			ByMail:     boolPtr(false),
			RemindAt:   time.Now().Add(in),
			Recurrence: "once",
			ContactID:  &contact.ID,
			Completed:  completed,
		}
		db.Create(&r)
		return r
	}

	// Two reminders due within the next 7 days.
	makeReminder("Due soon 1", 24*time.Hour, false)
	makeReminder("Due soon 2", 72*time.Hour, false)
	// A completed reminder within the window must be excluded.
	makeReminder("Already done", 48*time.Hour, true)
	// Four reminders beyond the 7-day window; only the earliest three should
	// be pulled in to reach the minimum of five.
	makeReminder("Far 1", 10*24*time.Hour, false)
	makeReminder("Far 2", 11*24*time.Hour, false)
	makeReminder("Far 3", 12*24*time.Hour, false)
	makeReminder("Far 4", 13*24*time.Hour, false)

	req, _ := http.NewRequest("GET", "/reminders/upcoming", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	reminders, ok := responseBody["reminders"].([]any)
	if !ok {
		t.Fatalf("expected reminders array in response")
	}
	assert.Len(t, reminders, 5)

	messages := make([]string, len(reminders))
	for i, r := range reminders {
		messages[i] = r.(map[string]any)["message"].(string)
	}
	assert.Equal(t, []string{"Due soon 1", "Due soon 2", "Far 1", "Far 2", "Far 3"}, messages)
}

func TestGetUpcomingReminders_ReturnsAllWhenManyDueSoon(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.GET("/reminders/upcoming", GetUpcomingReminders)

	boolPtr := func(b bool) *bool { return &b }

	contact := models.Contact{UserID: user.ID, Firstname: "Amy", Lastname: "Lee"}
	db.Create(&contact)

	makeReminder := func(msg string, in time.Duration) {
		r := models.Reminder{
			UserID:     user.ID,
			Message:    msg,
			ByMail:     boolPtr(false),
			RemindAt:   time.Now().Add(in),
			Recurrence: "once",
			ContactID:  &contact.ID,
		}
		db.Create(&r)
	}

	// Six reminders due within the next 7 days - exceeds the five-item floor,
	// so all six should come back rather than being capped at five.
	for i := 1; i <= 6; i++ {
		makeReminder(fmt.Sprintf("Due soon %d", i), time.Duration(i)*24*time.Hour)
	}
	// One reminder well beyond the window that must not appear.
	makeReminder("Far away", 10*24*time.Hour)

	req, _ := http.NewRequest("GET", "/reminders/upcoming", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	reminders, ok := responseBody["reminders"].([]any)
	if !ok {
		t.Fatalf("expected reminders array in response")
	}
	assert.Len(t, reminders, 6)
	for _, r := range reminders {
		assert.NotEqual(t, "Far away", r.(map[string]any)["message"])
	}
}

func TestCompleteReminder_OnceDeletesAndRecordsCompletion(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.PATCH("/reminders/:id/complete", CompleteReminder)

	boolPtr := func(b bool) *bool { return &b }

	contact := models.Contact{UserID: user.ID, Firstname: "Joan", Lastname: "Doe"}
	db.Create(&contact)

	reminder := models.Reminder{
		UserID:     user.ID,
		Message:    "One-off task",
		ByMail:     boolPtr(false),
		RemindAt:   time.Now().Add(-24 * time.Hour),
		Recurrence: "once",
		ContactID:  &contact.ID,
	}
	db.Create(&reminder)

	req, _ := http.NewRequest("PATCH", "/reminders/"+strconv.Itoa(int(reminder.ID))+"/complete", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]string
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, "Reminder completed and deleted", responseBody["message"])

	// "once" reminders are deleted after completion.
	var deleted models.Reminder
	result := db.First(&deleted, reminder.ID)
	assert.Error(t, result.Error)

	// A completion record should have been created for the timeline.
	var completions []models.ReminderCompletion
	db.Where("reminder_id = ?", reminder.ID).Find(&completions)
	if assert.Len(t, completions, 1) {
		assert.Equal(t, user.ID, completions[0].UserID)
		assert.Equal(t, contact.ID, completions[0].ContactID)
		assert.Equal(t, "One-off task", completions[0].Message)
	}
}

func TestCompleteReminder_RecurringReschedulesAndKeepsRecord(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.PATCH("/reminders/:id/complete", CompleteReminder)

	boolPtr := func(b bool) *bool { return &b }

	contact := models.Contact{UserID: user.ID, Firstname: "Jamie", Lastname: "Smith"}
	db.Create(&contact)

	originalRemindAt := time.Now().Add(-48 * time.Hour) // Overdue by 2 days
	reminder := models.Reminder{
		UserID:                user.ID,
		Message:               "Weekly check-in",
		ByMail:                boolPtr(false),
		RemindAt:              originalRemindAt,
		Recurrence:            "weekly",
		ReoccurFromCompletion: boolPtr(true),
		ContactID:             &contact.ID,
	}
	db.Create(&reminder)

	req, _ := http.NewRequest("PATCH", "/reminders/"+strconv.Itoa(int(reminder.ID))+"/complete", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, "Reminder completed successfully", responseBody["message"])

	// Recurring reminders survive completion, rescheduled to the next occurrence.
	var updated models.Reminder
	require.NoError(t, db.First(&updated, reminder.ID).Error)
	assert.False(t, updated.Completed)
	assert.False(t, updated.EmailSent)
	assert.True(t, updated.RemindAt.After(originalRemindAt))
	assert.True(t, updated.RemindAt.After(time.Now())) // Rescheduled into the future

	// A completion record should still be created for the timeline.
	var completions []models.ReminderCompletion
	db.Where("reminder_id = ?", reminder.ID).Find(&completions)
	assert.Len(t, completions, 1)
}

func TestCompleteReminder_SkipDoesNotCreateCompletionRecord(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.PATCH("/reminders/:id/complete", CompleteReminder)

	boolPtr := func(b bool) *bool { return &b }

	contact := models.Contact{UserID: user.ID, Firstname: "Jamie", Lastname: "Smith"}
	db.Create(&contact)

	reminder := models.Reminder{
		UserID:                user.ID,
		Message:               "Monthly check-in",
		ByMail:                boolPtr(false),
		RemindAt:              time.Now().Add(-24 * time.Hour),
		Recurrence:            "monthly",
		ReoccurFromCompletion: boolPtr(true),
		ContactID:             &contact.ID,
	}
	db.Create(&reminder)

	req, _ := http.NewRequest("PATCH", "/reminders/"+strconv.Itoa(int(reminder.ID))+"/complete?skip=true", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, "Reminder skipped successfully", responseBody["message"])

	var completions []models.ReminderCompletion
	db.Where("reminder_id = ?", reminder.ID).Find(&completions)
	assert.Len(t, completions, 0)
}

func TestGetCompletionsForContact(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.GET("/contacts/:id/completions", GetCompletionsForContact)

	contact := models.Contact{UserID: user.ID, Firstname: "Amy", Lastname: "Lee"}
	db.Create(&contact)

	older := models.ReminderCompletion{
		UserID:      user.ID,
		ContactID:   contact.ID,
		Message:     "Older completion",
		CompletedAt: time.Now().Add(-48 * time.Hour),
	}
	newer := models.ReminderCompletion{
		UserID:      user.ID,
		ContactID:   contact.ID,
		Message:     "Newer completion",
		CompletedAt: time.Now().Add(-1 * time.Hour),
	}
	db.Create(&older)
	db.Create(&newer)

	req, _ := http.NewRequest("GET", "/contacts/"+strconv.Itoa(int(contact.ID))+"/completions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	completions, ok := responseBody["completions"].([]any)
	if !ok {
		t.Fatalf("expected completions array in response")
	}
	assert.Len(t, completions, 2)
	// Ordered descending by completed_at: newer first
	assert.Equal(t, "Newer completion", completions[0].(map[string]any)["message"])
	assert.Equal(t, "Older completion", completions[1].(map[string]any)["message"])
}

// TestGetCompletionsForContactRejectsContactFromAnotherUser matches the
// established cross-user contact-ownership pattern used elsewhere in this
// package (see life_event_controller_test.go, tag_controller_test.go).
func TestGetCompletionsForContactRejectsContactFromAnotherUser(t *testing.T) {
	db, router := setupRouter()
	router.GET("/contacts/:id/completions", GetCompletionsForContact)

	otherUser := models.User{Username: "other", Password: "x", Email: "other@example.com"}
	require.NoError(t, db.Create(&otherUser).Error)
	othersContact := models.Contact{UserID: otherUser.ID, Firstname: "Not Yours"}
	db.Create(&othersContact)

	req, _ := http.NewRequest("GET", "/contacts/"+strconv.Itoa(int(othersContact.ID))+"/completions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteCompletion(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.DELETE("/completions/:id", DeleteCompletion)

	contact := models.Contact{UserID: user.ID, Firstname: "Amy", Lastname: "Lee"}
	db.Create(&contact)

	completion := models.ReminderCompletion{
		UserID:      user.ID,
		ContactID:   contact.ID,
		Message:     "Done",
		CompletedAt: time.Now(),
	}
	db.Create(&completion)

	req, _ := http.NewRequest("DELETE", "/completions/"+strconv.Itoa(int(completion.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]string
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, "Reminder completion deleted", responseBody["message"])

	var deleted models.ReminderCompletion
	result := db.First(&deleted, completion.ID)
	assert.Error(t, result.Error)
}

// TestDeleteCompletionRejectsCompletionFromAnotherUser is the ownership-boundary
// check called out explicitly for this work package: a user must not be able to
// delete another user's reminder completion record by guessing/enumerating its ID.
func TestDeleteCompletionRejectsCompletionFromAnotherUser(t *testing.T) {
	db, router := setupRouter()
	router.DELETE("/completions/:id", DeleteCompletion)

	otherUser := models.User{Username: "other", Password: "x", Email: "other@example.com"}
	require.NoError(t, db.Create(&otherUser).Error)
	othersContact := models.Contact{UserID: otherUser.ID, Firstname: "Not Yours"}
	db.Create(&othersContact)
	othersCompletion := models.ReminderCompletion{
		UserID:      otherUser.ID,
		ContactID:   othersContact.ID,
		Message:     "Their completion",
		CompletedAt: time.Now(),
	}
	require.NoError(t, db.Create(&othersCompletion).Error)

	req, _ := http.NewRequest("DELETE", "/completions/"+strconv.Itoa(int(othersCompletion.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	// The completion must remain untouched.
	var stillThere models.ReminderCompletion
	result := db.First(&stillThere, othersCompletion.ID)
	assert.NoError(t, result.Error)
}
