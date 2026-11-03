package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"meerkat/config"
	"meerkat/middleware"
	"meerkat/models"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
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

type reminderFixture struct {
	db       *gorm.DB
	router   *gin.Engine
	user     models.User
	stranger models.User
	contact  models.Contact
	theirs   models.Contact
}

func setupReminders(t *testing.T) reminderFixture {
	t.Helper()
	db := newTestDB(t)
	user := newTestUser(t, db, "owner", false)
	stranger := newTestUser(t, db, "stranger", false)
	router := routerAs(db, user.ID, config.Config{})

	validated := middleware.ValidateJSONMiddleware(&models.Reminder{})
	router.GET("/reminders", GetAllReminders)
	router.GET("/reminders/upcoming", GetUpcomingReminders)
	router.GET("/contacts/:id/reminders", GetRemindersForContact)
	router.POST("/contacts/:id/reminders", validated, CreateReminder)
	router.GET("/reminders/:id", GetReminder)
	router.PUT("/reminders/:id", validated, UpdateReminder)
	router.POST("/reminders/:id/complete", CompleteReminder)
	router.DELETE("/reminders/:id", DeleteReminder)
	router.GET("/contacts/:id/reminder-completions", GetCompletionsForContact)
	router.DELETE("/reminder-completions/:id", DeleteCompletion)

	return reminderFixture{
		db: db, router: router, user: user, stranger: stranger,
		contact: newTestContact(t, db, user.ID, "Mine"),
		theirs:  newTestContact(t, db, stranger.ID, "Theirs"),
	}
}

func (f reminderFixture) newReminder(t *testing.T, change func(*models.Reminder)) models.Reminder {
	t.Helper()
	reminder := models.Reminder{UserID: f.user.ID, ContactID: &f.contact.ID, Message: "Call", Recurrence: "weekly", RemindAt: time.Now().AddDate(0, 0, -1)}
	if change != nil {
		change(&reminder)
	}
	require.NoError(t, f.db.Create(&reminder).Error)
	return reminder
}

func (f reminderFixture) theirReminder(t *testing.T) models.Reminder {
	t.Helper()
	return f.newReminder(t, func(r *models.Reminder) {
		r.UserID = f.stranger.ID
		r.ContactID = &f.theirs.ID
		r.Message = "Their secret"
	})
}

func (f reminderFixture) reload(t *testing.T, id uint) models.Reminder {
	t.Helper()
	var reminder models.Reminder
	require.NoError(t, f.db.First(&reminder, id).Error)
	return reminder
}

func (f reminderFixture) completions(t *testing.T, userID uint) []models.ReminderCompletion {
	t.Helper()
	var completions []models.ReminderCompletion
	require.NoError(t, f.db.Where("user_id = ?", userID).Find(&completions).Error)
	return completions
}

func todayUTC() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func TestCompleteReminder_OnceIsDeletedAndRecorded(t *testing.T) {
	f := setupReminders(t)
	reminder := f.newReminder(t, func(r *models.Reminder) { r.Recurrence = "once" })

	w := doRequest(f.router, http.MethodPost, fmt.Sprintf("/reminders/%d/complete", reminder.ID), nil, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var gone models.Reminder
	assert.ErrorIs(t, f.db.First(&gone, reminder.ID).Error, gorm.ErrRecordNotFound)
	completions := f.completions(t, f.user.ID)
	require.Len(t, completions, 1)
	assert.Equal(t, "Call", completions[0].Message)
	assert.Equal(t, f.contact.ID, completions[0].ContactID)
	assert.Equal(t, reminder.ID, *completions[0].ReminderID)
}

func TestCompleteReminder_SkippingOnceDeletesWithoutATimelineEntry(t *testing.T) {
	f := setupReminders(t)
	reminder := f.newReminder(t, func(r *models.Reminder) { r.Recurrence = "once" })

	w := doRequest(f.router, http.MethodPost, fmt.Sprintf("/reminders/%d/complete?skip=true", reminder.ID), nil, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "skipped")

	var gone models.Reminder
	assert.ErrorIs(t, f.db.First(&gone, reminder.ID).Error, gorm.ErrRecordNotFound)
	assert.Empty(t, f.completions(t, f.user.ID))
}

func TestCompleteReminder_OverdueRecurringIsRescheduledFromToday(t *testing.T) {
	for recurrence, next := range map[string]func(time.Time) time.Time{
		"weekly": func(d time.Time) time.Time { return d.AddDate(0, 0, 7) },
		"yearly": func(d time.Time) time.Time { return d.AddDate(1, 0, 0) },
	} {
		t.Run(recurrence, func(t *testing.T) {
			f := setupReminders(t)
			reminder := f.newReminder(t, func(r *models.Reminder) {
				r.Recurrence = recurrence
				r.EmailSent = true
				r.RemindAt = time.Now().AddDate(0, 0, -30)
			})
			today := todayUTC()

			w := doRequest(f.router, http.MethodPost, fmt.Sprintf("/reminders/%d/complete", reminder.ID), nil, "")
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())

			reloaded := f.reload(t, reminder.ID)
			assert.False(t, reloaded.Completed, "a recurring reminder stays active")
			assert.False(t, reloaded.EmailSent, "the next occurrence must be emailed again")
			assert.NotNil(t, reloaded.LastSent)
			assert.True(t, next(today).Equal(reloaded.RemindAt.UTC()), "got %s", reloaded.RemindAt)
			assert.Len(t, f.completions(t, f.user.ID), 1)
		})
	}
}

func TestCompleteReminder_EarlyCompletionKeepsTheSchedule(t *testing.T) {
	f := setupReminders(t)
	due := todayUTC().AddDate(0, 0, 3)
	reminder := f.newReminder(t, func(r *models.Reminder) { r.RemindAt = due })

	w := doRequest(f.router, http.MethodPost, fmt.Sprintf("/reminders/%d/complete", reminder.ID), nil, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	assert.True(t, due.AddDate(0, 0, 7).Equal(f.reload(t, reminder.ID).RemindAt.UTC()))
}

func TestCompleteReminder_SkippingRecurringReschedulesWithoutATimelineEntry(t *testing.T) {
	f := setupReminders(t)
	reminder := f.newReminder(t, nil)
	today := todayUTC()

	w := doRequest(f.router, http.MethodPost, fmt.Sprintf("/reminders/%d/complete?skip=true", reminder.ID), nil, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	reloaded := f.reload(t, reminder.ID)
	assert.False(t, reloaded.Completed)
	assert.True(t, today.AddDate(0, 0, 7).Equal(reloaded.RemindAt.UTC()))
	assert.Empty(t, f.completions(t, f.user.ID))
}

func TestCompleteReminder_RecurringNotFromCompletionKeepsRecurring(t *testing.T) {
	f := setupReminders(t)
	notFromCompletion := false
	due := todayUTC().AddDate(0, 0, -2)
	reminder := f.newReminder(t, func(r *models.Reminder) {
		r.RemindAt = due
		r.ReoccurFromCompletion = &notFromCompletion
	})

	w := doRequest(f.router, http.MethodPost, fmt.Sprintf("/reminders/%d/complete", reminder.ID), nil, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	reloaded := f.reload(t, reminder.ID)
	assert.False(t, reloaded.Completed, "a weekly reminder must come back next week")
	assert.True(t, due.AddDate(0, 0, 7).Equal(reloaded.RemindAt.UTC()), "scheduled from the original date, got %s", reloaded.RemindAt)
}

func TestCompleteReminder_SeveralPeriodsOverdueIsNoLongerDue(t *testing.T) {
	f := setupReminders(t)
	notFromCompletion := false
	due := todayUTC().AddDate(0, 0, -22)
	reminder := f.newReminder(t, func(r *models.Reminder) {
		r.RemindAt = due
		r.ReoccurFromCompletion = &notFromCompletion
	})

	w := doRequest(f.router, http.MethodPost, fmt.Sprintf("/reminders/%d/complete", reminder.ID), nil, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	reloaded := f.reload(t, reminder.ID)
	assert.True(t, due.AddDate(0, 0, 28).Equal(reloaded.RemindAt.UTC()), "the first date on the schedule after today, got %s", reloaded.RemindAt)
	assert.False(t, reloaded.EmailSent)
}

func TestCreateReminder_BodyCannotSetServerState(t *testing.T) {
	f := setupReminders(t)

	w := doJSON(t, f.router, http.MethodPost, fmt.Sprintf("/contacts/%d/reminders", f.contact.ID), gin.H{
		"message": "Call", "recurrence": "monthly", "contact_id": f.contact.ID, "remind_at": "2030-05-17T00:00:00Z",
		"ID": 999999, "completed": true, "email_sent": true,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var created models.Reminder
	require.NoError(t, f.db.First(&created).Error)
	assert.NotEqual(t, uint(999999), created.ID)
	assert.False(t, created.Completed)
	assert.False(t, created.EmailSent)
}

func TestCreateReminder_NormalizesDateAndForcesOwner(t *testing.T) {
	f := setupReminders(t)

	w := doJSON(t, f.router, http.MethodPost, fmt.Sprintf("/contacts/%d/reminders", f.contact.ID), gin.H{
		"message": "Call", "recurrence": "monthly", "contact_id": f.contact.ID,
		"remind_at": "2030-05-17T15:45:10Z", "user_id": f.stranger.ID,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var created models.Reminder
	require.NoError(t, f.db.First(&created).Error)
	assert.Equal(t, f.user.ID, created.UserID)
	assert.Equal(t, f.contact.ID, *created.ContactID)
	assert.True(t, time.Date(2030, 5, 17, 0, 0, 0, 0, time.UTC).Equal(created.RemindAt), "time of day is dropped, got %s", created.RemindAt)
}

func TestCreateReminder_BodyCannotRetargetTheContact(t *testing.T) {
	f := setupReminders(t)

	w := doJSON(t, f.router, http.MethodPost, fmt.Sprintf("/contacts/%d/reminders", f.contact.ID), gin.H{
		"message": "Call", "recurrence": "monthly", "contact_id": f.contact.ID, "remind_at": "2030-05-17T00:00:00Z",
		"contact": gin.H{"ID": f.theirs.ID, "firstname": "Hijacked"},
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var created models.Reminder
	require.NoError(t, f.db.First(&created).Error)
	assert.Equal(t, f.contact.ID, *created.ContactID, "the reminder must stay on the contact in the URL")
}

func TestCreateReminder_Rejections(t *testing.T) {
	f := setupReminders(t)
	valid := gin.H{"message": "Call", "recurrence": "monthly", "contact_id": f.contact.ID, "remind_at": "2030-05-17T00:00:00Z"}

	assert.Equal(t, http.StatusNotFound, doJSON(t, f.router, http.MethodPost, fmt.Sprintf("/contacts/%d/reminders", f.theirs.ID), valid).Code, "another user's contact")
	assert.Equal(t, http.StatusBadRequest, doJSON(t, f.router, http.MethodPost, fmt.Sprintf("/contacts/%d/reminders", f.contact.ID), gin.H{
		"message": "Call", "recurrence": "fortnightly", "contact_id": f.contact.ID, "remind_at": "2030-05-17T00:00:00Z",
	}).Code, "unknown recurrence")
	assert.Equal(t, http.StatusBadRequest, doJSON(t, f.router, http.MethodPost, fmt.Sprintf("/contacts/%d/reminders", f.contact.ID), gin.H{
		"recurrence": "monthly", "contact_id": f.contact.ID, "remind_at": "2030-05-17T00:00:00Z",
	}).Code, "missing message")

	var count int64
	require.NoError(t, f.db.Model(&models.Reminder{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestUpdateReminder_ChangesFieldsAndContact(t *testing.T) {
	f := setupReminders(t)
	second := newTestContact(t, f.db, f.user.ID, "Second")
	reminder := f.newReminder(t, nil)

	w := doJSON(t, f.router, http.MethodPut, fmt.Sprintf("/reminders/%d", reminder.ID), gin.H{
		"message": "Visit", "recurrence": "yearly", "contact_id": second.ID, "remind_at": "2031-01-02T09:30:00Z", "by_mail": true,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	reloaded := f.reload(t, reminder.ID)
	assert.Equal(t, "Visit", reloaded.Message)
	assert.Equal(t, "yearly", reloaded.Recurrence)
	assert.Equal(t, second.ID, *reloaded.ContactID)
	assert.True(t, *reloaded.ByMail)
	assert.True(t, time.Date(2031, 1, 2, 0, 0, 0, 0, time.UTC).Equal(reloaded.RemindAt))
}

func TestUpdateReminder_CannotMoveToAnotherUsersContact(t *testing.T) {
	f := setupReminders(t)
	reminder := f.newReminder(t, nil)

	w := doJSON(t, f.router, http.MethodPut, fmt.Sprintf("/reminders/%d", reminder.ID), gin.H{
		"message": "Moved", "recurrence": "yearly", "contact_id": f.theirs.ID, "remind_at": "2031-01-02T00:00:00Z",
	})

	assert.Equal(t, http.StatusNotFound, w.Code)
	reloaded := f.reload(t, reminder.ID)
	assert.Equal(t, f.contact.ID, *reloaded.ContactID)
	assert.Equal(t, "Call", reloaded.Message)
}

func TestReminders_AnotherUsersReminderIsNotFound(t *testing.T) {
	f := setupReminders(t)
	reminder := f.theirReminder(t)
	path := fmt.Sprintf("/reminders/%d", reminder.ID)
	update := gin.H{"message": "Hacked", "recurrence": "yearly", "contact_id": f.contact.ID, "remind_at": "2031-01-02T00:00:00Z"}

	for name, w := range map[string]*httptest.ResponseRecorder{
		"get":      doRequest(f.router, http.MethodGet, path, nil, ""),
		"update":   doJSON(t, f.router, http.MethodPut, path, update),
		"complete": doRequest(f.router, http.MethodPost, path+"/complete", nil, ""),
		"delete":   doRequest(f.router, http.MethodDelete, path, nil, ""),
	} {
		assert.Equal(t, http.StatusNotFound, w.Code, name)
		assert.NotContains(t, w.Body.String(), "Their secret", name)
	}

	reloaded := f.reload(t, reminder.ID)
	assert.Equal(t, "Their secret", reloaded.Message)
	assert.Equal(t, f.theirs.ID, *reloaded.ContactID)
	assert.True(t, reminder.RemindAt.Equal(reloaded.RemindAt), "completing must not reschedule it")
	assert.Empty(t, f.completions(t, f.user.ID))
	assert.Empty(t, f.completions(t, f.stranger.ID))
}

func TestReminderLists_AreScopedToTheUser(t *testing.T) {
	f := setupReminders(t)
	later := f.newReminder(t, func(r *models.Reminder) { r.Message = "later"; r.RemindAt = time.Now().AddDate(0, 0, 2) })
	sooner := f.newReminder(t, func(r *models.Reminder) { r.Message = "sooner"; r.RemindAt = time.Now().AddDate(0, 0, 1) })
	f.theirReminder(t)

	w := doRequest(f.router, http.MethodGet, "/reminders", nil, "")
	require.Equal(t, http.StatusOK, w.Code)
	all := decodeJSON[struct{ Reminders []models.Reminder }](t, w).Reminders
	require.Len(t, all, 2)
	assert.Equal(t, sooner.ID, all[0].ID, "ordered by due date")
	assert.Equal(t, later.ID, all[1].ID)

	w = doRequest(f.router, http.MethodGet, fmt.Sprintf("/contacts/%d/reminders", f.contact.ID), nil, "")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, decodeJSON[struct{ Reminders []models.Reminder }](t, w).Reminders, 2)

	w = doRequest(f.router, http.MethodGet, fmt.Sprintf("/contacts/%d/reminders", f.theirs.ID), nil, "")
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NotContains(t, w.Body.String(), "Their secret")
}

func TestGetUpcomingReminders(t *testing.T) {
	upcoming := func(t *testing.T, f reminderFixture) []models.Reminder {
		w := doRequest(f.router, http.MethodGet, "/reminders/upcoming", nil, "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		return decodeJSON[struct{ Reminders []models.Reminder }](t, w).Reminders
	}

	t.Run("more than five due soon returns all of them", func(t *testing.T) {
		f := setupReminders(t)
		for i := range 7 {
			f.newReminder(t, func(r *models.Reminder) { r.RemindAt = time.Now().AddDate(0, 0, i-1) })
		}
		f.newReminder(t, func(r *models.Reminder) { r.Message = "far"; r.RemindAt = time.Now().AddDate(1, 0, 0) })
		f.theirReminder(t)

		reminders := upcoming(t, f)
		assert.Len(t, reminders, 7)
		for _, reminder := range reminders {
			assert.NotEqual(t, "far", reminder.Message)
			assert.Equal(t, "Call", reminder.Message)
		}
	})

	t.Run("few due soon are topped up to five, without completed ones", func(t *testing.T) {
		f := setupReminders(t)
		f.newReminder(t, func(r *models.Reminder) { r.RemindAt = time.Now().AddDate(0, 0, 1) })
		for i := range 6 {
			f.newReminder(t, func(r *models.Reminder) { r.RemindAt = time.Now().AddDate(0, 1, i) })
		}
		f.newReminder(t, func(r *models.Reminder) { r.Message = "done"; r.Completed = true; r.RemindAt = time.Now() })
		f.theirReminder(t)

		reminders := upcoming(t, f)
		require.Len(t, reminders, 5)
		for _, reminder := range reminders {
			assert.Equal(t, "Call", reminder.Message)
		}
		assert.True(t, reminders[0].RemindAt.Before(reminders[1].RemindAt))
	})
}

func TestReminderCompletions_ListAndDelete(t *testing.T) {
	f := setupReminders(t)
	older := models.ReminderCompletion{UserID: f.user.ID, ContactID: f.contact.ID, Message: "older", CompletedAt: time.Now().AddDate(0, 0, -2)}
	newer := models.ReminderCompletion{UserID: f.user.ID, ContactID: f.contact.ID, Message: "newer", CompletedAt: time.Now()}
	otherContact := models.ReminderCompletion{UserID: f.user.ID, ContactID: f.contact.ID + 100, Message: "elsewhere", CompletedAt: time.Now()}
	theirs := models.ReminderCompletion{UserID: f.stranger.ID, ContactID: f.theirs.ID, Message: "Their secret", CompletedAt: time.Now()}
	for _, completion := range []*models.ReminderCompletion{&older, &newer, &otherContact, &theirs} {
		require.NoError(t, f.db.Create(completion).Error)
	}

	w := doRequest(f.router, http.MethodGet, fmt.Sprintf("/contacts/%d/reminder-completions", f.contact.ID), nil, "")
	require.Equal(t, http.StatusOK, w.Code)
	listed := decodeJSON[struct{ Completions []models.ReminderCompletion }](t, w).Completions
	require.Len(t, listed, 2)
	assert.Equal(t, "newer", listed[0].Message, "newest first")
	assert.Equal(t, "older", listed[1].Message)

	w = doRequest(f.router, http.MethodGet, fmt.Sprintf("/contacts/%d/reminder-completions", f.theirs.ID), nil, "")
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NotContains(t, w.Body.String(), "Their secret")

	assert.Equal(t, http.StatusNotFound, doRequest(f.router, http.MethodDelete, fmt.Sprintf("/reminder-completions/%d", theirs.ID), nil, "").Code)
	assert.Len(t, f.completions(t, f.stranger.ID), 1, "another user's completion must survive")

	assert.Equal(t, http.StatusOK, doRequest(f.router, http.MethodDelete, fmt.Sprintf("/reminder-completions/%d", older.ID), nil, "").Code)
	assert.Len(t, f.completions(t, f.user.ID), 2)
	assert.Equal(t, http.StatusNotFound, doRequest(f.router, http.MethodDelete, fmt.Sprintf("/reminder-completions/%d", older.ID), nil, "").Code, "already deleted")
}
