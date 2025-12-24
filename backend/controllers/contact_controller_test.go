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

func TestGetContacts(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.GET("/contacts", GetContacts)

	// Create some contacts
	contacts := []models.Contact{
		{UserID: user.ID, Firstname: "Alice", Lastname: "Johnson"},
		{UserID: user.ID, Firstname: "Bob", Lastname: "Smith"},
		{UserID: user.ID, Firstname: "Carol", Lastname: "Williams"},
		{UserID: user.ID, Firstname: "David", Lastname: "Brown"},
		{UserID: user.ID, Firstname: "Eve", Lastname: "Davis"},
	}

	for _, contact := range contacts {
		db.Create(&contact)
	}

	// Test pagination (page=1, limit=2)
	req, _ := http.NewRequest("GET", "/contacts?page=1&limit=2", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)

	// Assert that only 2 contacts are returned
	contactsReturned := responseBody["contacts"]
	assert.Len(t, contactsReturned, 2)
	assert.Equal(t, float64(5), responseBody["total"]) // Total contacts
	assert.Equal(t, float64(1), responseBody["page"])  // Current page
	assert.Equal(t, float64(2), responseBody["limit"]) // Limit per page

	// Test pagination (page=2, limit=2)
	req, _ = http.NewRequest("GET", "/contacts?page=2&limit=2", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	json.Unmarshal(w.Body.Bytes(), &responseBody)

	// Assert that only the next 2 contacts are returned
	contactsReturned = responseBody["contacts"]
	assert.Len(t, contactsReturned, 2)
	assert.Equal(t, float64(5), responseBody["total"]) // Total contacts
	assert.Equal(t, float64(2), responseBody["page"])  // Current page
	assert.Equal(t, float64(2), responseBody["limit"]) // Limit per page

	// Test pagination for non-existent page (page=5, limit=2)
	req, _ = http.NewRequest("GET", "/contacts?page=5&limit=2", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	json.Unmarshal(w.Body.Bytes(), &responseBody)

	// Assert that no contacts are returned on non-existent page
	contactsReturned = responseBody["contacts"]

	assert.Len(t, contactsReturned, 0)
	assert.Equal(t, float64(5), responseBody["total"]) // Total contacts
	assert.Equal(t, float64(5), responseBody["page"])  // Current page
	assert.Equal(t, float64(2), responseBody["limit"]) // Limit per page
}

func TestGetContact(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.GET("/contacts/:id", GetContact)

	// Create a contact
	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "Jane",
		Lastname:  "Doe",
	}
	db.Create(&contact)

	// Make the request to get the contact by ID
	req, _ := http.NewRequest("GET", "/contacts/"+strconv.Itoa(int(contact.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	//TODO repeat the same but requesting specific fields only
	//TODO repeat the same but requesting specific relationships to be included
	//TODO repeat the same but using search criteria

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody models.Contact
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, contact.Firstname, responseBody.Firstname)
}

func TestCreateContact(t *testing.T) {
	_, router := setupRouter()

	router.POST("/contacts", withValidated(func() any { return &models.ContactInput{} }), CreateContact)

	// Create a contact
	newContact := models.ContactInput{
		Firstname: "Alice",
		Lastname:  "Johnson",
		Email:     "alice@example.com",
		Phone:     "1234567890",
	}

	//TODO repeat the same for a contact where all fields are filled out
	//TODO repeat the same for a contact where birthday is unknown, fully known, partially known

	jsonValue, _ := json.Marshal(newContact)

	req, _ := http.NewRequest("POST", "/contacts", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, "Contact created successfully", responseBody["message"])
}

func TestUpdateContact(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.PUT("/contacts/:id", withValidated(func() any { return &models.ContactInput{} }), UpdateContact)

	// Create a contact
	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "Alice",
		Lastname:  "Johnson",
	}
	db.Create(&contact)

	// Update the contact
	updatedContact := models.ContactInput{
		Firstname: "Alice Updated",
		Lastname:  "Johnson Updated",
	}
	jsonValue, _ := json.Marshal(updatedContact)

	req, _ := http.NewRequest("PUT", "/contacts/"+strconv.Itoa(int(contact.ID)), bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody models.Contact
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, updatedContact.Firstname, responseBody.Firstname)
}

func TestDeleteContact(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.DELETE("/contacts/:id", DeleteContact)

	// Create a contact
	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "Alice",
		Lastname:  "Johnson",
	}
	db.Create(&contact)

	// Make the request to delete the contact
	req, _ := http.NewRequest("DELETE", "/contacts/"+strconv.Itoa(int(contact.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]string
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, "Contact deleted", responseBody["message"])
}

func TestGetCircles(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.GET("/contacts/circles", GetCircles)

	contacts := []models.Contact{
		{UserID: user.ID, Firstname: "Alice", Lastname: "Johnson", Circles: []string{"Friends", "Family"}},
		{UserID: user.ID, Firstname: "Bob", Lastname: "Smith", Circles: []string{"Friends", "Work"}},
	}
	db.Create(&contacts[0])
	db.Create(&contacts[1])

	// Make the request to get circles
	req, _ := http.NewRequest("GET", "/contacts/circles", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody []string
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, int(3), len(responseBody))
	assert.ElementsMatch(t, []string{"Friends", "Family", "Work"}, responseBody)
}
