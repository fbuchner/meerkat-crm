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

func TestCreateCircle(t *testing.T) {
	db, router := setupRouter()
	router.POST("/circles", withValidated(func() any { return &models.CircleInput{} }), CreateCircle)

	payload := models.CircleInput{Name: "College friends"}
	jsonValue, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/circles", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var count int64
	db.Model(&models.Circle{}).Count(&count)
	assert.EqualValues(t, 1, count)
}

func TestGetCircleIncludesMembers(t *testing.T) {
	db, router := setupRouter()
	router.GET("/circles/:id", GetCircle)

	var user models.User
	db.First(&user)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)
	circle := models.Circle{UserID: user.ID, Name: "College friends"}
	db.Create(&circle)
	db.Create(&models.CircleMember{CircleID: circle.ID, UserID: user.ID, MemberVCardUID: contact.VCardUID})

	req, _ := http.NewRequest("GET", "/circles/"+circle.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Len(t, responseBody["members"], 1)
}

func TestGetCircleNotFoundForUnknownID(t *testing.T) {
	_, router := setupRouter()
	router.GET("/circles/:id", GetCircle)

	req, _ := http.NewRequest("GET", "/circles/does-not-exist", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListCircles(t *testing.T) {
	db, router := setupRouter()
	router.GET("/circles", ListCircles)

	var user models.User
	db.First(&user)
	db.Create(&models.Circle{UserID: user.ID, Name: "College friends"})
	db.Create(&models.Circle{UserID: user.ID, Name: "Cam Girl friends"})

	req, _ := http.NewRequest("GET", "/circles", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Len(t, responseBody["circles"], 2)
	assert.EqualValues(t, 2, responseBody["total"])
}

func TestUpdateCircle(t *testing.T) {
	db, router := setupRouter()
	router.PUT("/circles/:id", withValidated(func() any { return &models.CircleInput{} }), UpdateCircle)

	var user models.User
	db.First(&user)
	circle := models.Circle{UserID: user.ID, Name: "Old name"}
	db.Create(&circle)

	payload := models.CircleInput{Name: "New name"}
	jsonValue, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT", "/circles/"+circle.ID, bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var reloaded models.Circle
	db.First(&reloaded, "id = ?", circle.ID)
	assert.Equal(t, "New name", reloaded.Name)
}

func TestDeleteCircleCascadesMembers(t *testing.T) {
	db, router := setupRouter()
	router.DELETE("/circles/:id", DeleteCircle)

	var user models.User
	db.First(&user)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)
	circle := models.Circle{UserID: user.ID, Name: "College friends"}
	db.Create(&circle)
	db.Create(&models.CircleMember{CircleID: circle.ID, UserID: user.ID, MemberVCardUID: contact.VCardUID})

	req, _ := http.NewRequest("DELETE", "/circles/"+circle.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var count int64
	db.Model(&models.Circle{}).Count(&count)
	assert.Zero(t, count)
}

func TestAddCircleMember(t *testing.T) {
	db, router := setupRouter()
	router.POST("/circles/:id/members", withValidated(func() any { return &models.CircleMemberInput{} }), AddCircleMember)

	var user models.User
	db.First(&user)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)
	circle := models.Circle{UserID: user.ID, Name: "College friends"}
	db.Create(&circle)

	payload := models.CircleMemberInput{MemberVCardUID: contact.VCardUID}
	jsonValue, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/circles/"+circle.ID+"/members", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var count int64
	db.Model(&models.CircleMember{}).Count(&count)
	assert.EqualValues(t, 1, count)
}

func TestAddCircleMemberRejectsDuplicate(t *testing.T) {
	db, router := setupRouter()
	router.POST("/circles/:id/members", withValidated(func() any { return &models.CircleMemberInput{} }), AddCircleMember)

	var user models.User
	db.First(&user)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)
	circle := models.Circle{UserID: user.ID, Name: "College friends"}
	db.Create(&circle)
	db.Create(&models.CircleMember{CircleID: circle.ID, UserID: user.ID, MemberVCardUID: contact.VCardUID})

	payload := models.CircleMemberInput{MemberVCardUID: contact.VCardUID}
	jsonValue, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/circles/"+circle.ID+"/members", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestAddCircleMemberRejectsContactFromAnotherUser(t *testing.T) {
	db, router := setupRouter()
	router.POST("/circles/:id/members", withValidated(func() any { return &models.CircleMemberInput{} }), AddCircleMember)

	var user models.User
	db.First(&user)
	otherUser := models.User{Username: "other", Password: "x", Email: "other@example.com"}
	require.NoError(t, db.Create(&otherUser).Error)
	othersContact := models.Contact{UserID: otherUser.ID, Firstname: "Not Yours"}
	db.Create(&othersContact)
	circle := models.Circle{UserID: user.ID, Name: "College friends"}
	db.Create(&circle)

	payload := models.CircleMemberInput{MemberVCardUID: othersContact.VCardUID}
	jsonValue, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/circles/"+circle.ID+"/members", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRemoveCircleMember(t *testing.T) {
	db, router := setupRouter()
	router.DELETE("/circles/:id/members/:vcard_uid", RemoveCircleMember)

	var user models.User
	db.First(&user)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)
	circle := models.Circle{UserID: user.ID, Name: "College friends"}
	db.Create(&circle)
	db.Create(&models.CircleMember{CircleID: circle.ID, UserID: user.ID, MemberVCardUID: contact.VCardUID})

	req, _ := http.NewRequest("DELETE", "/circles/"+circle.ID+"/members/"+contact.VCardUID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var count int64
	db.Model(&models.CircleMember{}).Count(&count)
	assert.Zero(t, count)
}

func TestRemoveCircleMemberNotFound(t *testing.T) {
	db, router := setupRouter()
	router.DELETE("/circles/:id/members/:vcard_uid", RemoveCircleMember)

	var user models.User
	db.First(&user)
	circle := models.Circle{UserID: user.ID, Name: "College friends"}
	db.Create(&circle)

	req, _ := http.NewRequest("DELETE", "/circles/"+circle.ID+"/members/nonexistent-uid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
