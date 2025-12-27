package controllers

import (
	"bytes"
	"encoding/json"
	"meerkat/models"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

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

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody models.Contact
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, contact.Firstname, responseBody.Firstname)
}

func TestGetContactWithFieldSelection(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.GET("/contacts", GetContacts)

	// Create a contact with multiple fields
	contact := models.Contact{
		UserID:          user.ID,
		Firstname:       "Jane",
		Lastname:        "Doe",
		Email:           "jane@example.com",
		Phone:           "1234567890",
		Address:         "123 Main St",
		WorkInformation: "Software Engineer at TechCorp",
	}
	db.Create(&contact)

	// Request specific fields only (firstname, email)
	req, _ := http.NewRequest("GET", "/contacts?fields=firstname,email", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)

	contacts := responseBody["contacts"].([]any)
	assert.Len(t, contacts, 1)

	contactData := contacts[0].(map[string]any)
	assert.Equal(t, "Jane", contactData["firstname"])
	assert.Equal(t, "jane@example.com", contactData["email"])
	// Fields not requested should be empty/zero values
	assert.Empty(t, contactData["lastname"])
	assert.Empty(t, contactData["phone"])
}

func TestGetContactWithRelationships(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.GET("/contacts", GetContacts)

	// Create a contact
	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "Jane",
		Lastname:  "Doe",
	}
	db.Create(&contact)

	// Create a note for the contact
	note := models.Note{
		UserID:    user.ID,
		ContactID: &contact.ID,
		Content:   "Test note content",
	}
	db.Create(&note)

	// Create a reminder for the contact
	byMail := false
	reminder := models.Reminder{
		UserID:     user.ID,
		ContactID:  &contact.ID,
		Message:    "Follow up with Jane",
		RemindAt:   time.Date(2025, 12, 31, 10, 0, 0, 0, time.UTC),
		Recurrence: "once",
		ByMail:     &byMail,
	}
	db.Create(&reminder)

	// Request contacts with notes included
	req, _ := http.NewRequest("GET", "/contacts?includes=notes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)

	contacts := responseBody["contacts"].([]any)
	assert.Len(t, contacts, 1)

	contactData := contacts[0].(map[string]any)
	notes := contactData["notes"].([]any)
	assert.Len(t, notes, 1)
	assert.Equal(t, "Test note content", notes[0].(map[string]any)["content"])

	// Request contacts with reminders included
	req, _ = http.NewRequest("GET", "/contacts?includes=reminders", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	json.Unmarshal(w.Body.Bytes(), &responseBody)

	contacts = responseBody["contacts"].([]any)
	contactData = contacts[0].(map[string]any)
	reminders := contactData["reminders"].([]any)
	assert.Len(t, reminders, 1)
	assert.Equal(t, "Follow up with Jane", reminders[0].(map[string]any)["message"])

	// Request contacts with multiple relationships included
	req, _ = http.NewRequest("GET", "/contacts?includes=notes,reminders", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	json.Unmarshal(w.Body.Bytes(), &responseBody)

	contacts = responseBody["contacts"].([]any)
	contactData = contacts[0].(map[string]any)
	notes = contactData["notes"].([]any)
	reminders = contactData["reminders"].([]any)
	assert.Len(t, notes, 1)
	assert.Len(t, reminders, 1)
}

func TestGetContactsWithSearchCriteria(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.GET("/contacts", GetContacts)

	// Create multiple contacts
	contacts := []models.Contact{
		{UserID: user.ID, Firstname: "Alice", Lastname: "Johnson", Nickname: "Ali"},
		{UserID: user.ID, Firstname: "Bob", Lastname: "Smith", Nickname: "Bobby"},
		{UserID: user.ID, Firstname: "Carol", Lastname: "Williams", Nickname: ""},
	}

	for _, c := range contacts {
		db.Create(&c)
	}

	// Search by firstname
	req, _ := http.NewRequest("GET", "/contacts?search=Alice", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)

	returnedContacts := responseBody["contacts"].([]any)
	assert.Len(t, returnedContacts, 1)
	assert.Equal(t, "Alice", returnedContacts[0].(map[string]any)["firstname"])

	// Search by lastname
	req, _ = http.NewRequest("GET", "/contacts?search=Smith", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	json.Unmarshal(w.Body.Bytes(), &responseBody)

	returnedContacts = responseBody["contacts"].([]any)
	assert.Len(t, returnedContacts, 1)
	assert.Equal(t, "Bob", returnedContacts[0].(map[string]any)["firstname"])

	// Search by nickname
	req, _ = http.NewRequest("GET", "/contacts?search=Bobby", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	json.Unmarshal(w.Body.Bytes(), &responseBody)

	returnedContacts = responseBody["contacts"].([]any)
	assert.Len(t, returnedContacts, 1)
	assert.Equal(t, "Bob", returnedContacts[0].(map[string]any)["firstname"])

	// Search with no results
	req, _ = http.NewRequest("GET", "/contacts?search=NonExistent", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	json.Unmarshal(w.Body.Bytes(), &responseBody)

	returnedContacts = responseBody["contacts"].([]any)
	assert.Len(t, returnedContacts, 0)
}

func TestCreateContact(t *testing.T) {
	_, router := setupRouter()

	router.POST("/contacts", withValidated(func() any { return &models.ContactInput{} }), CreateContact)

	// Create a contact with basic fields
	newContact := models.ContactInput{
		Firstname: "Alice",
		Lastname:  "Johnson",
		Email:     "alice@example.com",
		Phone:     "1234567890",
	}

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

func TestCreateContactWithAllFields(t *testing.T) {
	_, router := setupRouter()

	router.POST("/contacts", withValidated(func() any { return &models.ContactInput{} }), CreateContact)

	// Create a contact with all fields filled out
	fullContact := models.ContactInput{
		Firstname:          "Robert",
		Lastname:           "Anderson",
		Nickname:           "Bob",
		Gender:             "male",
		Email:              "robert.anderson@example.com",
		Phone:              "+1-555-123-4567",
		Birthday:           "15-03-1985",
		Address:            "456 Oak Avenue, Springfield, IL 62701",
		HowWeMet:           "Met at a tech conference in 2020",
		FoodPreference:     "Vegetarian, loves Italian cuisine",
		WorkInformation:    "Senior Software Engineer at TechCorp Inc.",
		ContactInformation: "Prefers email, available weekdays 9-5",
		Circles:            []string{"Friends", "Work", "Tech Community"},
	}

	jsonValue, _ := json.Marshal(fullContact)

	req, _ := http.NewRequest("POST", "/contacts", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, "Contact created successfully", responseBody["message"])

	// Verify all fields are returned correctly
	contact := responseBody["contact"].(map[string]any)
	assert.Equal(t, "Robert", contact["firstname"])
	assert.Equal(t, "Anderson", contact["lastname"])
	assert.Equal(t, "Bob", contact["nickname"])
	assert.Equal(t, "male", contact["gender"])
	assert.Equal(t, "robert.anderson@example.com", contact["email"])
	assert.Equal(t, "+1-555-123-4567", contact["phone"])
	assert.Equal(t, "15-03-1985", contact["birthday"])
	assert.Equal(t, "456 Oak Avenue, Springfield, IL 62701", contact["address"])
	assert.Equal(t, "Met at a tech conference in 2020", contact["how_we_met"])
	assert.Equal(t, "Vegetarian, loves Italian cuisine", contact["food_preference"])
	assert.Equal(t, "Senior Software Engineer at TechCorp Inc.", contact["work_information"])
	assert.Equal(t, "Prefers email, available weekdays 9-5", contact["contact_information"])
	circles := contact["circles"].([]any)
	assert.Len(t, circles, 3)
}

func TestCreateContactWithBirthdayVariations(t *testing.T) {
	_, router := setupRouter()

	router.POST("/contacts", withValidated(func() any { return &models.ContactInput{} }), CreateContact)

	tests := []struct {
		name     string
		birthday string
		desc     string
	}{
		{
			name:     "Fully known birthday",
			birthday: "25-12-1990",
			desc:     "Day, month, and year known",
		},
		{
			name:     "Birthday without year",
			birthday: "25-12",
			desc:     "Day and month known, year unknown",
		},
		{
			name:     "Birthday with only month and year",
			birthday: "00-12-1990",
			desc:     "Month and year known, day unknown",
		},
		{
			name:     "Unknown birthday",
			birthday: "",
			desc:     "Birthday completely unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contact := models.ContactInput{
				Firstname: "Test",
				Lastname:  tt.name,
				Birthday:  tt.birthday,
			}

			jsonValue, _ := json.Marshal(contact)

			req, _ := http.NewRequest("POST", "/contacts", bytes.NewBuffer(jsonValue))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code, "Failed for: %s", tt.desc)

			var responseBody map[string]any
			json.Unmarshal(w.Body.Bytes(), &responseBody)
			assert.Equal(t, "Contact created successfully", responseBody["message"])

			contactResp := responseBody["contact"].(map[string]any)
			assert.Equal(t, tt.birthday, contactResp["birthday"])
		})
	}
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
