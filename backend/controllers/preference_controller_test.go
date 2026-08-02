package controllers

import (
	"bytes"
	"encoding/json"
	"mycorrhizal/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func registerPreferenceRoutes(t *testing.T, router *gin.Engine) {
	router.POST("/preferences", withValidated(func() any { return &models.PreferenceInput{} }), CreatePreference)
	router.GET("/preferences", ListPreferences)
	router.GET("/preferences/:id", GetPreference)
	router.PUT("/preferences/:id", withValidated(func() any { return &models.PreferenceInput{} }), UpdatePreference)
	router.DELETE("/preferences/:id", DeletePreference)
}

func seedPreferenceContact(t *testing.T, db *gorm.DB, userID uint) models.Contact {
	t.Helper()
	contact := models.Contact{UserID: userID, Firstname: "Alice"}
	require.NoError(t, db.Create(&contact).Error)
	return contact
}

func TestCreatePreference(t *testing.T) {
	db, router := setupRouter()
	registerPreferenceRoutes(t, router)

	var user models.User
	db.First(&user)
	contact := seedPreferenceContact(t, db, user.ID)

	payload := models.PreferenceInput{
		EntityID: contact.VCardUID,
		Category: models.PreferenceCategoryFood,
		Value:    "Vegetarian",
	}
	jsonValue, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/preferences", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var count int64
	db.Model(&models.Preference{}).Count(&count)
	assert.EqualValues(t, 1, count)

	var pref models.Preference
	db.First(&pref)
	assert.Equal(t, models.RelationshipSensitivityNormal, pref.Sensitivity, "omitted sensitivity must default to normal")
	assert.Equal(t, models.PreferenceCategoryFood, pref.Category)
}

func TestCreatePreferenceRejectsContactFromAnotherUser(t *testing.T) {
	db, router := setupRouter()
	registerPreferenceRoutes(t, router)

	var user models.User
	db.First(&user)
	otherUser := models.User{Username: "other", Password: "x", Email: "other@example.com"}
	require.NoError(t, db.Create(&otherUser).Error)
	othersContact := seedPreferenceContact(t, db, otherUser.ID)

	payload := models.PreferenceInput{
		EntityID: othersContact.VCardUID,
		Category: models.PreferenceCategoryFood,
		Value:    "Vegan",
	}
	jsonValue, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/preferences", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var count int64
	db.Model(&models.Preference{}).Count(&count)
	assert.Zero(t, count)
}

func TestGetPreference(t *testing.T) {
	db, router := setupRouter()
	registerPreferenceRoutes(t, router)

	var user models.User
	db.First(&user)
	contact := seedPreferenceContact(t, db, user.ID)
	pref := models.Preference{
		UserID: user.ID, EntityID: contact.VCardUID, Category: models.PreferenceCategoryHobby,
		Value: "Photography", Sensitivity: models.RelationshipSensitivityNormal,
	}
	require.NoError(t, db.Create(&pref).Error)

	req, _ := http.NewRequest("GET", "/preferences/"+pref.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var got models.Preference
	json.Unmarshal(w.Body.Bytes(), &got)
	assert.Equal(t, pref.ID, got.ID)
	assert.Equal(t, "Photography", got.Value)
}

func TestGetPreferenceRejectsAnotherUsersPreference(t *testing.T) {
	db, router := setupRouter()
	registerPreferenceRoutes(t, router)

	var user models.User
	db.First(&user)
	otherUser := models.User{Username: "other", Password: "x", Email: "other@example.com"}
	require.NoError(t, db.Create(&otherUser).Error)
	othersContact := seedPreferenceContact(t, db, otherUser.ID)
	othersPref := models.Preference{
		UserID: otherUser.ID, EntityID: othersContact.VCardUID, Category: models.PreferenceCategoryHobby,
		Value: "Knitting", Sensitivity: models.RelationshipSensitivityNormal,
	}
	require.NoError(t, db.Create(&othersPref).Error)

	req, _ := http.NewRequest("GET", "/preferences/"+othersPref.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListPreferencesFiltersByEntity(t *testing.T) {
	db, router := setupRouter()
	registerPreferenceRoutes(t, router)

	var user models.User
	db.First(&user)
	contactA := seedPreferenceContact(t, db, user.ID)
	contactB := models.Contact{UserID: user.ID, Firstname: "Bob"}
	require.NoError(t, db.Create(&contactB).Error)

	db.Create(&models.Preference{UserID: user.ID, EntityID: contactA.VCardUID, Category: models.PreferenceCategoryFood, Value: "Vegetarian", Sensitivity: models.RelationshipSensitivityNormal})
	db.Create(&models.Preference{UserID: user.ID, EntityID: contactA.VCardUID, Category: models.PreferenceCategoryHobby, Value: "Hiking", Sensitivity: models.RelationshipSensitivityNormal})
	db.Create(&models.Preference{UserID: user.ID, EntityID: contactB.VCardUID, Category: models.PreferenceCategoryHobby, Value: "Golf", Sensitivity: models.RelationshipSensitivityNormal})

	req, _ := http.NewRequest("GET", "/preferences?entity_id="+contactA.VCardUID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var responseBody struct {
		Preferences []models.Preference `json:"preferences"`
		Total       int64               `json:"total"`
	}
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.EqualValues(t, 2, responseBody.Total)
}

func TestUpdatePreference(t *testing.T) {
	db, router := setupRouter()
	registerPreferenceRoutes(t, router)

	var user models.User
	db.First(&user)
	contact := seedPreferenceContact(t, db, user.ID)
	pref := models.Preference{
		UserID: user.ID, EntityID: contact.VCardUID, Category: models.PreferenceCategoryFood,
		Value: "Vegetarian", Sensitivity: models.RelationshipSensitivityNormal,
	}
	require.NoError(t, db.Create(&pref).Error)

	payload := models.PreferenceInput{
		EntityID:    contact.VCardUID,
		Category:    models.PreferenceCategoryFood,
		Value:       "Vegan",
		Sensitivity: "private",
	}
	jsonValue, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT", "/preferences/"+pref.ID, bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var reloaded models.Preference
	db.First(&reloaded, "id = ?", pref.ID)
	assert.Equal(t, "Vegan", reloaded.Value)
	assert.Equal(t, "private", reloaded.Sensitivity)
}

func TestDeletePreference(t *testing.T) {
	db, router := setupRouter()
	registerPreferenceRoutes(t, router)

	var user models.User
	db.First(&user)
	contact := seedPreferenceContact(t, db, user.ID)
	pref := models.Preference{
		UserID: user.ID, EntityID: contact.VCardUID, Category: models.PreferenceCategoryHobby,
		Value: "Origami", Sensitivity: models.RelationshipSensitivityNormal,
	}
	require.NoError(t, db.Create(&pref).Error)

	req, _ := http.NewRequest("DELETE", "/preferences/"+pref.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var count int64
	db.Model(&models.Preference{}).Count(&count)
	assert.Zero(t, count)
}
