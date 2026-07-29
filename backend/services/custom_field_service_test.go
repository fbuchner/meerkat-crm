package services

import (
	"encoding/json"
	"testing"

	"mycorrhizal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func raw(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

func TestValidateFieldValue_String(t *testing.T) {
	maxLen := 5
	def := models.FieldDefinition{Key: "nickname", Type: models.FieldTypeString, Constraints: models.FieldConstraints{MaxLength: &maxLen}}

	assert.NoError(t, ValidateFieldValue(def, raw(t, "short")))
	assert.Error(t, ValidateFieldValue(def, raw(t, "way too long")), "exceeds MaxLength")
	assert.Error(t, ValidateFieldValue(def, raw(t, 42)), "wrong JSON type")
}

func TestValidateFieldValue_StringPattern(t *testing.T) {
	def := models.FieldDefinition{Key: "code", Type: models.FieldTypeString, Constraints: models.FieldConstraints{Pattern: `^[A-Z]{3}$`}}

	assert.NoError(t, ValidateFieldValue(def, raw(t, "ABC")))
	assert.Error(t, ValidateFieldValue(def, raw(t, "abc")), "lowercase does not match pattern")
}

func TestValidateFieldValue_Number(t *testing.T) {
	min, max := 0.0, 10.0
	def := models.FieldDefinition{Key: "rating", Type: models.FieldTypeNumber, Constraints: models.FieldConstraints{Min: &min, Max: &max}}

	assert.NoError(t, ValidateFieldValue(def, raw(t, 7)))
	assert.Error(t, ValidateFieldValue(def, raw(t, 11)), "above max")
	assert.Error(t, ValidateFieldValue(def, raw(t, -1)), "below min")
	assert.Error(t, ValidateFieldValue(def, raw(t, "not a number")))
}

func TestValidateFieldValue_Boolean(t *testing.T) {
	def := models.FieldDefinition{Key: "opted_in", Type: models.FieldTypeBoolean}

	assert.NoError(t, ValidateFieldValue(def, raw(t, true)))
	assert.Error(t, ValidateFieldValue(def, raw(t, "true")), "string is not a JSON boolean")
}

func TestValidateFieldValue_Date(t *testing.T) {
	def := models.FieldDefinition{Key: "anniversary", Type: models.FieldTypeDate}

	assert.NoError(t, ValidateFieldValue(def, raw(t, "2024-06-01")))
	assert.NoError(t, ValidateFieldValue(def, raw(t, "--06-01")), "year-optional partial date")
	assert.Error(t, ValidateFieldValue(def, raw(t, "June 1st")), "not ISO 8601")
}

func TestValidateFieldValue_DateTime(t *testing.T) {
	def := models.FieldDefinition{Key: "last_contacted", Type: models.FieldTypeDateTime}

	assert.NoError(t, ValidateFieldValue(def, raw(t, "2024-06-01T12:00:00Z")))
	assert.Error(t, ValidateFieldValue(def, raw(t, "2024-06-01")), "not RFC3339")
}

func TestValidateFieldValue_URI(t *testing.T) {
	def := models.FieldDefinition{Key: "profile_url", Type: models.FieldTypeURI}

	assert.NoError(t, ValidateFieldValue(def, raw(t, "https://example.com")))
	assert.Error(t, ValidateFieldValue(def, raw(t, "javascript:alert(1)")), "unsafe scheme rejected")
}

func TestValidateFieldValue_Email(t *testing.T) {
	def := models.FieldDefinition{Key: "alt_email", Type: models.FieldTypeEmail}

	assert.NoError(t, ValidateFieldValue(def, raw(t, "a@example.com")))
	assert.Error(t, ValidateFieldValue(def, raw(t, "not-an-email")))
}

func TestValidateFieldValue_Phone(t *testing.T) {
	def := models.FieldDefinition{Key: "fax", Type: models.FieldTypePhone}

	assert.NoError(t, ValidateFieldValue(def, raw(t, "+12025551234")))
	assert.Error(t, ValidateFieldValue(def, raw(t, "x")), "too short to be a phone number")
}

func TestValidateFieldValue_Enum(t *testing.T) {
	def := models.FieldDefinition{Key: "size", Type: models.FieldTypeEnum, Constraints: models.FieldConstraints{Values: []string{"S", "M", "L"}}}

	assert.NoError(t, ValidateFieldValue(def, raw(t, "M")))
	assert.Error(t, ValidateFieldValue(def, raw(t, "XL")), "not in the allowed values")
}

func TestValidateFieldValue_EnumWithNoAllowedValuesConfigured(t *testing.T) {
	def := models.FieldDefinition{Key: "size", Type: models.FieldTypeEnum}
	assert.Error(t, ValidateFieldValue(def, raw(t, "M")), "misconfigured definition (no Values) must not silently pass")
}

// §94.4's list<T>: Constraints.Multi turns any scalar type into a
// JSON-array-of-that-type, each element validated independently.
func TestValidateFieldValue_MultiValuedEnum(t *testing.T) {
	def := models.FieldDefinition{
		Key: "pronouns", Type: models.FieldTypeEnum,
		Constraints: models.FieldConstraints{Values: []string{"she/her", "he/him", "they/them"}, Multi: true},
	}

	assert.NoError(t, ValidateFieldValue(def, raw(t, []string{"she/her", "they/them"})))
	assert.Error(t, ValidateFieldValue(def, raw(t, []string{"she/her", "xyz"})), "one invalid element fails the whole list")
	assert.Error(t, ValidateFieldValue(def, raw(t, "she/her")), "a scalar is not a valid list value")
}

func TestValidateFieldValue_UnknownType(t *testing.T) {
	def := models.FieldDefinition{Key: "mystery", Type: "not-a-real-type"}
	assert.Error(t, ValidateFieldValue(def, raw(t, "anything")))
}
