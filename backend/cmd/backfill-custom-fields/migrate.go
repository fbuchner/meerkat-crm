package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"mycorrhizal/models"

	"gorm.io/gorm"
)

// definitionOutcome summarizes what migrateUserFieldDefinitions did (or, in
// dry run, would do) for one User.CustomFieldNames entry.
type definitionOutcome struct {
	// Skipped is non-empty when this name was not (and, in dry run, would
	// not be) migrated; its value is the human-readable reason.
	Skipped string
	// DefinitionID is populated only when write=true and the definition was
	// actually persisted (empty in dry run).
	DefinitionID string
}

// migrateUserFieldDefinitions migrates one entry of User.CustomFieldNames
// (models/user.go) into a models.FieldDefinition, per docs/fork-plan/
// 94-custom-fields.md §94.6: "Each User.CustomFieldNames entry -> a
// FieldDefinition{type:string, constraints:{}, projection:internal-only}."
//
// name becomes BOTH Label and Key verbatim, with no slugification: v1 has no
// separate machine-name concept (Contact.CustomFields is keyed by this exact
// string), so a lossless migration must preserve it exactly, letting the
// user rename Label later while Key (and therefore every existing
// FieldValue's join) stays stable.
//
// write=false performs every check but makes zero writes. Idempotent via the
// (user_id, key) unique index: a name already migrated is reported as
// skipped. Unlike migrateContactFieldValue below, there is no -force
// override here -- a migrated definition's Label/Key/Type never drifts from
// v1 (there is nothing in v1 to re-sync it from), so reprocessing one would
// be a no-op at best.
func migrateUserFieldDefinitions(db *gorm.DB, userID uint, name string, write bool) (definitionOutcome, error) {
	if name == "" {
		return definitionOutcome{Skipped: "empty custom field name"}, nil
	}

	var existing models.FieldDefinition
	err := db.Where("user_id = ? AND key = ?", userID, name).First(&existing).Error
	switch {
	case err == nil:
		return definitionOutcome{Skipped: fmt.Sprintf("already migrated to definition %s", existing.ID)}, nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		// Not yet migrated -- the common case.
	default:
		return definitionOutcome{}, fmt.Errorf("checking idempotency for user_id=%d key=%q: %w", userID, name, err)
	}

	if !write {
		return definitionOutcome{}, nil
	}

	def := models.FieldDefinition{
		UserID:      userID,
		Label:       name,
		Key:         name,
		Target:      models.FieldDefinitionTargetContact,
		Type:        models.FieldTypeString,
		Projection:  "internal-only",
		Sensitivity: models.RelationshipSensitivityNormal,
	}
	if err := db.Create(&def).Error; err != nil {
		return definitionOutcome{}, fmt.Errorf("creating field definition for user_id=%d key=%q: %w", userID, name, err)
	}

	return definitionOutcome{DefinitionID: def.ID}, nil
}

// valueOutcome summarizes what migrateContactFieldValue did (or, in dry run,
// would do) for one Contact.CustomFields[key] entry.
type valueOutcome struct {
	Skipped string
	ValueID uint
}

// migrateContactFieldValue migrates one Contact.CustomFields[key] entry
// (models/contact.go) into a models.FieldValue, per §94.6: "Each
// Contact.CustomFields[key] value -> a FieldValue{value:<string>} under the
// matching definition." Requires migrateUserFieldDefinitions to have already
// created the owning user's definition for this key (main.go runs the
// definitions pass to completion before the values pass, matching the
// natural dependency).
func migrateContactFieldValue(db *gorm.DB, contact models.Contact, key, value string, write, force bool) (valueOutcome, error) {
	var def models.FieldDefinition
	if err := db.Where("user_id = ? AND key = ?", contact.UserID, key).First(&def).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return valueOutcome{Skipped: fmt.Sprintf("no field definition found for user_id=%d key=%q (definitions pass may not have run with -write)", contact.UserID, key)}, nil
		}
		return valueOutcome{}, fmt.Errorf("loading field definition for user_id=%d key=%q: %w", contact.UserID, key, err)
	}

	var existing models.FieldValue
	err := db.Where("field_definition_id = ? AND entity_id = ?", def.ID, contact.VCardUID).First(&existing).Error
	switch {
	case err == nil:
		if !force {
			return valueOutcome{Skipped: fmt.Sprintf("already migrated to value id=%d (pass -force to reprocess)", existing.ID)}, nil
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
		// Not yet migrated -- the common case. existing stays the zero
		// value, so the Save() below performs an insert.
	default:
		return valueOutcome{}, fmt.Errorf("checking idempotency for definition=%s entity=%s: %w", def.ID, contact.VCardUID, err)
	}

	if !write {
		return valueOutcome{}, nil
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return valueOutcome{}, fmt.Errorf("marshaling value for definition=%s entity=%s: %w", def.ID, contact.VCardUID, err)
	}

	fv := existing
	fv.FieldDefinitionID = def.ID
	fv.UserID = contact.UserID
	fv.EntityID = contact.VCardUID
	fv.Value = raw

	if err := db.Save(&fv).Error; err != nil {
		return valueOutcome{}, fmt.Errorf("saving field value for definition=%s entity=%s: %w", def.ID, contact.VCardUID, err)
	}

	return valueOutcome{ValueID: fv.ID}, nil
}
