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

func TestGetContactNotes(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.GET("/contacts/:id/notes", GetNotesForContact)

	// Create a contact
	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "John",
		Lastname:  "Doe",
	}
	db.Create(&contact)

	// Create some notes for the contact
	note1 := models.Note{UserID: user.ID, Content: "First note", Date: time.Now(), ContactID: &contact.ID}
	note2 := models.Note{UserID: user.ID, Content: "Second note", Date: time.Now(), ContactID: &contact.ID}
	db.Create(&note1)
	db.Create(&note2)

	// Make the request to get notes for the contact
	req, _ := http.NewRequest("GET", "/contacts/"+strconv.Itoa(int(contact.ID))+"/notes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody struct {
		Notes []models.Note `json:"notes"`
	}
	json.Unmarshal(w.Body.Bytes(), &responseBody)

	// Assert that the notes returned belong to the contact
	assert.Len(t, responseBody.Notes, 2)
	assert.Equal(t, note1.Content, responseBody.Notes[0].Content)
	assert.Equal(t, note2.Content, responseBody.Notes[1].Content)
}

func TestCreateContactNote(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.POST("/contacts/:id/notes", withValidated(func() any { return &models.NoteInput{} }), CreateNote)

	// Create a contact
	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "Jane",
		Lastname:  "Smith",
	}
	db.Create(&contact)

	// Create a note
	now := time.Now()
	newNote := models.NoteInput{
		Content:   "This is a new note.",
		Date:      now,
		ContactID: &contact.ID,
	}

	jsonValue, _ := json.Marshal(newNote)

	req, _ := http.NewRequest("POST", "/contacts/"+strconv.Itoa(int(contact.ID))+"/notes", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, "Note created successfully", responseBody["message"])
}

func TestGetNote(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.GET("/notes/:id", GetNote)

	// Create a note
	note := models.Note{
		UserID:  user.ID,
		Content: "Note for retrieval.",
	}
	db.Create(&note)

	// Make the request to get the note by ID
	req, _ := http.NewRequest("GET", "/notes/"+strconv.Itoa(int(note.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody models.Note
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, note.Content, responseBody.Content)
}

func TestGetNotes(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.GET("/notes", GetUnassignedNotes)

	// Create some unassigned notes
	notes := []models.Note{
		{UserID: user.ID, Content: "Unassigned Note 1"},
		{UserID: user.ID, Content: "Unassigned Note 2"},
	}
	db.Create(&notes)

	// Make the request to get unassigned notes
	req, _ := http.NewRequest("GET", "/notes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody []models.Note
	json.Unmarshal(w.Body.Bytes(), &responseBody)

	// Assert that we received the correct number of unassigned notes
	assert.Len(t, responseBody, 2)
	assert.Equal(t, notes[0].Content, responseBody[0].Content)
	assert.Equal(t, notes[1].Content, responseBody[1].Content)
}

func TestCreateNote(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.POST("/notes", withValidated(func() any { return &models.NoteInput{} }), CreateUnassignedNote)

	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "Standalone",
		Lastname:  "Owner",
	}
	db.Create(&contact)

	// Create a note
	noteDate := time.Now()
	newNote := models.NoteInput{
		Content:   "This is a standalone note.",
		Date:      noteDate,
		ContactID: &contact.ID,
	}

	jsonValue, _ := json.Marshal(newNote)

	req, _ := http.NewRequest("POST", "/notes", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, "Note created successfully", responseBody["message"])
}

func TestUpdateNote(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.PUT("/notes/:id", withValidated(func() any { return &models.NoteInput{} }), UpdateNote)
	router.GET("/notes/:id", GetNote)

	// Seed a contact to satisfy validation and ownership checks.
	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "Linked",
		Lastname:  "Contact",
	}
	db.Create(&contact)

	// Create a note
	note := models.Note{
		UserID:    user.ID,
		Content:   "Original note content.",
		Date:      time.Now(),
		ContactID: &contact.ID,
	}
	db.Create(&note)

	// Update the note
	updatedNote := models.NoteInput{
		Content:   "Updated note content.",
		Date:      time.Now().Add(time.Hour),
		ContactID: &contact.ID,
	}
	jsonValue, _ := json.Marshal(updatedNote)

	req, _ := http.NewRequest("PUT", "/notes/"+strconv.Itoa(int(note.ID)), bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, "Note updated successfully", responseBody["message"])

	// Fetch the note back to verify changes
	fetchReq, _ := http.NewRequest("GET", "/notes/"+strconv.Itoa(int(note.ID)), nil)
	fetchW := httptest.NewRecorder()
	router.ServeHTTP(fetchW, fetchReq)

	var fetchedNote models.Note
	json.Unmarshal(fetchW.Body.Bytes(), &fetchedNote)

	assert.Equal(t, updatedNote.Content, fetchedNote.Content)
}

func TestDeleteNote(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.DELETE("/notes/:id", DeleteNote)

	// Create a note
	note := models.Note{
		UserID:  user.ID,
		Content: "Note to be deleted.",
	}
	db.Create(&note)

	// Make the request to delete the note
	req, _ := http.NewRequest("DELETE", "/notes/"+strconv.Itoa(int(note.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]string
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, "Note deleted", responseBody["message"])
}
