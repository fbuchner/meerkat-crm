package controllers

import (
	"bytes"
	"encoding/json"
	"mycorrhizal/contactmodel"
	"mycorrhizal/models"
	"net/http"
	"net/http/httptest"
	"testing"

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

	assert.Equal(t, http.StatusOK, w.Code)

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
}
