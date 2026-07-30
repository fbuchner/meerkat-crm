package services

import (
	"testing"

	"mycorrhizal/models"

	"github.com/stretchr/testify/assert"
)

// Tier 3c item 11c (docs/fork-plan/95-backlog-and-priorities.md): MergeImportedContact's
// "incoming wins when non-empty, existing survives when incoming blank" policy had only ever
// been asserted for a single scalar field (Phone, in TestParseVCF_DuplicateDetectionAndMerge).
// These tests pin the same policy down for every array field (Emails/Phones/Addresses/URLs/
// IMPPs/Circles) and for the "existing survives" direction, which was never asserted at all.

// TestMergeImportedContact_ArrayFieldsOverwriteWhenIncomingNonEmpty proves the "incoming wins"
// half of the policy holds for every multi-valued field, not just the one scalar field the
// pre-existing test happened to cover.
func TestMergeImportedContact_ArrayFieldsOverwriteWhenIncomingNonEmpty(t *testing.T) {
	existing := &models.Contact{
		Emails:    []models.ContactEmail{{Type: "old", Value: "old@example.com"}},
		Phones:    []models.ContactPhone{{Type: "old", Value: "555-0000"}},
		Addresses: []models.ContactAddress{{Type: "old", Street: "1 Old St"}},
		URLs:      []models.ContactURL{{Type: "old", Value: "https://old.example.com"}},
		IMPPs:     []models.ContactIMPP{{Type: "old", Value: "old-impp"}},
		Circles:   []string{"OldCircle"},
	}
	incoming := &models.Contact{
		Emails:    []models.ContactEmail{{Type: "new", Value: "new@example.com"}},
		Phones:    []models.ContactPhone{{Type: "new", Value: "555-1111"}},
		Addresses: []models.ContactAddress{{Type: "new", Street: "2 New Ave"}},
		URLs:      []models.ContactURL{{Type: "new", Value: "https://new.example.com"}},
		IMPPs:     []models.ContactIMPP{{Type: "new", Value: "new-impp"}},
		Circles:   []string{"NewCircle"},
	}

	MergeImportedContact(existing, incoming)

	assert.Equal(t, incoming.Emails, existing.Emails)
	assert.Equal(t, incoming.Phones, existing.Phones)
	assert.Equal(t, incoming.Addresses, existing.Addresses)
	assert.Equal(t, incoming.URLs, existing.URLs)
	assert.Equal(t, incoming.IMPPs, existing.IMPPs)
	assert.Equal(t, incoming.Circles, existing.Circles)
}

// TestMergeImportedContact_ExistingSurvivesWhenIncomingBlank proves the other, previously
// wholly-untested half of the policy: an incoming contact with no data for a field (empty
// scalar, nil/empty array) must never blank out the existing contact's value for that field.
func TestMergeImportedContact_ExistingSurvivesWhenIncomingBlank(t *testing.T) {
	existing := &models.Contact{
		Firstname: "Jane",
		Email:     "jane@example.com",
		Phone:     "555-0000",
		Emails:    []models.ContactEmail{{Type: "home", Value: "jane@example.com"}},
		Phones:    []models.ContactPhone{{Type: "home", Value: "555-0000"}},
		Addresses: []models.ContactAddress{{Type: "home", Street: "1 Existing St"}},
		URLs:      []models.ContactURL{{Type: "home", Value: "https://existing.example.com"}},
		IMPPs:     []models.ContactIMPP{{Type: "home", Value: "existing-impp"}},
		Circles:   []string{"Family"},
	}
	// A zero-value incoming Contact: every field blank/nil, as a minimal or
	// partially-parsed import row would produce.
	incoming := &models.Contact{}

	MergeImportedContact(existing, incoming)

	assert.Equal(t, "Jane", existing.Firstname)
	assert.Equal(t, "jane@example.com", existing.Email)
	assert.Equal(t, "555-0000", existing.Phone)
	assert.Equal(t, []models.ContactEmail{{Type: "home", Value: "jane@example.com"}}, existing.Emails)
	assert.Equal(t, []models.ContactPhone{{Type: "home", Value: "555-0000"}}, existing.Phones)
	assert.Equal(t, []models.ContactAddress{{Type: "home", Street: "1 Existing St"}}, existing.Addresses)
	assert.Equal(t, []models.ContactURL{{Type: "home", Value: "https://existing.example.com"}}, existing.URLs)
	assert.Equal(t, []models.ContactIMPP{{Type: "home", Value: "existing-impp"}}, existing.IMPPs)
	assert.Equal(t, []string{"Family"}, existing.Circles)
}
