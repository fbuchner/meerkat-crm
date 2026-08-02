package controllers

import (
	"bytes"
	"encoding/json"
	"mycorrhizal/contactmodel"
	"mycorrhizal/models"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateLifeEvent(t *testing.T) {
	db, router := setupRouter()
	router.POST("/life-events", withValidated(func() any { return &models.LifeEventInput{} }), CreateLifeEvent)

	var user models.User
	db.First(&user)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)

	year := 2024
	payload := models.LifeEventInput{
		EntityID: contact.VCardUID, Type: models.LifeEventTypeGraduated,
		Date: &contactmodel.PartialDate{Year: &year}, Source: models.LifeEventSourceUser,
	}
	jsonValue, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/life-events", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var count int64
	db.Model(&models.LifeEvent{}).Count(&count)
	assert.EqualValues(t, 1, count)
}

func TestCreateLifeEventRejectsContactFromAnotherUser(t *testing.T) {
	db, router := setupRouter()
	router.POST("/life-events", withValidated(func() any { return &models.LifeEventInput{} }), CreateLifeEvent)

	var user models.User
	db.First(&user)
	otherUser := models.User{Username: "other", Password: "x", Email: "other@example.com"}
	require.NoError(t, db.Create(&otherUser).Error)
	othersContact := models.Contact{UserID: otherUser.ID, Firstname: "Not Yours"}
	db.Create(&othersContact)

	payload := models.LifeEventInput{EntityID: othersContact.VCardUID, Type: models.LifeEventTypeMoved}
	jsonValue, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/life-events", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetLifeEvent(t *testing.T) {
	db, router := setupRouter()
	router.GET("/life-events/:id", GetLifeEvent)

	var user models.User
	db.First(&user)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)
	event := models.LifeEvent{UserID: user.ID, EntityID: contact.VCardUID, Type: models.LifeEventTypeRetired}
	db.Create(&event)

	req, _ := http.NewRequest("GET", "/life-events/"+event.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListLifeEventsFiltersByEntityID(t *testing.T) {
	db, router := setupRouter()
	router.GET("/life-events", ListLifeEvents)

	var user models.User
	db.First(&user)
	alice := models.Contact{UserID: user.ID, Firstname: "Alice"}
	bob := models.Contact{UserID: user.ID, Firstname: "Bob"}
	db.Create(&alice)
	db.Create(&bob)
	db.Create(&models.LifeEvent{UserID: user.ID, EntityID: alice.VCardUID, Type: models.LifeEventTypeGraduated})
	db.Create(&models.LifeEvent{UserID: user.ID, EntityID: bob.VCardUID, Type: models.LifeEventTypeMoved})

	req, _ := http.NewRequest("GET", "/life-events?entity_id="+alice.VCardUID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Len(t, responseBody["life_events"], 1)
}

func TestUpdateLifeEvent(t *testing.T) {
	db, router := setupRouter()
	router.PUT("/life-events/:id", withValidated(func() any { return &models.LifeEventInput{} }), UpdateLifeEvent)

	var user models.User
	db.First(&user)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)
	event := models.LifeEvent{UserID: user.ID, EntityID: contact.VCardUID, Type: models.LifeEventTypeMoved}
	db.Create(&event)

	payload := models.LifeEventInput{EntityID: contact.VCardUID, Type: models.LifeEventTypeRetired, Description: "Retired from teaching"}
	jsonValue, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT", "/life-events/"+event.ID, bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var reloaded models.LifeEvent
	db.First(&reloaded, "id = ?", event.ID)
	assert.Equal(t, models.LifeEventTypeRetired, reloaded.Type)
	assert.Equal(t, "Retired from teaching", reloaded.Description)
}

func TestDeleteLifeEvent(t *testing.T) {
	db, router := setupRouter()
	router.DELETE("/life-events/:id", DeleteLifeEvent)

	var user models.User
	db.First(&user)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)
	event := models.LifeEvent{UserID: user.ID, EntityID: contact.VCardUID, Type: models.LifeEventTypeMoved}
	db.Create(&event)

	req, _ := http.NewRequest("DELETE", "/life-events/"+event.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var count int64
	db.Model(&models.LifeEvent{}).Count(&count)
	assert.Zero(t, count)

	// M2: affirm this was a soft-delete, not a hard-delete.
	var unscopedCount int64
	db.Unscoped().Model(&models.LifeEvent{}).Where("id = ?", event.ID).Count(&unscopedCount)
	assert.EqualValues(t, 1, unscopedCount)
}

// M1: Creating a LifeEvent with Remind=true and month/day creates a
// materialised yearly Reminder row.
func TestCreateLifeEventWithRemind(t *testing.T) {
	db, router := setupRouter()

	router.POST("/life-events", withValidated(func() any { return &models.LifeEventInput{} }), CreateLifeEvent)

	var user models.User
	db.First(&user)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)

	month := 8
	day := 15
	payload := models.LifeEventInput{
		EntityID: contact.VCardUID,
		Type:     models.LifeEventTypeMoved,
		Date:     &contactmodel.PartialDate{Month: &month, Day: &day},
		Remind:   true,
	}

	jsonValue, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/life-events", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var responseBody struct {
		LifeEvent models.LifeEvent `json:"life_event"`
	}
	json.Unmarshal(w.Body.Bytes(), &responseBody)

	var count int64
	db.Model(&models.Reminder{}).Unscoped().Where("life_event_id = ?", responseBody.LifeEvent.ID).Count(&count)
	assert.EqualValues(t, 1, count)

	var reminder models.Reminder
	db.Where("life_event_id = ?", responseBody.LifeEvent.ID).First(&reminder)
	assert.Equal(t, "yearly", reminder.Recurrence)
	assert.NotNil(t, reminder.ContactID)
}

// M1b: Creating a LifeEvent with Remind=true but year-only date does NOT
// create a reminder (no month/day to anchor recurrence).
func TestCreateLifeEventWithRemindYearOnly(t *testing.T) {
	db, router := setupRouter()

	router.POST("/life-events", withValidated(func() any { return &models.LifeEventInput{} }), CreateLifeEvent)

	var user models.User
	db.First(&user)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)

	year := 1985
	payload := models.LifeEventInput{
		EntityID: contact.VCardUID,
		Type:     models.LifeEventTypeMarried,
		Date:     &contactmodel.PartialDate{Year: &year},
		Remind:   true,
	}

	jsonValue, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/life-events", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var responseBody struct {
		LifeEvent models.LifeEvent `json:"life_event"`
	}
	json.Unmarshal(w.Body.Bytes(), &responseBody)

	var count int64
	db.Model(&models.Reminder{}).Unscoped().Where("life_event_id = ?", responseBody.LifeEvent.ID).Count(&count)
	assert.Zero(t, count, "year-only events must not produce a reminder")
}

// M1c: Toggling Remind from true→false deletes the reminder.
func TestUpdateLifeEventRemindToggleOff(t *testing.T) {
	db, router := setupRouter()

	router.PUT("/life-events/:id", withValidated(func() any { return &models.LifeEventInput{} }), UpdateLifeEvent)

	var user models.User
	db.First(&user)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)

	month := 3
	day := 14
	event := models.LifeEvent{
		UserID:   user.ID,
		EntityID: contact.VCardUID,
		Type:     models.LifeEventTypeJobChange,
		Date:     &contactmodel.PartialDate{Month: &month, Day: &day},
		Remind:   true,
	}
	db.Create(&event)

	remindAt := time.Date(2026, 3, 14, 9, 0, 0, 0, time.UTC)
	db.Create(&models.Reminder{
		UserID:      user.ID,
		Message:     "test",
		RemindAt:    remindAt,
		Recurrence:  "yearly",
		ContactID:   &contact.ID,
		LifeEventID: &event.ID,
	})

	payload := models.LifeEventInput{
		EntityID: contact.VCardUID,
		Type:     models.LifeEventTypeJobChange,
		Date:     &contactmodel.PartialDate{Month: &month, Day: &day},
		Remind:   false,
	}
	jsonValue, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT", "/life-events/"+event.ID, bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var count int64
	db.Model(&models.Reminder{}).Unscoped().Where("life_event_id = ?", event.ID).Count(&count)
	assert.Zero(t, count, "reminder must be hard-deleted when Remind is turned off")
}

// M1d: Impossible calendar dates are rejected by eventHasMonthDay.
func TestEventHasMonthDayRejectsInvalidDates(t *testing.T) {
	// April 31 never exists.
	apr31 := &contactmodel.PartialDate{Month: intPtr(4), Day: intPtr(31)}
	assert.False(t, eventHasMonthDay(apr31), "April 31 is not a real date")

	// Feb 30 never exists.
	feb30 := &contactmodel.PartialDate{Month: intPtr(2), Day: intPtr(30)}
	assert.False(t, eventHasMonthDay(feb30), "Feb 30 is not a real date")

	// Feb 29 IS valid (leap year exists). The check uses a fixed leap year
	// (2000) so Feb 29 passes validation — correct: the user can enter it.
	feb29 := &contactmodel.PartialDate{Month: intPtr(2), Day: intPtr(29)}
	assert.True(t, eventHasMonthDay(feb29), "Feb 29 is valid in leap years")

	// Normal valid dates pass.
	mar15 := &contactmodel.PartialDate{Month: intPtr(3), Day: intPtr(15)}
	assert.True(t, eventHasMonthDay(mar15), "March 15 is valid")
}
