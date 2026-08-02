package controllers

import (
	"bytes"
	"encoding/json"
	"mycorrhizal/middleware"
	"mycorrhizal/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// createHouseholdTestContact optionally sets CRM.Kind (pass "" for an
// ordinary human contact). Setting Contact.CRM directly before Create does
// NOT survive BeforeSave — it rebuilds CRM from the flat fields, discarding
// the nested kind (the exact WP-81/WP-83 trap the ticket warns about). Use
// ApplyRecordToContact, mirroring household_service_test.go.
func createHouseholdTestContact(t *testing.T, db *gorm.DB, userID uint, firstname, kind string) models.Contact {
	t.Helper()
	contact := models.Contact{UserID: userID, Firstname: firstname}
	if kind != "" {
		record := models.RecordForContact(&contact, "", nil)
		record.Envelope.Kind = kind
		models.ApplyRecordToContact(&contact, record, "")
	}
	require.NoError(t, db.Create(&contact).Error)
	return contact
}

func registerHouseholdRoutes(t *testing.T, router *gin.Engine) {
	router.POST("/households", withValidated(func() any { return &models.HouseholdInput{} }), CreateHousehold)
	router.GET("/households", ListHouseholds)
	router.GET("/households/:id", GetHousehold)
	router.PUT("/households/:id", withValidated(func() any { return &models.HouseholdInput{} }), UpdateHousehold)
	router.DELETE("/households/:id", DeleteHousehold)
	router.POST("/households/:id/members", withValidated(func() any { return &models.HouseholdMemberInput{} }), AddHouseholdMember)
	router.DELETE("/households/:id/members/:vcard_uid", RemoveHouseholdMember)
	router.POST("/households/:id/suggest-relationships", SuggestHouseholdRelationships)
}

func TestCreateHousehold(t *testing.T) {
	db, router := setupRouter()
	registerHouseholdRoutes(t, router)

	payload := models.HouseholdInput{Name: "Smith Family", Type: models.HouseholdTypeFamilyUnit}
	jsonValue, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/households", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var count int64
	db.Model(&models.Household{}).Count(&count)
	assert.EqualValues(t, 1, count)
}

func TestCreateHouseholdRejectsInvalidType(t *testing.T) {
	_, router := setupRouter()
	// Use the real validation middleware (not withValidated) so the `oneof`
	// tag on HouseholdInput.Type is actually enforced.
	router.POST("/households", middleware.ValidateJSONMiddleware(&models.HouseholdInput{}), CreateHousehold)

	payload := models.HouseholdInput{Name: "Smith Family", Type: "tribe"}
	jsonValue, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/households", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetHouseholdIncludesMembers(t *testing.T) {
	db, router := setupRouter()
	registerHouseholdRoutes(t, router)

	var user models.User
	db.First(&user)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)
	household := models.Household{UserID: user.ID, Name: "Smith Family", Type: models.HouseholdTypeFamilyUnit}
	db.Create(&household)
	db.Create(&models.HouseholdMember{HouseholdID: household.ID, UserID: user.ID, MemberVCardUID: contact.VCardUID, Role: models.HouseholdRoleHead})

	req, _ := http.NewRequest("GET", "/households/"+household.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Len(t, responseBody["members"], 1)
}

func TestGetHouseholdNotFoundForUnknownID(t *testing.T) {
	_, router := setupRouter()
	registerHouseholdRoutes(t, router)

	req, _ := http.NewRequest("GET", "/households/does-not-exist", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetHouseholdRejectsAnotherUsersHousehold(t *testing.T) {
	db, router := setupRouter()
	registerHouseholdRoutes(t, router)

	var user models.User
	db.First(&user)
	otherUser := models.User{Username: "other", Password: "x", Email: "other@example.com"}
	require.NoError(t, db.Create(&otherUser).Error)
	otherHousehold := models.Household{UserID: otherUser.ID, Name: "Not Yours", Type: models.HouseholdTypeFamilyUnit}
	require.NoError(t, db.Create(&otherHousehold).Error)

	req, _ := http.NewRequest("GET", "/households/"+otherHousehold.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListHouseholds(t *testing.T) {
	db, router := setupRouter()
	registerHouseholdRoutes(t, router)

	var user models.User
	db.First(&user)
	db.Create(&models.Household{UserID: user.ID, Name: "Smith Family", Type: models.HouseholdTypeFamilyUnit})
	db.Create(&models.Household{UserID: user.ID, Name: "Apt 4B", Type: models.HouseholdTypeRoommates})

	req, _ := http.NewRequest("GET", "/households", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Len(t, responseBody["households"], 2)
	assert.EqualValues(t, 2, responseBody["total"])
}

func TestUpdateHousehold(t *testing.T) {
	db, router := setupRouter()
	registerHouseholdRoutes(t, router)

	var user models.User
	db.First(&user)
	household := models.Household{UserID: user.ID, Name: "Old name", Type: models.HouseholdTypeFamilyUnit}
	db.Create(&household)

	payload := models.HouseholdInput{Name: "New name", Type: models.HouseholdTypeRoommates}
	jsonValue, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT", "/households/"+household.ID, bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var reloaded models.Household
	db.First(&reloaded, "id = ?", household.ID)
	assert.Equal(t, "New name", reloaded.Name)
	assert.Equal(t, models.HouseholdTypeRoommates, reloaded.Type)
}

func TestDeleteHouseholdCascadesMembers(t *testing.T) {
	db, router := setupRouter()
	registerHouseholdRoutes(t, router)

	var user models.User
	db.First(&user)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)
	household := models.Household{UserID: user.ID, Name: "Smith Family", Type: models.HouseholdTypeFamilyUnit}
	db.Create(&household)
	db.Create(&models.HouseholdMember{HouseholdID: household.ID, UserID: user.ID, MemberVCardUID: contact.VCardUID})

	req, _ := http.NewRequest("DELETE", "/households/"+household.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var count int64
	db.Model(&models.Household{}).Count(&count)
	assert.Zero(t, count)
}

func TestAddHouseholdMember(t *testing.T) {
	db, router := setupRouter()
	registerHouseholdRoutes(t, router)

	var user models.User
	db.First(&user)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)
	household := models.Household{UserID: user.ID, Name: "Smith Family", Type: models.HouseholdTypeFamilyUnit}
	db.Create(&household)

	payload := models.HouseholdMemberInput{MemberVCardUID: contact.VCardUID, Role: models.HouseholdRoleHead}
	jsonValue, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/households/"+household.ID+"/members", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var member models.HouseholdMember
	require.NoError(t, db.First(&member, "household_id = ?", household.ID).Error)
	assert.Equal(t, models.HouseholdRoleHead, member.Role)
}

func TestAddHouseholdMemberRejectsDuplicate(t *testing.T) {
	db, router := setupRouter()
	registerHouseholdRoutes(t, router)

	var user models.User
	db.First(&user)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)
	household := models.Household{UserID: user.ID, Name: "Smith Family", Type: models.HouseholdTypeFamilyUnit}
	db.Create(&household)
	db.Create(&models.HouseholdMember{HouseholdID: household.ID, UserID: user.ID, MemberVCardUID: contact.VCardUID})

	payload := models.HouseholdMemberInput{MemberVCardUID: contact.VCardUID}
	jsonValue, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/households/"+household.ID+"/members", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestAddHouseholdMemberRejectsContactFromAnotherUser(t *testing.T) {
	db, router := setupRouter()
	registerHouseholdRoutes(t, router)

	var user models.User
	db.First(&user)
	otherUser := models.User{Username: "other", Password: "x", Email: "other@example.com"}
	require.NoError(t, db.Create(&otherUser).Error)
	othersContact := models.Contact{UserID: otherUser.ID, Firstname: "Not Yours"}
	db.Create(&othersContact)
	household := models.Household{UserID: user.ID, Name: "Smith Family", Type: models.HouseholdTypeFamilyUnit}
	db.Create(&household)

	payload := models.HouseholdMemberInput{MemberVCardUID: othersContact.VCardUID}
	jsonValue, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/households/"+household.ID+"/members", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAddHouseholdMemberRejectsAnotherUsersHousehold(t *testing.T) {
	db, router := setupRouter()
	registerHouseholdRoutes(t, router)

	var user models.User
	db.First(&user)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)
	otherUser := models.User{Username: "other", Password: "x", Email: "other@example.com"}
	require.NoError(t, db.Create(&otherUser).Error)
	otherHousehold := models.Household{UserID: otherUser.ID, Name: "Not Yours", Type: models.HouseholdTypeFamilyUnit}
	require.NoError(t, db.Create(&otherHousehold).Error)

	payload := models.HouseholdMemberInput{MemberVCardUID: contact.VCardUID}
	jsonValue, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/households/"+otherHousehold.ID+"/members", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRemoveHouseholdMember(t *testing.T) {
	db, router := setupRouter()
	registerHouseholdRoutes(t, router)

	var user models.User
	db.First(&user)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	db.Create(&contact)
	household := models.Household{UserID: user.ID, Name: "Smith Family", Type: models.HouseholdTypeFamilyUnit}
	db.Create(&household)
	db.Create(&models.HouseholdMember{HouseholdID: household.ID, UserID: user.ID, MemberVCardUID: contact.VCardUID})

	req, _ := http.NewRequest("DELETE", "/households/"+household.ID+"/members/"+contact.VCardUID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var count int64
	db.Model(&models.HouseholdMember{}).Count(&count)
	assert.Zero(t, count)
}

func TestRemoveHouseholdMemberNotFound(t *testing.T) {
	db, router := setupRouter()
	registerHouseholdRoutes(t, router)

	var user models.User
	db.First(&user)
	household := models.Household{UserID: user.ID, Name: "Smith Family", Type: models.HouseholdTypeFamilyUnit}
	db.Create(&household)

	req, _ := http.NewRequest("DELETE", "/households/"+household.ID+"/members/nonexistent-uid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestSuggestHouseholdRelationships is the ticket's core round-trip against
// the AutoMigrate setup: create a household with two adults + a pet, run the
// trigger, assert the expected suggested edges exist (spouse_of + owned_by),
// re-run, assert nothing duplicated.
func TestSuggestHouseholdRelationships(t *testing.T) {
	db, router := setupRouter()
	registerHouseholdRoutes(t, router)

	var user models.User
	db.First(&user)
	adult1 := createHouseholdTestContact(t, db, user.ID, "Alice", "")
	adult2 := createHouseholdTestContact(t, db, user.ID, "Bob", "")
	pet := createHouseholdTestContact(t, db, user.ID, "Fluffy", "pet")

	household := models.Household{UserID: user.ID, Name: "Smith Family", Type: models.HouseholdTypeFamilyUnit}
	require.NoError(t, db.Create(&household).Error)
	db.Create(&models.HouseholdMember{HouseholdID: household.ID, UserID: user.ID, MemberVCardUID: adult1.VCardUID, Role: models.HouseholdRoleHead})
	db.Create(&models.HouseholdMember{HouseholdID: household.ID, UserID: user.ID, MemberVCardUID: adult2.VCardUID, Role: models.HouseholdRoleHead})
	db.Create(&models.HouseholdMember{HouseholdID: household.ID, UserID: user.ID, MemberVCardUID: pet.VCardUID, Role: models.HouseholdRolePet})

	req, _ := http.NewRequest("POST", "/households/"+household.ID+"/suggest-relationships", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var created struct {
		Message        string                    `json:"message"`
		SuggestedEdges []models.RelationshipEdge `json:"suggested_edges"`
		Total          int                       `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, 3, created.Total, "1 spouse_of + one owned_by per human->pet")
	typeCounts := map[string]int{}
	for _, edge := range created.SuggestedEdges {
		typeCounts[edge.Type]++
		assert.Equal(t, models.RelationshipStatusSuggested, edge.Status)
		assert.Equal(t, models.RelationshipSourceHouseholdInferred, edge.Source)
	}
	assert.Equal(t, 1, typeCounts["spouse_of"])
	assert.Equal(t, 2, typeCounts["owned_by"])

	// Idempotency: a second run must create nothing new.
	req2, _ := http.NewRequest("POST", "/households/"+household.ID+"/suggest-relationships", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	var second struct {
		Total int `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &second))
	assert.Equal(t, 0, second.Total, "re-running the trigger must not duplicate edges")

	var totalEdges int64
	db.Model(&models.RelationshipEdge{}).Count(&totalEdges)
	assert.EqualValues(t, 3, totalEdges)
}

// The suggestion trigger must refuse to run against another user's household.
func TestSuggestHouseholdRelationshipsRejectsAnotherUsersHousehold(t *testing.T) {
	db, router := setupRouter()
	registerHouseholdRoutes(t, router)

	var user models.User
	db.First(&user)
	otherUser := models.User{Username: "other", Password: "x", Email: "other@example.com"}
	require.NoError(t, db.Create(&otherUser).Error)
	otherHousehold := models.Household{UserID: otherUser.ID, Name: "Not Yours", Type: models.HouseholdTypeFamilyUnit}
	require.NoError(t, db.Create(&otherHousehold).Error)

	req, _ := http.NewRequest("POST", "/households/"+otherHousehold.ID+"/suggest-relationships", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
