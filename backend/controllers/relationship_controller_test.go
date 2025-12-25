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

func TestGetRelationships(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.GET("/contacts/:id/relationships", GetRelationships)

	// Create a contact and relationships
	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "Jane",
		Lastname:  "Doe",
	}
	db.Create(&contact)

	relationship1 := models.Relationship{
		UserID:           user.ID,
		Name:             "Brother",
		Type:             "Sibling",
		Gender:           "Male",
		ContactID:        contact.ID,
		RelatedContactID: nil, // Using no linked contact for this test
	}
	relationship2 := models.Relationship{
		UserID:           user.ID,
		Name:             "Sister",
		Type:             "Sibling",
		Gender:           "Female",
		ContactID:        contact.ID,
		RelatedContactID: nil,
	}
	db.Create(&relationship1)
	db.Create(&relationship2)

	req, _ := http.NewRequest("GET", "/contacts/"+strconv.Itoa(int(contact.ID))+"/relationships", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Len(t, responseBody["relationships"], 2) // Should return both relationships for the contact
}

func TestCreateRelationship(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.POST("/contacts/:id/relationships", withValidated(func() any { return &models.RelationshipInput{} }), CreateRelationship)

	// Create a contact to associate with the relationship
	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "Alice",
		Lastname:  "Wonderland",
	}
	db.Create(&contact)

	// Create a new relationship
	newRelationship := models.Relationship{
		UserID: user.ID,
		Name:   "Best Friend",
		Type:   "Friendship",
		Gender: "Female",
	}

	jsonValue, _ := json.Marshal(newRelationship)
	req, _ := http.NewRequest("POST", "/contacts/"+strconv.Itoa(int(contact.ID))+"/relationships", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, newRelationship.Name, responseBody["relationship"].(map[string]any)["name"]) // Checking if the created relationship name matches
}

func TestUpdateRelationship(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.PUT("/relationships/:rid", withValidated(func() any { return &models.RelationshipInput{} }), UpdateRelationship)

	// Create a relationship to update
	existingRelationship := models.Relationship{
		UserID: user.ID,
		Name:   "Colleague",
		Type:   "Work",
		Gender: "Male",
	}
	db.Create(&existingRelationship)

	// Update the relationship
	updatedRelationship := models.Relationship{
		UserID: user.ID,
		Name:   "Close Colleague",
		Type:   "Work",
		Gender: "Male",
	}
	jsonValue, _ := json.Marshal(updatedRelationship)

	req, _ := http.NewRequest("PUT", "/relationships/"+strconv.Itoa(int(existingRelationship.ID)), bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody models.Relationship
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, updatedRelationship.Name, responseBody.Name) // Checking if the updated relationship name matches
}

func TestDeleteRelationship(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.DELETE("/relationships/:rid", DeleteRelationship)

	// Create a relationship to delete
	relationshipToDelete := models.Relationship{
		UserID: user.ID,
		Name:   "Cousin",
		Type:   "Family",
		Gender: "Female",
	}
	db.Create(&relationshipToDelete)

	// Make the request to delete the relationship
	req, _ := http.NewRequest("DELETE", "/relationships/"+strconv.Itoa(int(relationshipToDelete.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]string
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, "Relationship deleted", responseBody["message"])

	// Verify relationship has been deleted
	var deletedRelationship models.Relationship
	result := db.First(&deletedRelationship, relationshipToDelete.ID)
	assert.Error(t, result.Error) // This should return an error as it has been deleted
}
