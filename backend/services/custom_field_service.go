package services

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"mycorrhizal/middleware"
	"mycorrhizal/models"
)

// ValidateFieldValue checks a raw JSON value against a FieldDefinition's
// Type + Constraints (docs/fork-plan/94-custom-fields.md §94.4: "Validation
// on write is driven by the definition's type + constraints, routed to the
// matching validator"). Lives in services, not models, because the
// validators it reuses (middleware.ValidateVar/ValidateEmail) live in
// middleware, and middleware already imports models -- models importing
// middleware back would be a cycle. services already imports middleware
// today (import_service.go's middleware.ValidateEmail call), so this is the
// correct home, matching household_service.go/birthday_service.go's
// precedent for algorithmic logic that isn't a model or a controller.
//
// When def.Constraints.Multi is set (§94.4's list<T>), value must be a JSON
// array and every element is validated against the same scalar rule.
func ValidateFieldValue(def models.FieldDefinition, value json.RawMessage) error {
	if !def.Constraints.Multi {
		return validateScalar(def.Type, def.Constraints, value)
	}

	var elements []json.RawMessage
	if err := json.Unmarshal(value, &elements); err != nil {
		return fmt.Errorf("field %q: expected a list value", def.Key)
	}
	for i, el := range elements {
		if err := validateScalar(def.Type, def.Constraints, el); err != nil {
			return fmt.Errorf("field %q, item %d: %w", def.Key, i, err)
		}
	}
	return nil
}

func validateScalar(fieldType string, c models.FieldConstraints, raw json.RawMessage) error {
	switch fieldType {
	case models.FieldTypeString, models.FieldTypeText:
		return validateStringConstraints(c, raw)
	case models.FieldTypeNumber:
		return validateNumber(c, raw)
	case models.FieldTypeBoolean:
		var b bool
		if err := json.Unmarshal(raw, &b); err != nil {
			return fmt.Errorf("expected a boolean value")
		}
		return nil
	case models.FieldTypeDate:
		s, err := unmarshalString(raw)
		if err != nil {
			return err
		}
		if !middleware.ValidateVar(s, "birthday") {
			return fmt.Errorf("invalid date %q (expected YYYY-MM-DD or --MM-DD)", s)
		}
		return nil
	case models.FieldTypeDateTime:
		s, err := unmarshalString(raw)
		if err != nil {
			return err
		}
		if _, err := time.Parse(time.RFC3339, s); err != nil {
			return fmt.Errorf("invalid datetime %q (expected RFC3339)", s)
		}
		return nil
	case models.FieldTypeURI:
		s, err := unmarshalString(raw)
		if err != nil {
			return err
		}
		if !middleware.ValidateVar(s, "uri,safeurl") {
			return fmt.Errorf("invalid uri %q", s)
		}
		return nil
	case models.FieldTypeEmail:
		s, err := unmarshalString(raw)
		if err != nil {
			return err
		}
		if !middleware.ValidateEmail(s) {
			return fmt.Errorf("invalid email %q", s)
		}
		return nil
	case models.FieldTypePhone:
		s, err := unmarshalString(raw)
		if err != nil {
			return err
		}
		if !middleware.ValidateVar(s, "phone") {
			return fmt.Errorf("invalid phone %q", s)
		}
		return nil
	case models.FieldTypeEnum:
		s, err := unmarshalString(raw)
		if err != nil {
			return err
		}
		if len(c.Values) == 0 {
			return fmt.Errorf("enum field has no allowed values configured")
		}
		if !middleware.ValidateVar(s, "oneof="+strings.Join(c.Values, " ")) {
			return fmt.Errorf("%q is not one of the allowed values %v", s, c.Values)
		}
		return nil
	default:
		return fmt.Errorf("unknown field type %q", fieldType)
	}
}

func unmarshalString(raw json.RawMessage) (string, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", fmt.Errorf("expected a string value")
	}
	return s, nil
}

func validateStringConstraints(c models.FieldConstraints, raw json.RawMessage) error {
	s, err := unmarshalString(raw)
	if err != nil {
		return err
	}
	if c.MaxLength != nil && len(s) > *c.MaxLength {
		return fmt.Errorf("value exceeds maximum length of %d", *c.MaxLength)
	}
	if c.Pattern != "" {
		matched, err := regexp.MatchString(c.Pattern, s)
		if err != nil {
			return fmt.Errorf("field definition has an invalid pattern: %w", err)
		}
		if !matched {
			return fmt.Errorf("value does not match the required pattern")
		}
	}
	return nil
}

func validateNumber(c models.FieldConstraints, raw json.RawMessage) error {
	var n float64
	if err := json.Unmarshal(raw, &n); err != nil {
		return fmt.Errorf("expected a number value")
	}
	var tags []string
	if c.Min != nil {
		tags = append(tags, fmt.Sprintf("min=%v", *c.Min))
	}
	if c.Max != nil {
		tags = append(tags, fmt.Sprintf("max=%v", *c.Max))
	}
	if len(tags) == 0 {
		return nil
	}
	if !middleware.ValidateVar(n, strings.Join(tags, ",")) {
		return fmt.Errorf("value %v is out of the allowed range", n)
	}
	return nil
}
