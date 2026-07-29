package models

import (
	"sort"
	"strings"
)

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
		Synonyms:     []string{"mother_of", "father_of", "mom_of", "dad_of", "mother", "father", "mom", "dad", "parent"},
		VCardTypeTag: "parent",
	},
	"child_of": {
		Inverse:      "parent_of",
		Synonyms:     []string{"son_of", "daughter_of", "son", "daughter", "child"},
		VCardTypeTag: "child",
	},
	"spouse_of": {
		Inverse:      "spouse_of",
		Symmetric:    true,
		Synonyms:     []string{"married_to", "husband_of", "wife_of", "spouse", "husband", "wife", "married"},
		VCardTypeTag: "spouse",
	},
	"sibling_of": {
		Inverse:      "sibling_of",
		Symmetric:    true,
		Synonyms:     []string{"brother_of", "sister_of", "sibling", "brother", "sister"},
		VCardTypeTag: "sibling",
	},
	"friend_of": {
		Inverse:      "friend_of",
		Symmetric:    true,
		Synonyms:     []string{"friend", "friendship"},
		VCardTypeTag: "friend",
	},
	"roommate_of": {
		Inverse:      "roommate_of",
		Symmetric:    true,
		Synonyms:     []string{"roommate", "housemate"},
		VCardTypeTag: "co-resident",
	},
	// No RFC 6350 token distinguishes an unmarried romantic partner from a
	// spouse, and reusing "spouse" would misrepresent the relationship on
	// export — so this stays non-projecting like co_parent_of below, per
	// §91.2's own examples of edge types with no standard home.
	"partner_of": {
		Inverse:   "partner_of",
		Symmetric: true,
		Synonyms:  []string{"dating", "boyfriend_of", "girlfriend_of", "partner", "boyfriend", "girlfriend"},
	},
	"co_parent_of": {
		Inverse:   "co_parent_of",
		Symmetric: true,
		Synonyms:  []string{"co-parent", "coparent", "co_parent"},
	},
	"mentor_of": {
		Inverse:  "mentee_of",
		Synonyms: []string{"mentor"},
	},
	"mentee_of": {
		Inverse:  "mentor_of",
		Synonyms: []string{"mentee"},
	},
	// Fork-invented (pets, §90 D3's thin-entity graph invariant); no RFC 6350
	// equivalent.
	"owned_by": {
		Inverse:  "owns",
		Synonyms: []string{"pet"},
	},
	"owns": {
		Inverse:  "owned_by",
		Synonyms: []string{"owner"},
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

	// related_to is the deliberate fallback for a known relationship whose
	// specific nature couldn't be determined — WP-81's migration uses this
	// for legacy free-text Type values that match nothing above (e.g.
	// "Work", "Family": real values found in this codebase's own test
	// fixtures with no home in the vocabulary above). Unlike every other
	// non-projecting entry in this registry, this one IS given a vCard tag:
	// RFC 6350's own generic "known contact, nature unspecified" token,
	// rather than dropping the relationship's existence from exports
	// entirely just because its type is uncertain. Never assigned to a
	// relationship created directly through this app's own UI — the
	// relation_type validator only accepts registered tokens, and a human
	// choosing a type always has a specific one to pick.
	"related_to": {
		Inverse:      "related_to",
		Symmetric:    true,
		VCardTypeTag: "contact",
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

// IsSymmetricRelationType reports whether a registered token's reciprocal is
// itself (e.g. spouse_of), as opposed to a distinct inverse token (e.g.
// parent_of/child_of). Unregistered tokens report false. Used by WP-81's
// migration to derive RelationshipEdge.Directional from the matched type
// rather than hardcoding it, since legacy data has no separate concept of
// directionality to read it from.
func IsSymmetricRelationType(token string) bool {
	return relationTypeRegistry[token].Symmetric
}

// RelationVCardTypeTag returns the RFC 6350 §6.6.6 RELATED TYPE token a
// relation-role projects to, or "" when the type has no standard equivalent
// and must stay internal-only on export.
func RelationVCardTypeTag(token string) string {
	return relationTypeRegistry[token].VCardTypeTag
}

// MatchLegacyRelationType resolves free text (a legacy Relationship.Type
// value, or eventually a WP-86 search query term) to a registered relation
// token, via a case-insensitive match against registry keys and Synonyms,
// with simple pluralization tolerance ("Friends" -> "friend"). Returns
// ok=false if nothing matches — callers (WP-81's migration) fall back to the
// "related_to" token in that case rather than dropping the relationship.
//
// Deliberately a plain function over the registry rather than a method or a
// separate lookup table: this is the same matching problem WP-86's search
// will eventually need ("mom"/"mother" -> parent_of), so it belongs with the
// registry it reads from, not duplicated per caller.
func MatchLegacyRelationType(text string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return "", false
	}

	if token, ok := matchNormalizedRelationType(normalized); ok {
		return token, true
	}

	// Simple pluralization tolerance: only strip a trailing "s" when the
	// remainder is still a plausible word, so short unrelated tokens ("is",
	// "as") don't get mangled into false matches.
	if strings.HasSuffix(normalized, "s") && len(normalized) > 3 {
		if token, ok := matchNormalizedRelationType(strings.TrimSuffix(normalized, "s")); ok {
			return token, true
		}
	}

	return "", false
}

// matchNormalizedRelationType checks an already-lowercased, trimmed string
// against registry keys, then every entry's Synonyms.
func matchNormalizedRelationType(normalized string) (string, bool) {
	if _, ok := relationTypeRegistry[normalized]; ok {
		return normalized, true
	}
	for token, def := range relationTypeRegistry {
		for _, syn := range def.Synonyms {
			if strings.ToLower(syn) == normalized {
				return token, true
			}
		}
	}
	return "", false
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
