package controllers

import (
	"bytes"
	"encoding/json"
	"meerkat/models"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateEmploymentHistory(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.POST("/contacts/:id/employment-history",
		withValidated(func() any { return &models.EmploymentHistoryInput{} }), CreateEmploymentHistory)

	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)

	input := models.EmploymentHistoryInput{Organization: "Acme", JobTitle: "Engineer", StartDate: "2022-01"}
	jsonValue, _ := json.Marshal(input)
	req, _ := http.NewRequest("POST", "/contacts/"+strconv.Itoa(int(contact.ID))+"/employment-history", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var updatedContact models.Contact
	db.First(&updatedContact, contact.ID)
	assert.Equal(t, "Acme", updatedContact.Organization)
	assert.Equal(t, "Engineer", updatedContact.JobTitle)
}

func TestCreateEmploymentHistory_RejectsEndDateBeforeStartDate(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.POST("/contacts/:id/employment-history",
		withValidated(func() any { return &models.EmploymentHistoryInput{} }), CreateEmploymentHistory)

	contact := models.Contact{UserID: user.ID, Firstname: "Eve"}
	db.Create(&contact)

	input := models.EmploymentHistoryInput{Organization: "Acme", StartDate: "2022-06", EndDate: "2021-01"}
	jsonValue, _ := json.Marshal(input)
	req, _ := http.NewRequest("POST", "/contacts/"+strconv.Itoa(int(contact.ID))+"/employment-history", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var count int64
	db.Model(&models.EmploymentHistory{}).Where("contact_id = ?", contact.ID).Count(&count)
	assert.Equal(t, int64(0), count, "no entry should be created when validation fails")
}

func TestGetEmploymentHistory_OrdersMostRecentFirst(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.GET("/contacts/:id/employment-history", GetEmploymentHistory)

	contact := models.Contact{UserID: user.ID, Firstname: "Bob"}
	db.Create(&contact)
	db.Create(&models.EmploymentHistory{UserID: user.ID, ContactID: contact.ID, Organization: "Old Co", StartDate: "2015", EndDate: "2019"})
	db.Create(&models.EmploymentHistory{UserID: user.ID, ContactID: contact.ID, Organization: "New Co", StartDate: "2022"})

	req, _ := http.NewRequest("GET", "/contacts/"+strconv.Itoa(int(contact.ID))+"/employment-history", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string][]models.EmploymentHistory
	json.Unmarshal(w.Body.Bytes(), &body)
	entries := body["employment_history"]
	assert.Len(t, entries, 2)
	assert.Equal(t, "New Co", entries[0].Organization) // open entry first
}

func TestEndEmploymentHistory_ClosesOpenEntryAndRecomputesContact(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.POST("/contacts/:id/employment-history/:eid/end", EndEmploymentHistory)

	// NOTE: Contact.Firstname is set without Organization here (deliberately)
	// so Contact.AfterSave doesn't auto-create its own open EmploymentHistory
	// entry alongside the one we create explicitly below — see Task 3's
	// syncCurrentEmploymentHistory. We then set Organization via a second
	// Save, which lands on the entry we just created (no duplicate) because
	// it already matches.
	contact := models.Contact{UserID: user.ID, Firstname: "Carl"}
	db.Create(&contact)
	entry := models.EmploymentHistory{UserID: user.ID, ContactID: contact.ID, Organization: "Acme"}
	db.Create(&entry)
	contact.Organization = "Acme"
	db.Save(&contact)

	req, _ := http.NewRequest("POST",
		"/contacts/"+strconv.Itoa(int(contact.ID))+"/employment-history/"+strconv.Itoa(int(entry.ID))+"/end", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var closed models.EmploymentHistory
	db.First(&closed, entry.ID)
	assert.NotEmpty(t, closed.EndDate)

	var updatedContact models.Contact
	db.First(&updatedContact, contact.ID)
	assert.Equal(t, "", updatedContact.Organization) // no open entries left
}

func TestEndEmploymentHistory_RejectsAlreadyClosedEntry(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.POST("/contacts/:id/employment-history/:eid/end", EndEmploymentHistory)

	contact := models.Contact{UserID: user.ID, Firstname: "Dana"}
	db.Create(&contact)
	entry := models.EmploymentHistory{UserID: user.ID, ContactID: contact.ID, Organization: "Acme", EndDate: "2020-01-01"}
	db.Create(&entry)

	req, _ := http.NewRequest("POST",
		"/contacts/"+strconv.Itoa(int(contact.ID))+"/employment-history/"+strconv.Itoa(int(entry.ID))+"/end", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code)
}
