package controllers

import (
	"bytes"
	"encoding/json"
	"mycorrhizal/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTag(t *testing.T) {
	db, router := setupRouter()
	router.POST("/tags", withValidated(func() any { return &models.TagInput{} }), CreateTag)

	payload := models.TagInput{Name: "poly"}
	jsonValue, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/tags", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var count int64
	db.Model(&models.Tag{}).Count(&count)
	assert.EqualValues(t, 1, count)
}

func TestGetTagIncludesTaggedContacts(t *testing.T) {
	db, router := setupRouter()
	router.GET("/tags/:id", GetTag)

	var user models.User
	db.First(&user)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)
	tag := models.Tag{UserID: user.ID, Name: "poly"}
	db.Create(&tag)
	db.Create(&models.ContactTag{TagID: tag.ID, UserID: user.ID, ContactVCardUID: contact.VCardUID})

	req, _ := http.NewRequest("GET", "/tags/"+tag.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Len(t, responseBody["contacts"], 1)
}

func TestListTags(t *testing.T) {
	db, router := setupRouter()
	router.GET("/tags", ListTags)

	var user models.User
	db.First(&user)
	db.Create(&models.Tag{UserID: user.ID, Name: "poly"})
	db.Create(&models.Tag{UserID: user.ID, Name: "SWer"})

	req, _ := http.NewRequest("GET", "/tags", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Len(t, responseBody["tags"], 2)
	assert.EqualValues(t, 2, responseBody["total"])
}

func TestUpdateTag(t *testing.T) {
	db, router := setupRouter()
	router.PUT("/tags/:id", withValidated(func() any { return &models.TagInput{} }), UpdateTag)

	var user models.User
	db.First(&user)
	tag := models.Tag{UserID: user.ID, Name: "old"}
	db.Create(&tag)

	payload := models.TagInput{Name: "new"}
	jsonValue, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT", "/tags/"+tag.ID, bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var reloaded models.Tag
	db.First(&reloaded, "id = ?", tag.ID)
	assert.Equal(t, "new", reloaded.Name)
}

func TestDeleteTagCascadesTaggings(t *testing.T) {
	db, router := setupRouter()
	router.DELETE("/tags/:id", DeleteTag)

	var user models.User
	db.First(&user)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)
	tag := models.Tag{UserID: user.ID, Name: "poly"}
	db.Create(&tag)
	db.Create(&models.ContactTag{TagID: tag.ID, UserID: user.ID, ContactVCardUID: contact.VCardUID})

	req, _ := http.NewRequest("DELETE", "/tags/"+tag.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var count int64
	db.Model(&models.Tag{}).Count(&count)
	assert.Zero(t, count)
}

func TestAddContactTag(t *testing.T) {
	db, router := setupRouter()
	router.POST("/tags/:id/contacts", withValidated(func() any { return &models.ContactTagInput{} }), AddContactTag)

	var user models.User
	db.First(&user)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)
	tag := models.Tag{UserID: user.ID, Name: "poly"}
	db.Create(&tag)

	payload := models.ContactTagInput{ContactVCardUID: contact.VCardUID}
	jsonValue, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/tags/"+tag.ID+"/contacts", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var count int64
	db.Model(&models.ContactTag{}).Count(&count)
	assert.EqualValues(t, 1, count)
}

func TestAddContactTagRejectsDuplicate(t *testing.T) {
	db, router := setupRouter()
	router.POST("/tags/:id/contacts", withValidated(func() any { return &models.ContactTagInput{} }), AddContactTag)

	var user models.User
	db.First(&user)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)
	tag := models.Tag{UserID: user.ID, Name: "poly"}
	db.Create(&tag)
	db.Create(&models.ContactTag{TagID: tag.ID, UserID: user.ID, ContactVCardUID: contact.VCardUID})

	payload := models.ContactTagInput{ContactVCardUID: contact.VCardUID}
	jsonValue, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/tags/"+tag.ID+"/contacts", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestAddContactTagRejectsContactFromAnotherUser(t *testing.T) {
	db, router := setupRouter()
	router.POST("/tags/:id/contacts", withValidated(func() any { return &models.ContactTagInput{} }), AddContactTag)

	var user models.User
	db.First(&user)
	otherUser := models.User{Username: "other", Password: "x", Email: "other@example.com"}
	require.NoError(t, db.Create(&otherUser).Error)
	othersContact := models.Contact{UserID: otherUser.ID, Firstname: "Not Yours"}
	db.Create(&othersContact)
	tag := models.Tag{UserID: user.ID, Name: "poly"}
	db.Create(&tag)

	payload := models.ContactTagInput{ContactVCardUID: othersContact.VCardUID}
	jsonValue, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/tags/"+tag.ID+"/contacts", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRemoveContactTag(t *testing.T) {
	db, router := setupRouter()
	router.DELETE("/tags/:id/contacts/:vcard_uid", RemoveContactTag)

	var user models.User
	db.First(&user)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)
	tag := models.Tag{UserID: user.ID, Name: "poly"}
	db.Create(&tag)
	db.Create(&models.ContactTag{TagID: tag.ID, UserID: user.ID, ContactVCardUID: contact.VCardUID})

	req, _ := http.NewRequest("DELETE", "/tags/"+tag.ID+"/contacts/"+contact.VCardUID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var count int64
	db.Model(&models.ContactTag{}).Count(&count)
	assert.Zero(t, count)
}
