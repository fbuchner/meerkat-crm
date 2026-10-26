package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"meerkat/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContactMergeEndpoints(t *testing.T) {
	db, router := setupRouter()
	require.NoError(t, db.AutoMigrate(&models.ReminderCompletion{}, &models.ContactDuplicateDismissal{}))

	var user models.User
	db.First(&user)

	router.GET("/contacts/duplicates", GetDuplicateContacts)
	router.POST("/contacts/duplicates/dismiss", withValidated(func() any { return &models.DuplicateDismissInput{} }), DismissDuplicateContacts)
	router.GET("/contacts/:id/merge-preview", GetContactMergePreview)
	router.POST("/contacts/:id/merge", withValidated(func() any { return &models.ContactMergeInput{} }), MergeContact)

	target := models.Contact{UserID: user.ID, Firstname: "Anna", Lastname: "Schmidt", JobTitle: "Engineer"}
	source := models.Contact{UserID: user.ID, Firstname: "anna", Lastname: "schmidt", JobTitle: "Manager"}
	require.NoError(t, db.Create(&target).Error)
	require.NoError(t, db.Create(&source).Error)

	do := func(method, url string, body any) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			require.NoError(t, json.NewEncoder(&buf).Encode(body))
		}
		req, _ := http.NewRequest(method, url, &buf)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	w := do("GET", "/contacts/duplicates", nil)
	require.Equal(t, http.StatusOK, w.Code)
	var dupes struct {
		Duplicates []models.DuplicatePair `json:"duplicates"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dupes))
	require.Len(t, dupes.Duplicates, 1)
	assert.Equal(t, []string{"name"}, dupes.Duplicates[0].Reasons)

	w = do("GET", fmt.Sprintf("/contacts/%d/merge-preview?source_id=%d", target.ID, source.ID), nil)
	require.Equal(t, http.StatusOK, w.Code)
	var preview models.MergePreview
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &preview))
	assert.Equal(t, target.ID, preview.Target.ID)
	assert.Equal(t, []string{"firstname", "lastname", "job_title"},
		[]string{preview.Conflicts[0].Field, preview.Conflicts[1].Field, preview.Conflicts[2].Field})

	w = do("GET", fmt.Sprintf("/contacts/%d/merge-preview?source_id=%d", target.ID, target.ID), nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = do("GET", fmt.Sprintf("/contacts/%d/merge-preview?source_id=abc", target.ID), nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = do("GET", fmt.Sprintf("/contacts/%d/merge-preview?source_id=99999", target.ID), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	w = do("POST", fmt.Sprintf("/contacts/%d/merge", target.ID), map[string]any{
		"source_id":   source.ID,
		"resolutions": map[string]string{"job_title": "source"},
	})
	require.Equal(t, http.StatusOK, w.Code)
	var merged models.Contact
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &merged))
	assert.Equal(t, "Manager", merged.JobTitle)
	assert.Equal(t, "Anna", merged.Firstname)

	w = do("GET", "/contacts/duplicates", nil)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dupes))
	assert.Empty(t, dupes.Duplicates)

	w = do("POST", "/contacts/duplicates/dismiss", map[string]any{"contact_a_id": target.ID, "contact_b_id": source.ID})
	assert.Equal(t, http.StatusNotFound, w.Code, "the source was deleted by the merge")
}
