package models

import "sort"

// relationTypeDef describes one canonical relation-role token: its reciprocal
// token, whether that reciprocal is itself (symmetric), search synonyms
// (unused until WP-86's traversal/search work), and the RFC 6350 §6.6.6
// RELATED TYPE token it projects to on export — empty when the relation has
// no standard equivalent and must stay internal (docs/fork-plan/91-envelope-
// data-model.md §91.2's "deliberately lossy" export rule).
type relationTypeDef struct {
	Inverse      string
	Symmetric    bool
	Synonyms     []string
	VCardTypeTag string
}

// relationTypeRegistry is the single source of truth for every known
// relation-role token: what CreateRelationshipEdge/UpdateRelationshipEdge
// validate against (via the relation_type validator tag, middleware/
// validation.go), and what RecordForContact's projection step (models/
// contact_record.go) reads to decide whether and how an edge appears in
// Card.RelatedTo. Extend this map, not a second list, when adding a token.
//
// parent_of/child_of is the only asymmetric pair with two distinct vCard
// tags; every other row is symmetric so Inverse just points back at itself.
var relationTypeRegistry = map[string]relationTypeDef{
	"parent_of": {
		Inverse:      "child_of",
		Synonyms:     []string{"mother_of", "father_of", "mom_of", "dad_of"},
		VCardTypeTag: "parent",
	},
	"child_of": {
		Inverse:      "parent_of",
		Synonyms:     []string{"son_of", "daughter_of"},
		VCardTypeTag: "child",
	},
	"spouse_of": {
		Inverse:      "spouse_of",
		Symmetric:    true,
		Synonyms:     []string{"married_to", "husband_of", "wife_of"},
		VCardTypeTag: "spouse",
	},
	"sibling_of": {
		Inverse:      "sibling_of",
		Symmetric:    true,
		Synonyms:     []string{"brother_of", "sister_of"},
		VCardTypeTag: "sibling",
	},
	"friend_of": {
		Inverse:      "friend_of",
		Symmetric:    true,
		VCardTypeTag: "friend",
	},
	"roommate_of": {
		Inverse:      "roommate_of",
		Symmetric:    true,
		VCardTypeTag: "co-resident",
	},
	// No RFC 6350 token distinguishes an unmarried romantic partner from a
	// spouse, and reusing "spouse" would misrepresent the relationship on
	// export — so this stays non-projecting like co_parent_of below, per
	// §91.2's own examples of edge types with no standard home.
	"partner_of": {
		Inverse:   "partner_of",
		Symmetric: true,
		Synonyms:  []string{"dating", "boyfriend_of", "girlfriend_of"},
	},
	"co_parent_of": {
		Inverse:   "co_parent_of",
		Symmetric: true,
	},
	"mentor_of": {
		Inverse: "mentee_of",
	},
	"mentee_of": {
		Inverse: "mentor_of",
	},
	// Fork-invented (pets, §90 D3's thin-entity graph invariant); no RFC 6350
	// equivalent.
	"owned_by": {
		Inverse: "owns",
	},
	"owns": {
		Inverse: "owned_by",
	},
	// Affinity edges (§91.2 "Affinity edges" subsection) — pairwise
	// compatibility, not a structural bond. Always non-projecting:
	// gets_along_with has no vCard equivalent, and conflicts_with must never
	// reach an export regardless (its sensitivity defaults to private/secret
	// at the call site that creates it, per §91.13 — the registry doesn't
	// enforce that, the creation path does).
	"gets_along_with": {
		Inverse:   "gets_along_with",
		Symmetric: true,
	},
	"conflicts_with": {
		Inverse:   "conflicts_with",
		Symmetric: true,
	},
}

// IsKnownRelationType reports whether token is a registered relation-role.
// Backs the relation_type validator tag (middleware/validation.go).
func IsKnownRelationType(token string) bool {
	_, ok := relationTypeRegistry[token]
	return ok
}

// InverseRelationType returns the reciprocal token for a known relation type,
// or "" if token is unregistered. Never stored (§91.2: "store one edge,
// derive the inverse") — always derived through this function.
func InverseRelationType(token string) string {
	return relationTypeRegistry[token].Inverse
}

// RelationVCardTypeTag returns the RFC 6350 §6.6.6 RELATED TYPE token a
// relation-role projects to, or "" when the type has no standard equivalent
// and must stay internal-only on export.
func RelationVCardTypeTag(token string) string {
	return relationTypeRegistry[token].VCardTypeTag
}

// KnownRelationTypes returns every registered token, sorted, for tests and
// any future admin/debug surface. Not used by the validator itself (which
// only needs membership, not the full list).
func KnownRelationTypes() []string {
	tokens := make([]string, 0, len(relationTypeRegistry))
	for token := range relationTypeRegistry {
		tokens = append(tokens, token)
	}
	sort.Strings(tokens)
	return tokens
}
