package services

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"mycorrhizal/contactmodel"
	"mycorrhizal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TC-1.4 (docs/fork-plan/45-test-coverage-closure.md): gap-fill tests for the
// previously 0%-or-low functions in import_service.go: ParseCSV,
// GenerateCSVPreview, GetStringField, NormalizeBirthday, NormalizeGender,
// CreateMergeNote, plus branch closure on diagnosticsToStrings and
// extractPhotoFromRecord.

// errReader is an io.Reader that yields some valid bytes and then fails,
// simulating a network/file read error mid-stream (rather than a clean EOF).
type errReader struct {
	data []byte
	pos  int
	err  error
}

func (r *errReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, r.err
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// --- ParseCSV -----------------------------------------------------------

func TestParseCSV_HappyPath(t *testing.T) {
	reader := strings.NewReader("First Name,Email\nAda,ada@example.com\nBob,bob@example.com\n")
	headers, rows, err := ParseCSV(reader)
	require.NoError(t, err)
	assert.Equal(t, []string{"First Name", "Email"}, headers)
	require.Len(t, rows, 2)
	assert.Equal(t, []string{"Ada", "ada@example.com"}, rows[0])
	assert.Equal(t, []string{"Bob", "bob@example.com"}, rows[1])
}

func TestParseCSV_EmptyFile(t *testing.T) {
	_, _, err := ParseCSV(strings.NewReader(""))
	assert.Error(t, err)
}

func TestParseCSV_HeaderOnlyNoDataRows(t *testing.T) {
	_, _, err := ParseCSV(strings.NewReader("First Name,Email\n"))
	assert.Error(t, err)
}

// TestParseCSV_RaggedRows confirms ParseCSV is lenient about inconsistent
// column counts (FieldsPerRecord = -1 in the implementation) rather than
// hard-failing, per the degradation policy (00-overview.md §0.5) applied
// generally across this fork's import paths.
func TestParseCSV_RaggedRows(t *testing.T) {
	reader := strings.NewReader("A,B,C\n1,2\n1,2,3,4\n")
	headers, rows, err := ParseCSV(reader)
	require.NoError(t, err)
	assert.Equal(t, []string{"A", "B", "C"}, headers)
	require.Len(t, rows, 2)
	assert.Equal(t, []string{"1", "2"}, rows[0])
	assert.Equal(t, []string{"1", "2", "3", "4"}, rows[1])
}

func TestParseCSV_ReaderErrorMidStream(t *testing.T) {
	r := &errReader{data: []byte("A,B\n1,2\n"), err: errors.New("simulated read failure")}
	_, _, err := ParseCSV(r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid CSV format")
}

// --- GenerateCSVPreview ---------------------------------------------------

func TestGenerateCSVPreview(t *testing.T) {
	db, _ := setupRouter()
	userID := uint(1)

	existing := models.Contact{
		UserID: userID, Firstname: "John", Lastname: "Doe", Email: "john@example.com",
	}
	require.NoError(t, db.Create(&existing).Error)

	headers := []string{"First Name", "Last Name", "Email", "Some Random Column"}
	mappings := SuggestColumnMappings(headers)
	// "Some Random Column" must be left unmapped by the header guesser.
	require.Equal(t, "", findMapping(t, mappings, "Some Random Column").ContactField)

	rows := [][]string{
		{"John", "Doe", "john@example.com", "ignored value"}, // duplicate (email match)
		{"Jane", "Roe", "jane@example.com", "ignored"},       // new, valid contact
		{"", "NoFirstName", "bad@x.com", "x"},                // invalid: missing firstname
	}

	contacts, previews, stats := GenerateCSVPreview(db, userID, rows, headers, mappings)

	require.Len(t, contacts, 3)
	require.Len(t, previews, 3)

	assert.Equal(t, 2, stats.ValidCount)
	assert.Equal(t, 1, stats.DuplicateCount)
	assert.Equal(t, 1, stats.ErrorCount)

	// Row 0: mapped columns produce correct model.Contact-shaped preview
	// fields, and duplicate detection is wired through to DuplicateMatch/stats.
	assert.Equal(t, "update", previews[0].SuggestedAction)
	require.NotNil(t, previews[0].DuplicateMatch)
	assert.Equal(t, existing.ID, previews[0].DuplicateMatch.ExistingContactID)
	assert.Equal(t, "email", previews[0].DuplicateMatch.MatchReason)
	assert.Equal(t, "John", previews[0].ParsedContact["firstname"])
	assert.Equal(t, "Doe", previews[0].ParsedContact["lastname"])
	assert.Equal(t, "john@example.com", previews[0].ParsedContact["email"])
	// The unmapped column must not leak into the parsed-contact preview map.
	assert.NotContains(t, previews[0].ParsedContact, "some_random_column")
	assert.NotContains(t, previews[0].ParsedContact, "Some Random Column")

	// Row 1: valid, no duplicate.
	assert.Equal(t, "add", previews[1].SuggestedAction)
	assert.Nil(t, previews[1].DuplicateMatch)
	assert.Equal(t, "Jane", previews[1].ParsedContact["firstname"])

	// Row 2: invalid -> skip, surfaced validation error.
	assert.Equal(t, "skip", previews[2].SuggestedAction)
	assert.Contains(t, previews[2].ValidationErrors, "First name is required")
}

// --- GetStringField --------------------------------------------------------

func TestGetStringField(t *testing.T) {
	parsed := map[string]interface{}{
		"firstname": "Ada",
		"age":       30,            // non-string type
		"score":     3.14,          // non-string type
		"active":    true,          // non-string type
		"tags":      []string{"a"}, // non-string type
	}

	assert.Equal(t, "Ada", GetStringField(parsed, "firstname"))

	// Non-string types must not panic; graceful empty-string fallback per the
	// degradation policy (00-overview.md §0.5).
	assert.NotPanics(t, func() {
		assert.Equal(t, "", GetStringField(parsed, "age"))
	})
	assert.Equal(t, "", GetStringField(parsed, "score"))
	assert.Equal(t, "", GetStringField(parsed, "active"))
	assert.Equal(t, "", GetStringField(parsed, "tags"))

	// Key absent.
	assert.Equal(t, "", GetStringField(parsed, "missing_key"))

	// Nil map must not panic either.
	assert.NotPanics(t, func() {
		assert.Equal(t, "", GetStringField(nil, "firstname"))
	})
}

// --- NormalizeBirthday -------------------------------------------------

func TestNormalizeBirthday_FullISODate(t *testing.T) {
	assert.Equal(t, "1958-06-29", NormalizeBirthday("1958-06-29"))
}

func TestNormalizeBirthday_YearUnknownForm(t *testing.T) {
	assert.Equal(t, "--04-20", NormalizeBirthday("--04-20"))
}

func TestNormalizeBirthday_LegacyDotFormatWithYear(t *testing.T) {
	assert.Equal(t, "1958-06-29", NormalizeBirthday("29.06.1958"))
}

func TestNormalizeBirthday_LegacyDotFormatNoYear(t *testing.T) {
	assert.Equal(t, "--06-29", NormalizeBirthday("29.06."))
}

// TestNormalizeBirthday_Garbage documents the function's actual, literal
// contract on unparseable input: reading the implementation shows it falls
// through every recognized-format branch and returns the input unchanged
// (comment: "Unknown format - return as-is (will fail validation)") rather
// than erroring or blanking the value — a graceful passthrough consistent
// with the degradation policy (00-overview.md §0.5: never hard-fail on
// unmappable/unusual data), even though the passed-through value will later
// fail ValidateImportedContact's format check.
func TestNormalizeBirthday_GarbagePassesThroughUnchanged(t *testing.T) {
	assert.Equal(t, "not-a-date", NormalizeBirthday("not-a-date"))
	assert.Equal(t, "June 29th 1958", NormalizeBirthday("June 29th 1958"))
	assert.False(t, IsValidBirthdayFormat(NormalizeBirthday("not-a-date")),
		"passed-through garbage should still fail validation downstream, but NormalizeBirthday itself must not error")
}

func TestNormalizeBirthday_EmptyInput(t *testing.T) {
	assert.Equal(t, "", NormalizeBirthday(""))
	assert.Equal(t, "", NormalizeBirthday("   "))
}

func TestNormalizeBirthday_Idempotent(t *testing.T) {
	inputs := []string{"1958-06-29", "--04-20", "29.06.1958", "29.06.", "not-a-date"}
	for _, in := range inputs {
		once := NormalizeBirthday(in)
		twice := NormalizeBirthday(once)
		assert.Equal(t, once, twice, "NormalizeBirthday should be idempotent for input %q", in)
	}
}

// --- NormalizeGender -----------------------------------------------------

func TestNormalizeGender_MaleSynonyms(t *testing.T) {
	for _, in := range []string{"m", "M", "male", "Male", "mann", "maennlich", "männlich", "masculin", "  male  "} {
		assert.Equal(t, "male", NormalizeGender(in), "input %q", in)
	}
}

func TestNormalizeGender_FemaleSynonyms(t *testing.T) {
	for _, in := range []string{"f", "F", "female", "frau", "weiblich", "feminin", "w"} {
		assert.Equal(t, "female", NormalizeGender(in), "input %q", in)
	}
}

func TestNormalizeGender_OtherSynonyms(t *testing.T) {
	for _, in := range []string{"o", "other", "andere", "divers", "d"} {
		assert.Equal(t, "other", NormalizeGender(in), "input %q", in)
	}
}

// TestNormalizeGender_PreferNotToSayMapsToOther documents an observable,
// slightly surprising behavior: even though models.Contact's Gender
// validation accepts the literal enum value "prefer_not_to_say", the
// "prefer not to say" / "keine angabe" synonyms recognized by
// NormalizeGender are mapped to "other", never to "prefer_not_to_say".
func TestNormalizeGender_PreferNotToSayMapsToOther(t *testing.T) {
	for _, in := range []string{"prefer not to say", "prefer_not_to_say", "keine angabe"} {
		assert.Equal(t, "other", NormalizeGender(in), "input %q", in)
	}
}

// TestNormalizeGender_UnrecognizedFallsBackGracefully asserts the
// degradation-policy-required behavior for unrecognized gender text: no
// error/panic, just an empty string (caller code only calls NormalizeGender
// when v != "", so the field is simply left blank rather than the import
// hard-failing).
func TestNormalizeGender_UnrecognizedFallsBackGracefully(t *testing.T) {
	assert.Equal(t, "", NormalizeGender("nonbinary"))
	assert.Equal(t, "", NormalizeGender("xyz123"))
	assert.Equal(t, "", NormalizeGender(""))
}

// --- CreateMergeNote -------------------------------------------------------

func TestCreateMergeNote_CSVChangesRecorded(t *testing.T) {
	db, _ := setupRouter()
	userID := uint(1)

	contact := models.Contact{UserID: userID, Firstname: "John", Lastname: "Doe"}
	require.NoError(t, db.Create(&contact).Error)

	newValues := map[string]interface{}{
		"firstname": "Jonathan",
		"email":     "jon@example.com",
	}

	err := CreateMergeNote(db, userID, contact.ID, &contact, newValues, "CSV")
	require.NoError(t, err)

	var notes []models.Note
	require.NoError(t, db.Find(&notes).Error)
	require.Len(t, notes, 1)

	note := notes[0]
	assert.Equal(t, userID, note.UserID)
	require.NotNil(t, note.ContactID)
	assert.Equal(t, contact.ID, *note.ContactID)
	assert.Contains(t, note.Content, "CSV Import updated this contact.")
	assert.Contains(t, note.Content, "- First Name: John → Jonathan")
	assert.Contains(t, note.Content, "- Email: (empty) → jon@example.com")
}

func TestCreateMergeNote_VCFLabelAndCirclesDiff(t *testing.T) {
	db, _ := setupRouter()
	userID := uint(1)

	contact := models.Contact{UserID: userID, Firstname: "Ada", Circles: []string{"Old"}}
	require.NoError(t, db.Create(&contact).Error)

	newValues := map[string]interface{}{"circles": "Friends, Family"}

	err := CreateMergeNote(db, userID, contact.ID, &contact, newValues, "VCF")
	require.NoError(t, err)

	var notes []models.Note
	require.NoError(t, db.Find(&notes).Error)
	require.Len(t, notes, 1)
	assert.Contains(t, notes[0].Content, "VCF Import updated this contact.")
	assert.Contains(t, notes[0].Content, "- Circles: Old → Friends, Family")
}

// TestCreateMergeNote_NoChangesCreatesNoNote asserts CreateMergeNote is a
// no-op (no Note row, no error) when nothing actually changed.
func TestCreateMergeNote_NoChangesCreatesNoNote(t *testing.T) {
	db, _ := setupRouter()
	userID := uint(1)

	contact := models.Contact{UserID: userID, Firstname: "John"}
	require.NoError(t, db.Create(&contact).Error)

	newValues := map[string]interface{}{"firstname": "John"} // identical value

	err := CreateMergeNote(db, userID, contact.ID, &contact, newValues, "VCF")
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Model(&models.Note{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

// --- diagnosticsToStrings (branch closure) --------------------------------

func TestDiagnosticsToStrings_Empty(t *testing.T) {
	assert.Nil(t, diagnosticsToStrings(nil))
	assert.Nil(t, diagnosticsToStrings([]contactmodel.Diagnostic{}))
}

func TestDiagnosticsToStrings_WithAndWithoutConcept(t *testing.T) {
	diags := []contactmodel.Diagnostic{
		{Severity: "warn", Concept: "EMAIL", Message: "dropped a field"},
		{Severity: "info", Concept: "", Message: "no concept on this one"},
	}
	out := diagnosticsToStrings(diags)
	require.Len(t, out, 2)
	assert.Equal(t, "[warn] EMAIL: dropped a field", out[0])
	assert.Equal(t, "[info] no concept on this one", out[1])
}

// --- extractPhotoFromRecord (branch closure) ------------------------------

func TestExtractPhotoFromRecord_NilRecord(t *testing.T) {
	data, mediaType, url := extractPhotoFromRecord(nil)
	assert.Nil(t, data)
	assert.Empty(t, mediaType)
	assert.Empty(t, url)
}

func TestExtractPhotoFromRecord_NoMedia(t *testing.T) {
	rec := &contactmodel.Record{}
	data, mediaType, url := extractPhotoFromRecord(rec)
	assert.Nil(t, data)
	assert.Empty(t, mediaType)
	assert.Empty(t, url)
}

func TestExtractPhotoFromRecord_MediaPresentButNotPhotoKind(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{Media: []contactmodel.Resource{
		{Kind: "logo", URI: "https://example.com/logo.png"},
	}}}
	data, mediaType, url := extractPhotoFromRecord(rec)
	assert.Nil(t, data)
	assert.Empty(t, mediaType)
	assert.Empty(t, url)
}

func TestExtractPhotoFromRecord_PhotoKindEmptyURI(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{Media: []contactmodel.Resource{
		{Kind: "photo", URI: ""},
	}}}
	data, mediaType, url := extractPhotoFromRecord(rec)
	assert.Nil(t, data)
	assert.Empty(t, mediaType)
	assert.Empty(t, url)
}

func TestExtractPhotoFromRecord_PhotoKindExternalURL(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{Media: []contactmodel.Resource{
		{Kind: "photo", URI: "https://example.com/photo.jpg", MediaType: "image/jpeg"},
	}}}
	data, mediaType, url := extractPhotoFromRecord(rec)
	assert.Nil(t, data)
	assert.Equal(t, "https://example.com/photo.jpg", url)
	_ = mediaType // media type propagation for URL photos is delegated to photostore; not asserted strictly here
}

func TestExtractPhotoFromRecord_PhotoKindEmbeddedDataURI(t *testing.T) {
	raw := []byte("fake-image-bytes")
	b64 := base64.StdEncoding.EncodeToString(raw)
	rec := &contactmodel.Record{Card: contactmodel.Card{Media: []contactmodel.Resource{
		{Kind: "photo", URI: "data:image/png;base64," + b64},
	}}}
	data, mediaType, url := extractPhotoFromRecord(rec)
	assert.Equal(t, raw, data)
	assert.Equal(t, "image/png", mediaType)
	assert.Empty(t, url)
}

// TestExtractPhotoFromRecord_FirstPhotoWins asserts only the first Media
// entry with Kind=="photo" is used, even if a non-photo entry precedes it.
func TestExtractPhotoFromRecord_FirstPhotoWins(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{Media: []contactmodel.Resource{
		{Kind: "logo", URI: "https://example.com/logo.png"},
		{Kind: "photo", URI: "https://example.com/first.jpg"},
		{Kind: "photo", URI: "https://example.com/second.jpg"},
	}}}
	_, _, url := extractPhotoFromRecord(rec)
	assert.Equal(t, "https://example.com/first.jpg", url)
}
