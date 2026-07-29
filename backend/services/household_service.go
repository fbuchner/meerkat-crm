package services

import (
	"fmt"

	"mycorrhizal/models"

	"gorm.io/gorm"
)

// memberClass is the suggestion engine's classification of a household
// member — see classifyMember. Confirmed with the user during WP-83
// planning: derived from Role + Contact.CRM.Kind only, never from
// Birthday/age (birthdays are frequently unknown, especially for the thin
// entities WP-81 promotes name-only relationships into).
type memberClass int

const (
	classAdult memberClass = iota
	classChild
	classPet
)

// classifyMember decides what role a household member plays in the
// suggestion rules below. Pet/animal is authoritative from Contact.CRM.Kind
// (WP-82 built that field for exactly this). Among humans, only an explicit
// "child" Role means child — every other Role value (including future ones
// this switch doesn't know about) defaults to adult, so the engine never
// blocks on an unrecognized or missing Role.
func classifyMember(role string, contact models.Contact) memberClass {
	if contact.CRM.Kind == "pet" || contact.CRM.Kind == "animal" {
		return classPet
	}
	if role == models.HouseholdRoleChild {
		return classChild
	}
	return classAdult
}

type classifiedMember struct {
	vcardUID string
	class    memberClass
}

// GenerateHouseholdSuggestions is the mechanism from docs/fork-plan/
// 91-envelope-data-model.md §91.4: re-scans a household's CURRENT membership
// and idempotently ensures a suggested RelationshipEdge exists for every
// applicable pair, rather than diffing what changed since a prior call —
// simpler and safe to call repeatedly (e.g. after every membership add).
//
// Every generated edge has Status: suggested, Source: household-inferred —
// §91.4 is explicit that a household's membership is never treated as a
// hard fact on its own, no matter how confidently the type implies a
// relationship. Confirming or rejecting a suggestion is a user action in a
// review surface this WP does not build (P-later, per the roadmap).
//
// Returns the edges newly created by this call (not ones that already
// existed and were skipped).
func GenerateHouseholdSuggestions(db *gorm.DB, household models.Household) ([]models.RelationshipEdge, error) {
	var members []models.HouseholdMember
	if err := db.Where("household_id = ?", household.ID).Find(&members).Error; err != nil {
		return nil, fmt.Errorf("loading members for household id=%s: %w", household.ID, err)
	}
	if len(members) < 2 {
		return nil, nil
	}

	classified := make([]classifiedMember, 0, len(members))
	for _, m := range members {
		var contact models.Contact
		if err := db.Where("vcard_uid = ?", m.MemberVCardUID).First(&contact).Error; err != nil {
			return nil, fmt.Errorf("loading contact vcard_uid=%s for household id=%s: %w", m.MemberVCardUID, household.ID, err)
		}
		classified = append(classified, classifiedMember{vcardUID: m.MemberVCardUID, class: classifyMember(m.Role, contact)})
	}

	var created []models.RelationshipEdge
	suggest := func(sourceID, targetID, edgeType string, confidence float64) error {
		edge, err := suggestEdgeIfNew(db, household.UserID, sourceID, targetID, edgeType, confidence)
		if err != nil {
			return err
		}
		if edge != nil {
			created = append(created, *edge)
		}
		return nil
	}

	switch household.Type {
	case models.HouseholdTypeFamilyUnit:
		// §91.4: adult<->adult spouse_of; adult->child parent_of; every
		// HUMAN (adult or child, not just adult) -> pet owned_by.
		const familyConfidence = 0.8
		for i := 0; i < len(classified); i++ {
			for j := i + 1; j < len(classified); j++ {
				a, b := classified[i], classified[j]
				switch {
				case a.class == classAdult && b.class == classAdult:
					if err := suggest(a.vcardUID, b.vcardUID, "spouse_of", familyConfidence); err != nil {
						return created, err
					}
				case a.class == classAdult && b.class == classChild:
					if err := suggest(a.vcardUID, b.vcardUID, "parent_of", familyConfidence); err != nil {
						return created, err
					}
				case b.class == classAdult && a.class == classChild:
					if err := suggest(b.vcardUID, a.vcardUID, "parent_of", familyConfidence); err != nil {
						return created, err
					}
				case a.class != classPet && b.class == classPet:
					// "A owned_by B" reads "A is owned by B" — the pet is
					// the source, matching the type registry's own
					// convention (models/relationship_type_registry.go).
					if err := suggest(b.vcardUID, a.vcardUID, "owned_by", familyConfidence); err != nil {
						return created, err
					}
				case b.class != classPet && a.class == classPet:
					if err := suggest(a.vcardUID, b.vcardUID, "owned_by", familyConfidence); err != nil {
						return created, err
					}
				// child<->child and pet<->pet: no rule in §91.4; skipped.
				default:
				}
			}
		}

	case models.HouseholdTypeRoommates:
		// §91.4: member<->member roommate_of only — explicitly never
		// parent/owner/spouse, regardless of role or kind.
		const roommateConfidence = 0.4
		for i := 0; i < len(classified); i++ {
			for j := i + 1; j < len(classified); j++ {
				if err := suggest(classified[i].vcardUID, classified[j].vcardUID, "roommate_of", roommateConfidence); err != nil {
					return created, err
				}
			}
		}

	default:
		// "other", and any type value this switch doesn't recognize: no
		// structural inference (§91.4's own table says exactly this for
		// "other" — unrecognized values get the same treatment, not an
		// error, matching how the rest of this WP degrades on open enums).
	}

	return created, nil
}

// suggestEdgeIfNew creates a suggested RelationshipEdge for (sourceID,
// targetID, edgeType) unless an edge for that relationship already exists —
// checked in EITHER storage direction, so a relationship already recorded
// as (target, source, InverseRelationType(edgeType)) is recognized as the
// same fact and not duplicated. Covers both the symmetric case (a type's
// inverse is itself) and the directional case (the reciprocal token)
// uniformly. Matches regardless of the existing edge's status — a
// `confirmed` edge is never re-suggested over, any more than a `suggested`
// one is duplicated.
//
// Returns the created edge, or nil if one already existed.
func suggestEdgeIfNew(db *gorm.DB, userID uint, sourceID, targetID, edgeType string, confidence float64) (*models.RelationshipEdge, error) {
	inverse := models.InverseRelationType(edgeType)

	var count int64
	err := db.Model(&models.RelationshipEdge{}).Where(
		"(source_id = ? AND target_id = ? AND type = ?) OR (source_id = ? AND target_id = ? AND type = ?)",
		sourceID, targetID, edgeType,
		targetID, sourceID, inverse,
	).Count(&count).Error
	if err != nil {
		return nil, fmt.Errorf("checking for an existing %s edge %s->%s: %w", edgeType, sourceID, targetID, err)
	}
	if count > 0 {
		return nil, nil
	}

	edge := models.RelationshipEdge{
		UserID:      userID,
		SourceID:    sourceID,
		TargetID:    targetID,
		Type:        edgeType,
		Directional: !models.IsSymmetricRelationType(edgeType),
		Source:      models.RelationshipSourceHouseholdInferred,
		Confidence:  confidence,
		Status:      models.RelationshipStatusSuggested,
		Sensitivity: models.RelationshipSensitivityNormal,
	}
	if err := db.Create(&edge).Error; err != nil {
		return nil, fmt.Errorf("creating suggested %s edge %s->%s: %w", edgeType, sourceID, targetID, err)
	}
	return &edge, nil
}
