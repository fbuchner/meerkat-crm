package models

import (
	"testing"

	"mycorrhizal/contactmodel"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// --- Type registry unit tests -----------------------------------------------

func TestIsKnownRelationType(t *testing.T) {
	assert.True(t, IsKnownRelationType("parent_of"))
	assert.True(t, IsKnownRelationType("conflicts_with"))
	assert.False(t, IsKnownRelationType("not_a_real_type"))
	assert.False(t, IsKnownRelationType(""))
}

func TestInverseRelationType(t *testing.T) {
	tests := []struct {
		token string
		want  string
	}{
		{"parent_of", "child_of"},
		{"child_of", "parent_of"},
		{"spouse_of", "spouse_of"},   // symmetric: inverse is itself
		{"sibling_of", "sibling_of"}, // symmetric
		{"mentor_of", "mentee_of"},
		{"mentee_of", "mentor_of"},
		{"owned_by", "owns"},
		{"owns", "owned_by"},
		{"unknown_token", ""}, // unregistered: no inverse
	}
	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			assert.Equal(t, tt.want, InverseRelationType(tt.token))
		})
	}
}

func TestRelationVCardTypeTag(t *testing.T) {
	tests := []struct {
		token string
		want  string
	}{
		{"parent_of", "parent"},
		{"child_of", "child"},
		{"spouse_of", "spouse"},
		{"roommate_of", "co-resident"},
		// Non-projecting: no RFC 6350 equivalent, must stay internal-only.
		{"co_parent_of", ""},
		{"partner_of", ""},
		{"mentor_of", ""},
		{"owned_by", ""},
		{"gets_along_with", ""},
		{"conflicts_with", ""},
		{"unknown_token", ""},
	}
	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			assert.Equal(t, tt.want, RelationVCardTypeTag(tt.token))
		})
	}
}

// Every inverse pointer must resolve to another entry in the same registry —
// a dangling inverse would silently break projection for one direction of a
// pair without any test noticing until it hit real data.
func TestRelationTypeRegistryInversesAreConsistent(t *testing.T) {
	for _, token := range KnownRelationTypes() {
		inverse := InverseRelationType(token)
		assert.NotEmpty(t, inverse, "token %q has no inverse", token)
		assert.True(t, IsKnownRelationType(inverse), "token %q's inverse %q is not itself registered", token, inverse)
	}
}

// MatchLegacyRelationType is what WP-81's migration uses to resolve legacy
// free-text Relationship.Type values to a registered token.
func TestMatchLegacyRelationType(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{"registry key verbatim", "parent_of", "parent_of"},
		{"case-insensitive plain noun", "Sibling", "sibling_of"},
		{"the real test-fixture value Friendship", "Friendship", "friend_of"},
		{"plain noun with different casing", "MOTHER", "parent_of"},
		{"simple plural tolerance", "Friends", "friend_of"},
		{"compound synonym", "married_to", "spouse_of"},
		{"leading/trailing whitespace", "  friend  ", "friend_of"},
		{"pet ownership synonym", "Owner", "owns"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, ok := MatchLegacyRelationType(tt.text)
			require.True(t, ok, "expected a match for %q", tt.text)
			assert.Equal(t, tt.want, token)
		})
	}
}

// The two real unmapped legacy values found in this codebase's own test
// fixtures (relationship_controller_test.go) — confirms the fallback path
// this migration relies on is real, not hypothetical.
func TestMatchLegacyRelationTypeNoMatch(t *testing.T) {
	for _, text := range []string{"Work", "Family", "", "   ", "asdfghjkl"} {
		t.Run(text, func(t *testing.T) {
			_, ok := MatchLegacyRelationType(text)
			assert.False(t, ok, "expected no match for %q", text)
		})
	}
}

func TestRelatedToFallbackToken(t *testing.T) {
	assert.True(t, IsKnownRelationType("related_to"))
	assert.Equal(t, "related_to", InverseRelationType("related_to"))
	assert.Equal(t, "contact", RelationVCardTypeTag("related_to"),
		"related_to must still project something on export, unlike every other non-projecting fallback")
}

// --- Projection integration tests -------------------------------------------

func setupRelationshipTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&User{}, &Contact{}, &RelationshipEdge{}))
	return db
}

func createRelationshipTestUser(t *testing.T, db *gorm.DB) User {
	t.Helper()
	user := User{Username: "tester", Password: "password123!A", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)
	return user
}

func createRelationshipTestContact(t *testing.T, db *gorm.DB, userID uint, firstname string) Contact {
	t.Helper()
	contact := Contact{UserID: userID, Firstname: firstname}
	require.NoError(t, db.Create(&contact).Error)
	require.NotEmpty(t, contact.VCardUID, "BeforeCreate must generate a VCardUID")
	return contact
}

// The P5 acceptance criterion, taken literally: a confirmed edge between two
// real contacts round-trips to Card.RelatedTo. Also verifies the direction
// logic worked through in the design (parent_of A->B => "parent" on B's
// card, "child" on A's card) rather than just asserting *something* appears.
func TestRecordForContact_ProjectsConfirmedEdgeOntoBothSides(t *testing.T) {
	db := setupRelationshipTestDB(t)
	user := createRelationshipTestUser(t, db)
	alice := createRelationshipTestContact(t, db, user.ID, "Alice")
	bob := createRelationshipTestContact(t, db, user.ID, "Bob")

	edge := RelationshipEdge{
		UserID:      user.ID,
		SourceID:    alice.VCardUID,
		TargetID:    bob.VCardUID,
		Type:        "parent_of",
		Directional: true,
		Source:      RelationshipSourceUserConfirmed,
		Confidence:  1.0,
		Status:      RelationshipStatusConfirmed,
		Sensitivity: RelationshipSensitivityNormal,
	}
	require.NoError(t, db.Create(&edge).Error)

	bobRecord := RecordForContact(&bob, "", db)
	require.Len(t, bobRecord.Card.RelatedTo, 1, "Bob's card should describe Alice as his parent")
	assert.Equal(t, "urn:uuid:"+alice.VCardUID, bobRecord.Card.RelatedTo[0].Target)
	assert.Equal(t, []string{"parent"}, bobRecord.Card.RelatedTo[0].Relations)

	aliceRecord := RecordForContact(&alice, "", db)
	require.Len(t, aliceRecord.Card.RelatedTo, 1, "Alice's card should describe Bob as her child")
	assert.Equal(t, "urn:uuid:"+bob.VCardUID, aliceRecord.Card.RelatedTo[0].Target)
	assert.Equal(t, []string{"child"}, aliceRecord.Card.RelatedTo[0].Relations)
}

// Symmetric types (registry Inverse == the type itself) must still emit the
// same tag on both sides, not require special-casing distinct from the
// asymmetric parent_of/child_of case above.
func TestRecordForContact_ProjectsSymmetricEdge(t *testing.T) {
	db := setupRelationshipTestDB(t)
	user := createRelationshipTestUser(t, db)
	alice := createRelationshipTestContact(t, db, user.ID, "Alice")
	bob := createRelationshipTestContact(t, db, user.ID, "Bob")

	edge := RelationshipEdge{
		UserID:      user.ID,
		SourceID:    alice.VCardUID,
		TargetID:    bob.VCardUID,
		Type:        "spouse_of",
		Source:      RelationshipSourceUserConfirmed,
		Confidence:  1.0,
		Status:      RelationshipStatusConfirmed,
		Sensitivity: RelationshipSensitivityNormal,
	}
	require.NoError(t, db.Create(&edge).Error)

	for _, tc := range []struct {
		name string
		self Contact
		want string
	}{
		{"alice's card", alice, "spouse"},
		{"bob's card", bob, "spouse"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := RecordForContact(&tc.self, "", db)
			require.Len(t, record.Card.RelatedTo, 1)
			assert.Equal(t, []string{tc.want}, record.Card.RelatedTo[0].Relations)
		})
	}
}

// §91.2: "only confirmed edges are authoritative" — a suggested edge (e.g.
// household-inferred, awaiting user review) must never appear in an export.
func TestRecordForContact_DoesNotProjectSuggestedEdge(t *testing.T) {
	db := setupRelationshipTestDB(t)
	user := createRelationshipTestUser(t, db)
	alice := createRelationshipTestContact(t, db, user.ID, "Alice")
	bob := createRelationshipTestContact(t, db, user.ID, "Bob")

	edge := RelationshipEdge{
		UserID:      user.ID,
		SourceID:    alice.VCardUID,
		TargetID:    bob.VCardUID,
		Type:        "spouse_of",
		Source:      RelationshipSourceHouseholdInferred,
		Confidence:  0.8,
		Status:      RelationshipStatusSuggested,
		Sensitivity: RelationshipSensitivityNormal,
	}
	require.NoError(t, db.Create(&edge).Error)

	record := RecordForContact(&bob, "", db)
	assert.Empty(t, record.Card.RelatedTo)
}

// §91.13: anything above "normal" sensitivity is excluded by default from
// exports. Private and secret are both tested since the gate is a single
// equality check that could accidentally only cover one of them.
func TestRecordForContact_DoesNotProjectAboveNormalSensitivity(t *testing.T) {
	for _, sensitivity := range []string{RelationshipSensitivityPrivate, RelationshipSensitivitySecret} {
		t.Run(sensitivity, func(t *testing.T) {
			db := setupRelationshipTestDB(t)
			user := createRelationshipTestUser(t, db)
			alice := createRelationshipTestContact(t, db, user.ID, "Alice")
			bob := createRelationshipTestContact(t, db, user.ID, "Bob")

			edge := RelationshipEdge{
				UserID:      user.ID,
				SourceID:    alice.VCardUID,
				TargetID:    bob.VCardUID,
				Type:        "partner_of",
				Source:      RelationshipSourceUserConfirmed,
				Confidence:  1.0,
				Status:      RelationshipStatusConfirmed,
				Sensitivity: sensitivity,
			}
			require.NoError(t, db.Create(&edge).Error)

			record := RecordForContact(&bob, "", db)
			assert.Empty(t, record.Card.RelatedTo)
		})
	}
}

// co_parent_of has no RFC 6350 equivalent (registry.VCardTypeTag == "") —
// the spec's own example of a type that must stay internal even when
// confirmed and normal-sensitivity.
func TestRecordForContact_DoesNotProjectNonStandardType(t *testing.T) {
	db := setupRelationshipTestDB(t)
	user := createRelationshipTestUser(t, db)
	alice := createRelationshipTestContact(t, db, user.ID, "Alice")
	bob := createRelationshipTestContact(t, db, user.ID, "Bob")

	edge := RelationshipEdge{
		UserID:      user.ID,
		SourceID:    alice.VCardUID,
		TargetID:    bob.VCardUID,
		Type:        "co_parent_of",
		Source:      RelationshipSourceUserConfirmed,
		Confidence:  1.0,
		Status:      RelationshipStatusConfirmed,
		Sensitivity: RelationshipSensitivityNormal,
	}
	require.NoError(t, db.Create(&edge).Error)

	record := RecordForContact(&bob, "", db)
	assert.Empty(t, record.Card.RelatedTo)
}

// A RELATED entry that arrived via external vCard/JSContact import (raw
// passthrough, stored wholesale in Contact.Card — see contact_record.go's
// doc comments) must survive alongside a graph-derived entry, and a
// graph-derived entry that happens to duplicate an existing passthrough fact
// must not appear twice.
func TestRecordForContact_MergesWithPassthroughWithoutDuplication(t *testing.T) {
	db := setupRelationshipTestDB(t)
	user := createRelationshipTestUser(t, db)
	alice := createRelationshipTestContact(t, db, user.ID, "Alice")
	bob := createRelationshipTestContact(t, db, user.ID, "Bob")

	// An external, non-graph passthrough fact unrelated to Alice/Bob, applied
	// via ApplyRecordToContact (not a direct field mutation + Save) so it
	// survives BeforeSave exactly as a real vCard/JSContact import would —
	// see Contact.cardSetDirectly's doc comment for why a bare field
	// assignment wouldn't stick.
	passthroughRecord := RecordForContact(&bob, "", nil)
	passthroughRecord.Card.RelatedTo = []contactmodel.Relation{
		{Target: "urn:uuid:some-external-uid", Relations: []string{"friend"}},
	}
	ApplyRecordToContact(&bob, passthroughRecord, "")
	require.NoError(t, db.Save(&bob).Error)

	edge := RelationshipEdge{
		UserID:      user.ID,
		SourceID:    alice.VCardUID,
		TargetID:    bob.VCardUID,
		Type:        "parent_of",
		Directional: true,
		Source:      RelationshipSourceUserConfirmed,
		Confidence:  1.0,
		Status:      RelationshipStatusConfirmed,
		Sensitivity: RelationshipSensitivityNormal,
	}
	require.NoError(t, db.Create(&edge).Error)

	record := RecordForContact(&bob, "", db)
	require.Len(t, record.Card.RelatedTo, 2, "the passthrough entry and the graph-derived entry must coexist")
	assert.Contains(t, record.Card.RelatedTo, contactmodel.Relation{
		Target: "urn:uuid:some-external-uid", Relations: []string{"friend"},
	})
	assert.Contains(t, record.Card.RelatedTo, contactmodel.Relation{
		Target: "urn:uuid:" + alice.VCardUID, Relations: []string{"parent"},
	})

	// The stored row itself must be untouched by the read above — projection
	// is never persisted back (D3: the graph is the source of truth).
	var reloaded Contact
	require.NoError(t, db.First(&reloaded, bob.ID).Error)
	assert.Len(t, reloaded.Card.RelatedTo, 1, "persisted Card.RelatedTo must still only hold the original passthrough entry")

	// Calling RecordForContact again must not accumulate duplicates.
	again := RecordForContact(&bob, "", db)
	assert.Len(t, again.Card.RelatedTo, 2)
}

// db == nil is the documented "skip projection" contract for callers that
// don't have a *gorm.DB handy — must return existing passthrough data
// untouched, not panic or silently drop it.
func TestRecordForContact_NilDBSkipsProjection(t *testing.T) {
	c := &Contact{
		Firstname: "Standalone",
		Card: contactmodel.Card{
			RelatedTo: []contactmodel.Relation{{Target: "urn:uuid:x", Relations: []string{"friend"}}},
		},
	}

	record := RecordForContact(c, "", nil)
	assert.Equal(t, c.Card.RelatedTo, record.Card.RelatedTo)
}
