package controllers

import (
	"bytes"
	"encoding/json"
	"mycorrhizal/contactmodel"
	"mycorrhizal/models"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func intPtr(v int) *int { return &v }

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

// TestGetContactsSearchMultiValue verifies search matches values in the emails/phones
// JSON arrays (including secondary entries), not just the denormalized primary scalar.
func TestGetContactsSearchMultiValue(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.GET("/contacts", GetContacts)

	c := models.Contact{
		UserID:    user.ID,
		Firstname: "Grace",
		Lastname:  "Hopper",
		Emails: []models.ContactEmail{
			{Type: "home", Value: "grace@home.example"},
			{Type: "work", Value: "grace@navy.example"},
		},
		Phones: []models.ContactPhone{{Type: "cell", Value: "+15551112222"}},
	}
	db.Create(&c)
	// A second contact that must NOT match the searches below.
	db.Create(&models.Contact{UserID: user.ID, Firstname: "Alan", Lastname: "Turing"})

	search := func(term string) int {
		req, _ := http.NewRequest("GET", "/contacts?search="+term, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		var body map[string]any
		json.Unmarshal(w.Body.Bytes(), &body)
		return int(body["total"].(float64))
	}

	assert.Equal(t, 1, search("navy"), "secondary work email should be searchable")
	assert.Equal(t, 1, search("grace@home"), "primary email should be searchable")
	assert.Equal(t, 1, search("5551112"), "phone array value should be searchable")
	assert.Equal(t, 1, search("Hopper"), "name should still be searchable")
	assert.Equal(t, 0, search("nomatch"), "unrelated term should match nothing")
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

	// GET /contacts/:id now returns the full neutral ContactRecordResponse
	// (Card/CRM/Passthrough nested), not a flat models.Contact (Gap 3/item 2).
	var responseBody models.ContactRecordResponse
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, contact.ID, responseBody.ID)
	if assert.NotNil(t, responseBody.Card.Name) {
		var gotFirst string
		for _, comp := range responseBody.Card.Name.Components {
			if comp.Kind == "given" {
				gotFirst = comp.Value
			}
		}
		assert.Equal(t, contact.Firstname, gotFirst)
	}
}

// TestGetContactsFieldsParamIgnored asserts fields= is gone (Gap 3): passing
// it no longer restricts or alters the response shape, which is always the
// fixed ContactSummary regardless of what (if anything) fields= requests.
func TestGetContactsFieldsParamIgnored(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.GET("/contacts", GetContacts)

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

	// fields=firstname,email would once have restricted the response to just
	// those columns; it must now have no effect at all.
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
	assert.Equal(t, "jane@example.com", contactData["primary_email"])
	// lastname was NOT in fields=, but the slim ContactSummary shape is
	// returned unconditionally now, so it's still present.
	assert.Equal(t, "Doe", contactData["lastname"])
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

// TestGetContactsArchiveAndCircleFiltering asserts Gap 2's binding
// preservation of the archive-filtering and circle-filtering mechanics
// against the new ContactSummary item shape.
func TestGetContactsArchiveAndCircleFiltering(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.GET("/contacts", GetContacts)

	active := models.Contact{UserID: user.ID, Firstname: "Active", Lastname: "One", Circles: []string{"friends"}}
	archived := models.Contact{UserID: user.ID, Firstname: "Archived", Lastname: "One", Circles: []string{"work"}, Archived: true}
	db.Create(&active)
	db.Create(&archived)

	// Default: archived excluded.
	req, _ := http.NewRequest("GET", "/contacts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	assert.Equal(t, float64(1), body["total"])

	// include_archived=true: both returned.
	req, _ = http.NewRequest("GET", "/contacts?include_archived=true", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	json.Unmarshal(w.Body.Bytes(), &body)
	assert.Equal(t, float64(2), body["total"])

	// archived=true (without include_archived): only the archived one.
	req, _ = http.NewRequest("GET", "/contacts?archived=true", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	json.Unmarshal(w.Body.Bytes(), &body)
	assert.Equal(t, float64(1), body["total"])
	contacts := body["contacts"].([]any)
	assert.Equal(t, "Archived", contacts[0].(map[string]any)["firstname"])

	// circle= filters by circle membership.
	req, _ = http.NewRequest("GET", "/contacts?circle=friends", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	json.Unmarshal(w.Body.Bytes(), &body)
	contacts = body["contacts"].([]any)
	assert.Len(t, contacts, 1)
	assert.Equal(t, "Active", contacts[0].(map[string]any)["firstname"])
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

	router.POST("/contacts", withValidated(func() any { return &models.ContactRecordInput{} }), CreateContact)

	// Create a contact with basic fields, using the new nested Card shape.
	newContact := models.ContactRecordInput{
		Card: contactmodel.Card{
			Name: &contactmodel.Name{Components: []contactmodel.NameComponent{
				{Kind: "given", Value: "Alice"},
				{Kind: "surname", Value: "Johnson"},
			}},
			Emails: []contactmodel.Email{{Address: "alice@example.com"}},
			Phones: []contactmodel.Phone{{Number: "1234567890"}},
		},
	}

	jsonValue, _ := json.Marshal(newContact)

	req, _ := http.NewRequest("POST", "/contacts", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, "Contact created successfully", responseBody["message"])
}

func TestCreateContactWithAllFields(t *testing.T) {
	_, router := setupRouter()

	router.POST("/contacts", withValidated(func() any { return &models.ContactRecordInput{} }), CreateContact)

	// Create a contact with all fields filled out, via the nested Card/CRM
	// shape (Gender rides alongside, per ContactRecordInput's doc comment).
	fullContact := models.ContactRecordInput{
		Gender: "male",
		Card: contactmodel.Card{
			Name: &contactmodel.Name{Components: []contactmodel.NameComponent{
				{Kind: "given", Value: "Robert"},
				{Kind: "surname", Value: "Anderson"},
			}},
			Nicknames: []contactmodel.Nickname{{Name: "Bob"}},
			Emails:    []contactmodel.Email{{Address: "robert.anderson@example.com"}},
			Phones:    []contactmodel.Phone{{Number: "+1-555-123-4567"}},
			Anniversaries: []contactmodel.Anniversary{
				{Kind: "birth", Date: contactmodel.AnniversaryDate{Partial: &contactmodel.PartialDate{
					Year: intPtr(1985), Month: intPtr(3), Day: intPtr(15),
				}}},
			},
			Addresses: []contactmodel.Address{{Full: "456 Oak Avenue, Springfield, IL 62701"}},
		},
		CRM: contactmodel.CRMEnvelope{
			HowWeMet:           "Met at a tech conference in 2020",
			FoodPreference:     "Vegetarian, loves Italian cuisine",
			WorkInformation:    "Senior Software Engineer at TechCorp Inc.",
			ContactInformation: "Prefers email, available weekdays 9-5",
			Circles:            []string{"Friends", "Work", "Tech Community"},
		},
	}

	jsonValue, _ := json.Marshal(fullContact)

	req, _ := http.NewRequest("POST", "/contacts", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, "Contact created successfully", responseBody["message"])

	// Verify all fields are returned correctly in the nested
	// ContactRecordResponse shape.
	contact := responseBody["contact"].(map[string]any)
	assert.Equal(t, "male", contact["gender"])

	crm := contact["crm"].(map[string]any)
	assert.Equal(t, "Met at a tech conference in 2020", crm["how_we_met"])
	assert.Equal(t, "Vegetarian, loves Italian cuisine", crm["food_preference"])
	assert.Equal(t, "Senior Software Engineer at TechCorp Inc.", crm["work_information"])
	assert.Equal(t, "Prefers email, available weekdays 9-5", crm["contact_information"])
	circles := crm["circles"].([]any)
	assert.Len(t, circles, 3)

	card := contact["card"].(map[string]any)
	nicknames := card["nicknames"].([]any)
	assert.Equal(t, "Bob", nicknames[0].(map[string]any)["name"])
	emails := card["emails"].([]any)
	assert.Equal(t, "robert.anderson@example.com", emails[0].(map[string]any)["address"])
}

// TestCreateContactBirthdayPartialDates replaces the old flat-scalar
// birthday-format test: the new nested input carries a birthday as a
// structured Card.Anniversaries[kind=birth].Date.Partial (year/month/day),
// not a raw formatted string, so this exercises the equivalent
// full-date/year-less/absent variations against that shape instead.
func TestCreateContactBirthdayPartialDates(t *testing.T) {
	_, router := setupRouter()

	router.POST("/contacts", withValidated(func() any { return &models.ContactRecordInput{} }), CreateContact)

	tests := []struct {
		name  string
		date  *contactmodel.PartialDate
		count int // expected len(card.anniversaries)
	}{
		{"Fully known birthday", &contactmodel.PartialDate{Year: intPtr(1990), Month: intPtr(12), Day: intPtr(25)}, 1},
		{"Birthday without year", &contactmodel.PartialDate{Month: intPtr(12), Day: intPtr(25)}, 1},
		{"Unknown birthday", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := models.ContactRecordInput{
				Card: contactmodel.Card{
					Name: &contactmodel.Name{Components: []contactmodel.NameComponent{{Kind: "given", Value: "Test"}}},
				},
			}
			if tt.date != nil {
				input.Card.Anniversaries = []contactmodel.Anniversary{
					{Kind: "birth", Date: contactmodel.AnniversaryDate{Partial: tt.date}},
				}
			}

			jsonValue, _ := json.Marshal(input)

			req, _ := http.NewRequest("POST", "/contacts", bytes.NewBuffer(jsonValue))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusCreated, w.Code)

			var responseBody map[string]any
			json.Unmarshal(w.Body.Bytes(), &responseBody)
			assert.Equal(t, "Contact created successfully", responseBody["message"])

			contactResp := responseBody["contact"].(map[string]any)
			card := contactResp["card"].(map[string]any)
			anniversaries, _ := card["anniversaries"].([]any)
			assert.Len(t, anniversaries, tt.count)
		})
	}
}

func TestUpdateContact(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.PUT("/contacts/:id", withValidated(func() any { return &models.ContactRecordInput{} }), UpdateContact)

	// Create a contact
	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "Alice",
		Lastname:  "Johnson",
	}
	db.Create(&contact)

	// Update the contact via the new nested shape
	updatedContact := models.ContactRecordInput{
		Card: contactmodel.Card{
			Name: &contactmodel.Name{Components: []contactmodel.NameComponent{
				{Kind: "given", Value: "Alice Updated"},
				{Kind: "surname", Value: "Johnson Updated"},
			}},
		},
	}
	jsonValue, _ := json.Marshal(updatedContact)

	req, _ := http.NewRequest("PUT", "/contacts/"+strconv.Itoa(int(contact.ID)), bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody models.ContactRecordResponse
	json.Unmarshal(w.Body.Bytes(), &responseBody)

	var gotFirst string
	if responseBody.Card.Name != nil {
		for _, comp := range responseBody.Card.Name.Components {
			if comp.Kind == "given" {
				gotFirst = comp.Value
			}
		}
	}
	assert.Equal(t, "Alice Updated", gotFirst)
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

// TestDeleteContact_CleansUpReferencingRows is the regression test for Tier
// 3c item 1 (docs/fork-plan/95-backlog-and-priorities.md): deleting a contact
// must remove every row that references it via Contact.VCardUID (or, for
// ContactSyncLink, Contact.ID), but must NOT delete the shared
// Household/Circle/Tag/FieldDefinition containers other contacts may still
// belong to.
func TestDeleteContact_CleansUpReferencingRows(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	contact := models.Contact{UserID: user.ID, Firstname: "Orphan", Lastname: "Check"}
	require.NoError(t, db.Create(&contact).Error)

	require.NoError(t, db.Create(&models.RelationshipEdge{UserID: user.ID, SourceID: contact.VCardUID, TargetID: contact.VCardUID, Type: "related_to"}).Error)
	require.NoError(t, db.Create(&models.LifeEvent{UserID: user.ID, EntityID: contact.VCardUID, Type: "custom"}).Error)

	household := models.Household{UserID: user.ID, Name: "h", Type: "other"}
	require.NoError(t, db.Create(&household).Error)
	require.NoError(t, db.Create(&models.HouseholdMember{HouseholdID: household.ID, UserID: user.ID, MemberVCardUID: contact.VCardUID, Role: "adult"}).Error)

	circle := models.Circle{UserID: user.ID, Name: "c"}
	require.NoError(t, db.Create(&circle).Error)
	require.NoError(t, db.Create(&models.CircleMember{CircleID: circle.ID, UserID: user.ID, MemberVCardUID: contact.VCardUID}).Error)

	tag := models.Tag{UserID: user.ID, Name: "t"}
	require.NoError(t, db.Create(&tag).Error)
	require.NoError(t, db.Create(&models.ContactTag{TagID: tag.ID, UserID: user.ID, ContactVCardUID: contact.VCardUID}).Error)

	fieldDef := models.FieldDefinition{UserID: user.ID, Label: "f", Key: "f", Target: "contact", Type: "text"}
	require.NoError(t, db.Create(&fieldDef).Error)
	require.NoError(t, db.Create(&models.FieldValue{FieldDefinitionID: fieldDef.ID, UserID: user.ID, EntityID: contact.VCardUID, Value: json.RawMessage(`"v"`)}).Error)

	sub := models.ContactSubscription{UserID: user.ID, Name: "sub", URL: "https://example.com/dav"}
	require.NoError(t, db.Create(&sub).Error)
	require.NoError(t, db.Create(&models.ContactSyncLink{SubscriptionID: sub.ID, UserID: user.ID, ContactID: contact.ID, Href: "/dav/1.vcf"}).Error)

	require.NoError(t, db.Create(&models.ReminderCompletion{UserID: user.ID, ContactID: contact.ID, Message: "done", CompletedAt: time.Now()}).Error)

	router.DELETE("/contacts/:id", DeleteContact)

	req, _ := http.NewRequest("DELETE", "/contacts/"+strconv.Itoa(int(contact.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	assertCount := func(name string, model any, want int64, where string, args ...any) {
		t.Helper()
		var count int64
		require.NoError(t, db.Model(model).Where(where, args...).Count(&count).Error)
		assert.Equal(t, want, count, "%s row count mismatch after DeleteContact", name)
	}

	assertCount("RelationshipEdge", &models.RelationshipEdge{}, 0, "user_id = ?", user.ID)
	assertCount("LifeEvent", &models.LifeEvent{}, 0, "user_id = ?", user.ID)
	assertCount("HouseholdMember", &models.HouseholdMember{}, 0, "user_id = ?", user.ID)
	assertCount("CircleMember", &models.CircleMember{}, 0, "user_id = ?", user.ID)
	assertCount("ContactTag", &models.ContactTag{}, 0, "user_id = ?", user.ID)
	assertCount("FieldValue", &models.FieldValue{}, 0, "user_id = ?", user.ID)
	assertCount("ContactSyncLink", &models.ContactSyncLink{}, 0, "user_id = ?", user.ID)
	assertCount("ReminderCompletion", &models.ReminderCompletion{}, 0, "user_id = ?", user.ID)

	// The shared containers themselves must survive
	assertCount("Household", &models.Household{}, 1, "id = ?", household.ID)
	assertCount("Circle", &models.Circle{}, 1, "id = ?", circle.ID)
	assertCount("Tag", &models.Tag{}, 1, "id = ?", tag.ID)
	assertCount("FieldDefinition", &models.FieldDefinition{}, 1, "id = ?", fieldDef.ID)
}

// TestDeleteContact_ClearsRelatedContactIDOnOtherRelationships is part of the
// same regression coverage: a Relationship row belonging to a DIFFERENT,
// still-existing contact may optionally link to the deleted contact via
// RelatedContactID. That link must be cleared, but the relationship row
// itself (which belongs to the other contact, not the deleted one) must
// survive.
func TestDeleteContact_ClearsRelatedContactIDOnOtherRelationships(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	victim := models.Contact{UserID: user.ID, Firstname: "Victim", Lastname: "ToDelete"}
	require.NoError(t, db.Create(&victim).Error)

	survivor := models.Contact{UserID: user.ID, Firstname: "Survivor", Lastname: "Stays"}
	require.NoError(t, db.Create(&survivor).Error)

	rel := models.Relationship{
		UserID:           user.ID,
		Name:             "Victim ToDelete",
		Type:             "friend",
		ContactID:        survivor.ID,
		RelatedContactID: &victim.ID,
	}
	require.NoError(t, db.Create(&rel).Error)

	router.DELETE("/contacts/:id", DeleteContact)

	req, _ := http.NewRequest("DELETE", "/contacts/"+strconv.Itoa(int(victim.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var reloaded models.Relationship
	require.NoError(t, db.First(&reloaded, rel.ID).Error)
	assert.Nil(t, reloaded.RelatedContactID, "related_contact_id should be cleared after the linked contact is deleted")
	assert.Equal(t, survivor.ID, reloaded.ContactID, "the relationship row itself must survive since it belongs to the surviving contact")
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

func TestDeleteContactCleansUpPhotos(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.DELETE("/contacts/:id", DeleteContact)

	// Create a temporary directory for test photos
	tempDir, err := os.MkdirTemp("", "photo_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Set the PROFILE_PHOTO_DIR environment variable
	originalDir := os.Getenv("PROFILE_PHOTO_DIR")
	os.Setenv("PROFILE_PHOTO_DIR", tempDir)
	defer os.Setenv("PROFILE_PHOTO_DIR", originalDir)

	// Create test photo files
	photoName := "test_photo.jpg"
	thumbnailName := "test_thumbnail.jpg"
	photoPath := filepath.Join(tempDir, photoName)
	thumbnailPath := filepath.Join(tempDir, thumbnailName)

	err = os.WriteFile(photoPath, []byte("fake photo data"), 0644)
	assert.NoError(t, err)
	err = os.WriteFile(thumbnailPath, []byte("fake thumbnail data"), 0644)
	assert.NoError(t, err)

	// Verify files exist
	_, err = os.Stat(photoPath)
	assert.NoError(t, err, "Photo file should exist before deletion")
	_, err = os.Stat(thumbnailPath)
	assert.NoError(t, err, "Thumbnail file should exist before deletion")

	// Create a contact with photo references
	contact := models.Contact{
		UserID:         user.ID,
		Firstname:      "Alice",
		Lastname:       "Johnson",
		Photo:          photoName,
		PhotoThumbnail: thumbnailName,
	}
	db.Create(&contact)

	// Make the request to delete the contact
	req, _ := http.NewRequest("DELETE", "/contacts/"+strconv.Itoa(int(contact.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var respBody map[string]string
	json.Unmarshal(w.Body.Bytes(), &respBody)
	assert.Equal(t, "Contact deleted", respBody["message"])

	// Verify photo files are deleted
	_, err = os.Stat(photoPath)
	assert.True(t, os.IsNotExist(err), "Photo file should be deleted")
	_, err = os.Stat(thumbnailPath)
	assert.True(t, os.IsNotExist(err), "Thumbnail file should be deleted")
}

func TestDeleteContactWithNoPhotos(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.DELETE("/contacts/:id", DeleteContact)

	// Create a contact without photos
	contact := models.Contact{
		UserID:         user.ID,
		Firstname:      "Bob",
		Lastname:       "Smith",
		Photo:          "",
		PhotoThumbnail: "",
	}
	db.Create(&contact)

	// Make the request to delete the contact (should not error even without photos)
	req, _ := http.NewRequest("DELETE", "/contacts/"+strconv.Itoa(int(contact.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var respBody map[string]string
	json.Unmarshal(w.Body.Bytes(), &respBody)
	assert.Equal(t, "Contact deleted", respBody["message"])
}
