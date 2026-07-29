package controllers

// This file exercises models.ContactRecordInput through the REAL
// middleware.ValidateJSONMiddleware + apperrors.ErrorHandlerMiddleware chain
// (per docs/fork-plan/45-test-coverage-closure.md, work package TC-1.6),
// rather than the withValidated test helper used elsewhere in
// contact_controller_test.go, which bypasses validation/JSON-bind entirely.
// It follows the exact wiring pattern established in
// middleware/validation_integration_test.go, just targeting the real
// CreateContact/UpdateContact handlers and the real ContactRecordInput
// struct instead of a synthetic one.
//
// Per ContactRecordInput's own doc comment (models/contact_summary.go),
// Card/CRM/Passthrough deliberately carry NO validate tags — only Gender
// does (`validate:"omitempty,oneof=male female other prefer_not_to_say"`).
// That is a degradation-policy design choice: unrecognized nested enum
// values (NameComponent.Kind, PersonalInfo.Kind, ...) must be accepted, not
// rejected. TestCreateContact_RealValidation_UnusualNestedDataAccepted below
// proves that holds true end-to-end, including verifying gin's JSON decoder
// itself is not configured with DisallowUnknownFields anywhere in the chain
// (which would silently defeat the policy without any validate tag being
// involved at all).

import (
	"bytes"
	"encoding/json"
	"mycorrhizal/contactmodel"
	apperrors "mycorrhizal/errors"
	"mycorrhizal/middleware"
	"mycorrhizal/models"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newValidatedContactRouter wires a fresh setupRouter() with the real
// validation + error-handling middleware in front of CreateContact/
// UpdateContact, mirroring middleware/validation_integration_test.go's
// established router.Use(apperrors.ErrorHandlerMiddleware()) +
// ValidateJSONMiddleware(&Struct{}) pattern.
func newValidatedContactRouter() (*gorm.DB, *gin.Engine) {
	db, router := setupRouter()
	router.Use(apperrors.ErrorHandlerMiddleware())
	router.POST("/contacts", middleware.ValidateJSONMiddleware(&models.ContactRecordInput{}), CreateContact)
	router.PUT("/contacts/:id", middleware.ValidateJSONMiddleware(&models.ContactRecordInput{}), UpdateContact)
	return db, router
}

func doJSONRequest(router *gin.Engine, method, path string, body []byte) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// baseValidCardJSON is a minimal structurally-valid Card carrying a given
// name, so CreateContact's post-conversion "Firstname required" check never
// fires and only the behavior under test (gender/JSON-shape) is exercised.
func baseValidCardJSON() contactmodel.Card {
	return contactmodel.Card{
		Name: &contactmodel.Name{Components: []contactmodel.NameComponent{
			{Kind: "given", Value: "Test"},
		}},
	}
}

func TestCreateContact_RealValidation_InvalidGender(t *testing.T) {
	_, router := newValidatedContactRouter()

	input := models.ContactRecordInput{
		Gender: "unspecified", // not in male|female|other|prefer_not_to_say
		Card:   baseValidCardJSON(),
	}
	body, _ := json.Marshal(input)

	w := doJSONRequest(router, "POST", "/contacts", body)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp apperrors.ErrorResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
	assert.Contains(t, resp.Error.Details, "Gender")
}

func TestCreateContact_RealValidation_ValidGenderValues(t *testing.T) {
	validGenders := []string{"male", "female", "other", "prefer_not_to_say"}

	for _, gender := range validGenders {
		t.Run(gender, func(t *testing.T) {
			_, router := newValidatedContactRouter()

			input := models.ContactRecordInput{
				Gender: gender,
				Card:   baseValidCardJSON(),
			}
			body, _ := json.Marshal(input)

			w := doJSONRequest(router, "POST", "/contacts", body)

			assert.Equal(t, http.StatusCreated, w.Code)

			var respBody map[string]any
			assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
			contact := respBody["contact"].(map[string]any)
			assert.Equal(t, gender, contact["gender"])
		})
	}
}

// TestCreateContact_RealValidation_EmptyGenderAccepted proves Gender's
// `omitempty` keeps an empty/omitted gender accepted through the real
// validator, not just when Gender happens to be a recognized value.
func TestCreateContact_RealValidation_EmptyGenderAccepted(t *testing.T) {
	_, router := newValidatedContactRouter()

	// Gender omitted entirely from the JSON body (not just empty-string).
	body := []byte(`{"card":{"name":{"components":[{"kind":"given","value":"NoGender"}]}}}`)

	w := doJSONRequest(router, "POST", "/contacts", body)

	assert.Equal(t, http.StatusCreated, w.Code)

	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	contact := respBody["contact"].(map[string]any)
	assert.Equal(t, "", contact["gender"])
}

// TestCreateContact_RealValidation_ThinEntityAccepted is WP-82's locking-in
// test for the "nothing but name required" thin-entity invariant (docs/
// fork-plan/90-vision-and-reconciliation.md D3, docs/fork-plan/
// 91-envelope-data-model.md §91.1) — investigation during WP-82's planning
// found this already worked end-to-end (both here and in the frontend's
// AddContactDialog), so this asserts that finding rather than building
// anything new. A request carrying only a given-name component — no email,
// phone, address, gender, or any CRM field — must succeed and produce a
// Contact with that name and nothing else, exactly what a pet or a minor
// child's relationship-graph node (WP-80/81) needs to exist as.
func TestCreateContact_RealValidation_ThinEntityAccepted(t *testing.T) {
	db, router := newValidatedContactRouter()

	body := []byte(`{"card":{"name":{"components":[{"kind":"given","value":"Fluffy"}]}}}`)

	w := doJSONRequest(router, "POST", "/contacts", body)

	if !assert.Equal(t, http.StatusCreated, w.Code, "a name-only contact must be accepted, body: %s", w.Body.String()) {
		return
	}

	// ContactRecordResponse has no top-level firstname/lastname (the name
	// lives in card.name.components) — check the response shape actually
	// returned, not an assumed flat one.
	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	contact := respBody["contact"].(map[string]any)
	assert.Equal(t, "", contact["gender"])
	card := contact["card"].(map[string]any)
	name := card["name"].(map[string]any)
	components := name["components"].([]any)
	require.Len(t, components, 1)
	given := components[0].(map[string]any)
	assert.Equal(t, "given", given["kind"])
	assert.Equal(t, "Fluffy", given["value"])

	// The stronger assertion: the persisted row itself has nothing but the
	// name — this is what actually makes a bare-name Contact valid as a
	// WP-80/81 relationship-graph node.
	var stored models.Contact
	require.NoError(t, db.First(&stored).Error)
	assert.Equal(t, "Fluffy", stored.Firstname)
	assert.Equal(t, "", stored.Lastname)
	assert.Equal(t, "", stored.Email)
	assert.Equal(t, "", stored.Phone)
	assert.Equal(t, "", stored.Gender)
}

// TestCreateContact_RealValidation_KindAccepted proves CRMEnvelope.Kind
// (WP-82, contactmodel/envelope.go) round-trips through the real create path
// with zero extra wiring — CRM is copied wholesale between Contact.CRM and
// contactmodel.Record.Envelope, so a new field needs no code change anywhere
// in that path, only in the struct definition itself. Also exercises an
// unrecognized value ("robot") to confirm the field is deliberately
// unvalidated, matching every other CRMEnvelope field's degradation policy
// (see this file's header) rather than a hardcoded enum.
func TestCreateContact_RealValidation_KindAccepted(t *testing.T) {
	for _, kind := range []string{"pet", "animal", "robot"} {
		t.Run(kind, func(t *testing.T) {
			_, router := newValidatedContactRouter()

			input := models.ContactRecordInput{
				Card: baseValidCardJSON(),
				CRM:  contactmodel.CRMEnvelope{Kind: kind},
			}
			body, _ := json.Marshal(input)

			w := doJSONRequest(router, "POST", "/contacts", body)

			if !assert.Equal(t, http.StatusCreated, w.Code, "kind=%q must be accepted, body: %s", kind, w.Body.String()) {
				return
			}

			var respBody map[string]any
			assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
			contact := respBody["contact"].(map[string]any)
			crm := contact["crm"].(map[string]any)
			assert.Equal(t, kind, crm["kind"])
		})
	}
}

func TestCreateContact_RealValidation_MalformedJSON(t *testing.T) {
	_, router := newValidatedContactRouter()

	// Genuinely broken syntax: unbalanced braces, not merely semantically odd.
	malformed := []byte(`{"card": {"name": {"components": [{"kind": "given", "value": "Broken"}]}`)

	w := doJSONRequest(router, "POST", "/contacts", malformed)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp apperrors.ErrorResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
}

func TestUpdateContact_RealValidation_MalformedJSON(t *testing.T) {
	db, router := newValidatedContactRouter()

	var user models.User
	db.First(&user)

	contact := models.Contact{UserID: user.ID, Firstname: "Existing", Lastname: "Contact"}
	db.Create(&contact)

	// Unbalanced brackets/braces mixed together.
	malformed := []byte(`{"card": {"name": [}}`)

	w := doJSONRequest(router, "PUT", "/contacts/"+strconv.Itoa(int(contact.ID)), malformed)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp apperrors.ErrorResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
}

// TestCreateContact_RealValidation_UnusualNestedDataAccepted is the key
// positive test for TC-1.6: structurally-valid JSON carrying UNUSUAL (not
// malformed) values nested under Card and Passthrough must be ACCEPTED, not
// rejected — proving the "no validate tags on nested Card/CRM/Passthrough"
// degradation policy holds end-to-end through the real middleware chain.
//
// It also deliberately includes an unrecognized TOP-LEVEL JSON field
// ("unexpected_top_level_field") alongside gender/card/crm/passthrough. If
// gin's ShouldBindJSON were ever configured with DisallowUnknownFields
// (directly or via gin.EnableJsonDecoderDisallowUnknownFields), this exact
// request would fail to bind and the test would catch it — confirmed by
// grepping the codebase for DisallowUnknownFields/
// EnableDecoderDisallowUnknownFields, which found zero occurrences.
func TestCreateContact_RealValidation_UnusualNestedDataAccepted(t *testing.T) {
	_, router := newValidatedContactRouter()

	body := []byte(`{
		"gender": "",
		"card": {
			"name": {
				"components": [
					{"kind": "given", "value": "Nova"},
					{"kind": "totally-unrecognized-name-kind", "value": "???"}
				]
			},
			"personalInfo": [
				{"kind": "totally-unrecognized-personal-info-kind", "value": "collects stamps"}
			]
		},
		"passthrough": {
			"jsContactProps": {
				"/some/unknown/pointer": {"totally": "unexpected", "structure": [1, 2, 3]}
			}
		},
		"unexpected_top_level_field": "should not break JSON binding"
	}`)

	w := doJSONRequest(router, "POST", "/contacts", body)

	if !assert.Equal(t, http.StatusCreated, w.Code, "unusual-but-valid nested data must be accepted, body: %s", w.Body.String()) {
		return
	}

	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	contact := respBody["contact"].(map[string]any)
	card := contact["card"].(map[string]any)

	name := card["name"].(map[string]any)
	components := name["components"].([]any)
	var sawUnrecognizedKind bool
	for _, comp := range components {
		c := comp.(map[string]any)
		if c["kind"] == "totally-unrecognized-name-kind" {
			sawUnrecognizedKind = true
		}
	}
	assert.True(t, sawUnrecognizedKind, "unrecognized NameComponent.Kind must round-trip, not be stripped or rejected")

	personalInfo, _ := card["personalInfo"].([]any)
	if assert.Len(t, personalInfo, 1) {
		pi := personalInfo[0].(map[string]any)
		assert.Equal(t, "totally-unrecognized-personal-info-kind", pi["kind"])
	}

	passthrough, _ := contact["passthrough"].(map[string]any)
	if assert.NotNil(t, passthrough, "passthrough with unknown nested keys must be preserved, not dropped") {
		jsContactProps, _ := passthrough["jsContactProps"].(map[string]any)
		assert.Contains(t, jsContactProps, "/some/unknown/pointer")
	}
}
