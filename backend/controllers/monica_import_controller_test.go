package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"meerkat/config"
	"meerkat/i18n"
	"meerkat/models"
	"meerkat/monica"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// fakeMonicaServer serves a small canned Monica v4 account.
func fakeMonicaServer(t *testing.T) *httptest.Server {
	t.Helper()

	paged := func(w http.ResponseWriter, items []map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{
			"data": items,
			"meta": map[string]int{"current_page": 1, "last_page": 1, "total": len(items)},
		})
	}

	contacts := []map[string]any{
		{
			"id": 101, "first_name": "Ada", "last_name": "Lovelace", "nickname": "",
			"gender": "Woman", "gender_type": "F", "is_partial": false, "is_starred": true,
			"description": "Mathematician", "food_preferences": "Vegetarian",
			"information": map[string]any{
				"dates": map[string]any{
					"birthdate": map[string]any{"date": "1990-12-10T00:00:00Z", "is_age_based": false, "is_year_unknown": false},
				},
				"career": map[string]any{"job": "Analyst", "company": "Engines Ltd"},
				"avatar": map[string]any{"url": nil, "source": "default"},
			},
			"tags":      []map[string]any{{"name": "friends"}},
			"addresses": []map[string]any{{"name": "Home", "street": "1 Main St", "city": "London", "province": "", "postal_code": "SW1", "country": map[string]any{"name": "UK"}}},
			"contactFields": []map[string]any{
				{"content": "ada@example.com", "contact_field_type": map[string]any{"name": "Email", "protocol": "mailto:", "type": "email"}},
			},
		},
		{
			"id": 102, "first_name": "Bob", "last_name": "Existing",
			"gender_type": "M", "is_partial": false,
			"information": map[string]any{},
		},
		{
			"id": 103, "first_name": "Stub", "last_name": "Person", "is_partial": true,
			"information": map[string]any{},
		},
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer good-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch {
		case r.URL.Path == "/api/contacts":
			paged(w, contacts)
		case r.URL.Path == "/api/contacts/101/relationships":
			paged(w, []map[string]any{{
				"relationship_type": map[string]any{"name": "daughter"},
				"contact_is":        map[string]any{"id": 101},
				"of_contact":        map[string]any{"id": 102, "first_name": "Bob", "last_name": "Existing", "complete_name": "Bob Existing"},
			}})
		case strings.HasPrefix(r.URL.Path, "/api/contacts/") && strings.HasSuffix(r.URL.Path, "/relationships"):
			paged(w, nil)
		case r.URL.Path == "/api/activities":
			paged(w, []map[string]any{
				{
					"id": 1, "summary": "Lunch", "description": "At the pub", "happened_at": "2023-05-01",
					"attendees": map[string]any{"contacts": []map[string]any{{"id": 101}, {"id": 102}}},
				},
				{
					"id": 2, "summary": "Stub only", "happened_at": "2023-05-02",
					"attendees": map[string]any{"contacts": []map[string]any{{"id": 103}}},
				},
			})
		case r.URL.Path == "/api/notes":
			paged(w, []map[string]any{
				{"body": "Loves poetry", "created_at": "2023-01-02T10:00:00Z", "contact": map[string]any{"id": 101}},
			})
		case r.URL.Path == "/api/reminders":
			future := time.Now().AddDate(1, 0, 0).Format("2006-01-02")
			paged(w, []map[string]any{
				{"title": "Wish happy birthday", "frequency_type": "year", "frequency_number": 1, "initial_date": future, "contact": map[string]any{"id": 101}},
			})
		case r.URL.Path == "/api/calls":
			paged(w, []map[string]any{
				{"content": "Talked about work", "called_at": "2023-04-01 12:00:00", "contact": map[string]any{"id": 101}},
			})
		case r.URL.Path == "/api/tasks", r.URL.Path == "/api/gifts", r.URL.Path == "/api/debts":
			paged(w, nil)
		default:
			t.Errorf("unexpected Monica API path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func setupMonicaRouter(t *testing.T) (*gorm.DB, *gin.Engine, *config.Config) {
	t.Helper()
	monica.DisableRateLimitForTesting()
	assert.NoError(t, i18n.Init())

	db, router := setupRouter()
	cfg := &config.Config{ProfilePhotoDir: t.TempDir()}

	router.POST("/import/monica/connect", withValidated(func() any { return &models.MonicaConnectRequest{} }), func(c *gin.Context) {
		ConnectMonicaImport(c, cfg)
	})
	router.POST("/import/monica/fetch", withValidated(func() any { return &models.MonicaFetchRequest{} }), StartMonicaFetch)
	router.GET("/import/monica/status", GetMonicaImportStatus)
	router.GET("/import/monica/preview", GetMonicaImportPreview)
	router.POST("/import/monica/confirm", withValidated(func() any { return &models.MonicaConfirmRequest{} }), func(c *gin.Context) {
		ConfirmMonicaImport(c, cfg)
	})
	router.DELETE("/import/monica/session", CancelMonicaImport)

	return db, router, cfg
}

func postJSON(router *gin.Engine, path string, payload any) *httptest.ResponseRecorder {
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func getPath(router *gin.Engine, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// runMonicaImportFlow walks connect → fetch → poll → preview and returns the
// session ID and preview.
func runMonicaImportFlow(t *testing.T, router *gin.Engine, serverURL string) (string, models.MonicaPreviewResponse) {
	t.Helper()

	w := postJSON(router, "/import/monica/connect", models.MonicaConnectRequest{BaseURL: serverURL, APIToken: "good-token"})
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var connectResp models.MonicaConnectResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &connectResp))
	assert.Equal(t, 3, connectResp.Totals.Contacts)
	assert.NotEmpty(t, connectResp.SessionID)

	w = postJSON(router, "/import/monica/fetch", models.MonicaFetchRequest{
		SessionID: connectResp.SessionID, IncludeRelationships: true, IncludeExtras: true,
	})
	assert.Equal(t, http.StatusAccepted, w.Code, w.Body.String())

	deadline := time.Now().Add(10 * time.Second)
	var status models.MonicaImportStatus
	for {
		w = getPath(router, "/import/monica/status?session_id="+connectResp.SessionID)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &status))
		if status.Phase == models.MonicaPhaseReady || status.Phase == models.MonicaPhaseFailed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("fetch did not finish, stuck in phase %s", status.Phase)
		}
		time.Sleep(10 * time.Millisecond)
	}
	assert.Equal(t, models.MonicaPhaseReady, status.Phase, status.Error)

	w = getPath(router, "/import/monica/preview?session_id="+connectResp.SessionID)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var preview models.MonicaPreviewResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &preview))
	return connectResp.SessionID, preview
}

func TestMonicaImportEndToEnd(t *testing.T) {
	server := fakeMonicaServer(t)
	defer server.Close()

	db, router, _ := setupMonicaRouter(t)
	var user models.User
	db.First(&user)

	// Seed an existing contact that duplicates Monica contact 102 by name.
	existing := models.Contact{UserID: user.ID, Firstname: "Bob", Lastname: "Existing"}
	assert.NoError(t, db.Create(&existing).Error)

	sessionID, preview := runMonicaImportFlow(t, router, server.URL)

	// The partial contact (id 103) must be filtered out.
	assert.Equal(t, 2, preview.TotalRows)
	assert.Equal(t, 1, preview.DuplicateCount)

	rowByName := map[string]models.MonicaImportRowPreview{}
	for _, row := range preview.Rows {
		rowByName[fmt.Sprint(row.ParsedContact["firstname"])] = row
	}
	adaRow := rowByName["Ada"]
	bobRow := rowByName["Bob"]
	assert.Equal(t, "add", adaRow.SuggestedAction)
	assert.Equal(t, "update", bobRow.SuggestedAction)
	assert.NotNil(t, bobRow.DuplicateMatch)
	assert.Equal(t, 1, adaRow.Related.Activities)
	assert.Equal(t, 1, adaRow.Related.Notes)
	assert.Equal(t, 1, adaRow.Related.Reminders)
	assert.Equal(t, 1, adaRow.Related.Relationships)
	assert.Equal(t, 1, adaRow.Related.ExtraNotes)

	// Confirm: add Ada, update Bob.
	w := postJSON(router, "/import/monica/confirm", models.MonicaConfirmRequest{
		SessionID: sessionID,
		Actions: []models.RowImportAction{
			{RowIndex: adaRow.RowIndex, Action: "add"},
			{RowIndex: bobRow.RowIndex, Action: "update"},
		},
	})
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var result models.MonicaImportResult
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Equal(t, 1, result.Created)
	assert.Equal(t, 1, result.Updated)
	assert.Equal(t, 1, result.ActivitiesCreated)
	assert.Equal(t, 2, result.NotesCreated) // "Loves poetry" + the call note (merge note is separate)
	assert.Equal(t, 1, result.RemindersCreated)
	assert.Equal(t, 1, result.RelationshipsCreated)
	assert.Equal(t, 0, result.PhotosQueued)

	// Contact content
	var ada models.Contact
	assert.NoError(t, db.Where("firstname = ?", "Ada").First(&ada).Error)
	assert.Equal(t, "female", ada.Gender)
	assert.Equal(t, "1990-12-10", ada.Birthday)
	assert.Equal(t, "ada@example.com", ada.Email)
	assert.Equal(t, []string{"friends"}, ada.Circles)
	assert.Equal(t, "Analyst", ada.JobTitle)
	assert.Equal(t, "Engines Ltd", ada.Organization)
	assert.Equal(t, "yes", ada.CustomFields["Starred"])

	// Activity linked to both contacts through the join table.
	var activity models.Activity
	assert.NoError(t, db.Preload("Contacts").Where("title = ?", "Lunch").First(&activity).Error)
	assert.Len(t, activity.Contacts, 2)

	// Relationship on Ada linked to Bob's Meerkat record.
	var relationship models.Relationship
	assert.NoError(t, db.Where("contact_id = ?", ada.ID).First(&relationship).Error)
	assert.Equal(t, "daughter", relationship.Type)
	assert.Equal(t, "Bob Existing", relationship.Name)
	assert.NotNil(t, relationship.RelatedContactID)
	assert.Equal(t, existing.ID, *relationship.RelatedContactID)

	// Reminder and notes on Ada.
	var reminderCount, noteCount int64
	db.Model(&models.Reminder{}).Where("contact_id = ?", ada.ID).Count(&reminderCount)
	assert.Equal(t, int64(1), reminderCount)
	db.Model(&models.Note{}).Where("contact_id = ?", ada.ID).Count(&noteCount)
	assert.Equal(t, int64(2), noteCount)

	// Custom field names registered on the user.
	var updatedUser models.User
	db.First(&updatedUser, user.ID)
	assert.Contains(t, updatedUser.CustomFieldNames, "Starred")

	// The stub contact must not exist.
	var stubCount int64
	db.Model(&models.Contact{}).Where("firstname = ?", "Stub").Count(&stubCount)
	assert.Equal(t, int64(0), stubCount)
}

func TestMonicaImportRerunIsIdempotent(t *testing.T) {
	server := fakeMonicaServer(t)
	defer server.Close()

	db, router, _ := setupMonicaRouter(t)

	runImport := func() {
		sessionID, preview := runMonicaImportFlow(t, router, server.URL)
		actions := make([]models.RowImportAction, 0, len(preview.Rows))
		for _, row := range preview.Rows {
			action := row.SuggestedAction
			if action == "skip" {
				action = "add"
			}
			actions = append(actions, models.RowImportAction{RowIndex: row.RowIndex, Action: action})
		}
		w := postJSON(router, "/import/monica/confirm", models.MonicaConfirmRequest{SessionID: sessionID, Actions: actions})
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	}

	runImport()
	runImport() // second run updates the same contacts and must not duplicate related entities

	var contactCount, activityCount, noteCount, reminderCount, relationshipCount int64
	db.Model(&models.Contact{}).Count(&contactCount)
	db.Model(&models.Activity{}).Count(&activityCount)
	// Merge notes from the update pass are expected; imported notes must not double.
	db.Model(&models.Note{}).Where("content IN ?", []string{"Loves poetry"}).Count(&noteCount)
	db.Model(&models.Reminder{}).Count(&reminderCount)
	db.Model(&models.Relationship{}).Count(&relationshipCount)

	assert.Equal(t, int64(2), contactCount)
	assert.Equal(t, int64(1), activityCount)
	assert.Equal(t, int64(1), noteCount)
	assert.Equal(t, int64(1), reminderCount)
	assert.Equal(t, int64(1), relationshipCount)
}

func TestMonicaConnectRejectsBadToken(t *testing.T) {
	server := fakeMonicaServer(t)
	defer server.Close()

	_, router, _ := setupMonicaRouter(t)
	w := postJSON(router, "/import/monica/connect", models.MonicaConnectRequest{BaseURL: server.URL, APIToken: "wrong"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "api_token")
}

func TestMonicaSessionOwnershipEnforced(t *testing.T) {
	server := fakeMonicaServer(t)
	defer server.Close()

	db, router, _ := setupMonicaRouter(t)

	w := postJSON(router, "/import/monica/connect", models.MonicaConnectRequest{BaseURL: server.URL, APIToken: "good-token"})
	assert.Equal(t, http.StatusOK, w.Code)
	var connectResp models.MonicaConnectResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &connectResp))

	// A second router with a different user must not see the session.
	otherUser := models.User{Username: "other", Password: "password123", Email: "other@example.com"}
	assert.NoError(t, db.Create(&otherUser).Error)
	otherRouter := gin.New()
	otherRouter.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("userID", otherUser.ID)
		c.Next()
	})
	otherRouter.GET("/import/monica/status", GetMonicaImportStatus)

	w = getPath(otherRouter, "/import/monica/status?session_id="+connectResp.SessionID)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Unknown sessions 404.
	w = getPath(router, "/import/monica/status?session_id=doesnotexist")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMonicaCancelDropsSession(t *testing.T) {
	server := fakeMonicaServer(t)
	defer server.Close()

	_, router, _ := setupMonicaRouter(t)

	w := postJSON(router, "/import/monica/connect", models.MonicaConnectRequest{BaseURL: server.URL, APIToken: "good-token"})
	assert.Equal(t, http.StatusOK, w.Code)
	var connectResp models.MonicaConnectResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &connectResp))

	req := httptest.NewRequest(http.MethodDelete, "/import/monica/session?session_id="+connectResp.SessionID, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	w = getPath(router, "/import/monica/status?session_id="+connectResp.SessionID)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
