package services

import (
	"testing"

	"mycorrhizal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeContactMergeResolution_MultiValuedUnion(t *testing.T) {
	keeper := &models.Contact{Emails: []models.ContactEmail{{Type: "home", Value: "alice@home.example"}}}
	loser := &models.Contact{Emails: []models.ContactEmail{{Type: "work", Value: "alice@work.example"}}}

	res := ComputeContactMergeResolution(keeper, loser)

	assert.Len(t, res.Emails, 2, "multi-valued fields must UNION, not overwrite -- this is the ticket's core regression")
	values := []string{res.Emails[0].Value, res.Emails[1].Value}
	assert.Contains(t, values, "alice@home.example")
	assert.Contains(t, values, "alice@work.example")
}

func TestComputeContactMergeResolution_EmailDedupCaseInsensitive(t *testing.T) {
	keeper := &models.Contact{Emails: []models.ContactEmail{{Type: "home", Value: "Alice@Example.com"}}}
	loser := &models.Contact{Emails: []models.ContactEmail{{Type: "home", Value: "alice@example.com"}}}

	res := ComputeContactMergeResolution(keeper, loser)

	require.Len(t, res.Emails, 1, "same (type, value) case-insensitively must dedup to one entry")
	assert.Equal(t, "Alice@Example.com", res.Emails[0].Value, "keeper's casing should win the tie")
}

func TestComputeContactMergeResolution_PhoneDedupByNormalizedDigits(t *testing.T) {
	// normalizePhoneForComparison strips non-digit punctuation but does not
	// reconcile an explicit country code against its absence -- these two
	// forms carry the same digit sequence and formatting only, matching
	// exactly what normalizePhoneForComparison already treats as equal for
	// DetectDuplicate's own phone-match path.
	keeper := &models.Contact{Phones: []models.ContactPhone{{Type: "mobile", Value: "(555) 123-4567"}}}
	loser := &models.Contact{Phones: []models.ContactPhone{{Type: "mobile", Value: "5551234567"}}}

	res := ComputeContactMergeResolution(keeper, loser)

	assert.Len(t, res.Phones, 1, "phones equal after normalizePhoneForComparison must dedup")
}

func TestComputeContactMergeResolution_ScalarOnlyOneSideSet(t *testing.T) {
	keeper := &models.Contact{Firstname: "Alice"}
	loser := &models.Contact{Lastname: "Smith"}

	res := ComputeContactMergeResolution(keeper, loser)

	assert.Equal(t, "Alice", res.ResolvedScalars["firstname"])
	assert.Equal(t, "Smith", res.ResolvedScalars["lastname"])
	assert.Empty(t, res.Conflicts)
}

func TestComputeContactMergeResolution_IdenticalScalarNotAConflict(t *testing.T) {
	keeper := &models.Contact{Firstname: "Alice"}
	loser := &models.Contact{Firstname: "Alice"}

	res := ComputeContactMergeResolution(keeper, loser)

	assert.Equal(t, "Alice", res.ResolvedScalars["firstname"])
	assert.Empty(t, res.Conflicts)
}

func TestComputeContactMergeResolution_DifferingScalarIsAConflict(t *testing.T) {
	keeper := &models.Contact{Firstname: "Alice"}
	loser := &models.Contact{Firstname: "Robert"}

	res := ComputeContactMergeResolution(keeper, loser)

	require.Len(t, res.Conflicts, 1)
	assert.Equal(t, "firstname", res.Conflicts[0].Field)
	assert.Equal(t, "Alice", res.Conflicts[0].KeeperValue)
	assert.Equal(t, "Robert", res.Conflicts[0].LoserValue)
	_, resolved := res.ResolvedScalars["firstname"]
	assert.False(t, resolved, "a conflicting field must not also appear as auto-resolved")
}

func TestComputeContactMergeResolution_CirclesUnionCaseInsensitive(t *testing.T) {
	keeper := &models.Contact{Circles: []string{"Friends"}}
	loser := &models.Contact{Circles: []string{"friends", "Work"}}

	res := ComputeContactMergeResolution(keeper, loser)

	assert.Equal(t, []string{"Friends", "Work"}, res.Circles)
}

func TestComputeContactMergeResolution_CustomFieldsUnionKeeperWinsCollision(t *testing.T) {
	keeper := &models.Contact{CustomFields: map[string]string{"shared": "keeper-value", "keeper-only": "k"}}
	loser := &models.Contact{CustomFields: map[string]string{"shared": "loser-value", "loser-only": "l"}}

	res := ComputeContactMergeResolution(keeper, loser)

	assert.Equal(t, "keeper-value", res.CustomFields["shared"])
	assert.Equal(t, "k", res.CustomFields["keeper-only"])
	assert.Equal(t, "l", res.CustomFields["loser-only"])
}

func TestApplyContactMergeResolution_MissingResolutionErrors(t *testing.T) {
	keeper := &models.Contact{Firstname: "Alice"}
	loser := &models.Contact{Firstname: "Robert"}
	res := ComputeContactMergeResolution(keeper, loser)

	err := ApplyContactMergeResolution(keeper, res, map[string]string{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "firstname")
}

func TestApplyContactMergeResolution_AppliesChosenValueAndUnions(t *testing.T) {
	keeper := &models.Contact{
		Firstname: "Alice",
		Emails:    []models.ContactEmail{{Type: "home", Value: "alice@home.example"}},
	}
	loser := &models.Contact{
		Firstname: "Robert",
		Emails:    []models.ContactEmail{{Type: "work", Value: "bob@work.example"}},
	}
	res := ComputeContactMergeResolution(keeper, loser)

	err := ApplyContactMergeResolution(keeper, res, map[string]string{"firstname": "Alice"})

	require.NoError(t, err)
	assert.Equal(t, "Alice", keeper.Firstname)
	assert.Len(t, keeper.Emails, 2)
}

func TestUnionAddresses_DedupsOnFullNormalizedTuple(t *testing.T) {
	a := []models.ContactAddress{{Type: "home", Street: "123 Main St", City: "Springfield"}}
	b := []models.ContactAddress{{Type: "HOME", Street: "123 main st", City: "springfield"}}

	out := unionAddresses(a, b)

	assert.Len(t, out, 1, "same address differing only in case must dedup")
}

func TestUnionAddresses_DistinctAddressesBothSurvive(t *testing.T) {
	a := []models.ContactAddress{{Type: "home", Street: "123 Main St", City: "Springfield"}}
	b := []models.ContactAddress{{Type: "work", Street: "456 Oak Ave", City: "Shelbyville"}}

	out := unionAddresses(a, b)

	assert.Len(t, out, 2)
}
