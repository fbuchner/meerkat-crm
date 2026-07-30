package controllers

import (
	"bytes"
	"encoding/json"
	"mycorrhizal/models"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestGetIncomingRelationships mirrors TestGetRelationships but for the
// inverse direction: relationships where the given contact is the *target*
// (related_contact_id), not the owner (contact_id).
func TestGetIncomingRelationships(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.GET("/contacts/:id/relationships/incoming", GetIncomingRelationships)

	// The target contact that other contacts point to.
	target := models.Contact{UserID: user.ID, Firstname: "Target", Lastname: "Person"}
	db.Create(&target)

	source := models.Contact{UserID: user.ID, Firstname: "Source", Lastname: "Person"}
	db.Create(&source)

	incoming := models.Relationship{
		UserID:           user.ID,
		Name:             "Parent",
		Type:             "Family",
		Gender:           "Female",
		ContactID:        source.ID,
		RelatedContactID: &target.ID,
	}
	db.Create(&incoming)

	// A relationship in the other direction (target -> someone else) must not
	// appear in target's *incoming* list.
	unrelated := models.Contact{UserID: user.ID, Firstname: "Unrelated", Lastname: "Person"}
	db.Create(&unrelated)
	outgoing := models.Relationship{
		UserID:           user.ID,
		Name:             "Friend",
		Type:             "Friendship",
		ContactID:        target.ID,
		RelatedContactID: &unrelated.ID,
	}
	db.Create(&outgoing)

	req, _ := http.NewRequest("GET", "/contacts/"+strconv.Itoa(int(target.ID))+"/relationships/incoming", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	relationships, ok := responseBody["incoming_relationships"].([]any)
	if !ok {
		t.Fatalf("expected incoming_relationships array in response")
	}
	if assert.Len(t, relationships, 1) {
		assert.Equal(t, "Parent", relationships[0].(map[string]any)["name"])
	}
}

// TestGetIncomingRelationshipsRejectsContactFromAnotherUser matches the
// established cross-user contact-ownership pattern used elsewhere in this
// package (see life_event_controller_test.go, tag_controller_test.go): a
// user cannot query incoming relationships for a contact they don't own.
func TestGetIncomingRelationshipsRejectsContactFromAnotherUser(t *testing.T) {
	db, router := setupRouter()
	router.GET("/contacts/:id/relationships/incoming", GetIncomingRelationships)

	otherUser := models.User{Username: "other", Password: "x", Email: "other@example.com"}
	require.NoError(t, db.Create(&otherUser).Error)
	othersContact := models.Contact{UserID: otherUser.ID, Firstname: "Not Yours"}
	db.Create(&othersContact)

	req, _ := http.NewRequest("GET", "/contacts/"+strconv.Itoa(int(othersContact.ID))+"/relationships/incoming", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestGetIncomingRelationshipsExcludesAnotherUsersRelationshipRow is a
// defense-in-depth ownership-boundary check: even if a Relationship row
// existed whose related_contact_id happened to match a contact owned by the
// current user, but whose own user_id belonged to someone else, it must not
// leak into the response. GetIncomingRelationships scopes its query by both
// user_id and related_contact_id (relationship_controller.go), so this should
// never surface such a row.
func TestGetIncomingRelationshipsExcludesAnotherUsersRelationshipRow(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)
	router.GET("/contacts/:id/relationships/incoming", GetIncomingRelationships)

	target := models.Contact{UserID: user.ID, Firstname: "Target", Lastname: "Person"}
	db.Create(&target)

	otherUser := models.User{Username: "other", Password: "x", Email: "other@example.com"}
	require.NoError(t, db.Create(&otherUser).Error)
	otherSource := models.Contact{UserID: otherUser.ID, Firstname: "Intruder", Lastname: "Source"}
	db.Create(&otherSource)

	// Seeded directly via the DB (bypassing CreateRelationship, which would
	// itself reject a cross-user RelatedContactID) to verify the read path
	// alone enforces the boundary.
	otherRel := models.Relationship{
		UserID:           otherUser.ID,
		Name:             "Should not leak",
		Type:             "Unknown",
		ContactID:        otherSource.ID,
		RelatedContactID: &target.ID,
	}
	require.NoError(t, db.Create(&otherRel).Error)

	req, _ := http.NewRequest("GET", "/contacts/"+strconv.Itoa(int(target.ID))+"/relationships/incoming", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	relationships, ok := responseBody["incoming_relationships"].([]any)
	if !ok {
		t.Fatalf("expected incoming_relationships array in response")
	}
	assert.Len(t, relationships, 0)
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
