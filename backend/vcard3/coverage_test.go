package vcard3

import (
	"testing"

	"meerkat/correspondence"
)

// coverage_test.go is the 40-testing.md §40.5 coverage gate: every
// correspondence concept with a non-"-" V3Prop must have at least one import
// test and one export test. Rather than relying on test-execution ordering
// (coverage_test.go is not guaranteed to run after the import_*/export_*
// files alphabetically preceding or following it), this is a hand-maintained
// concept -> {import, export} map, kept in sync with the test files in this
// package. testedConcepts is cross-checked mechanically below against
// correspondence.Load() so a concept can't be silently forgotten.
var testedConcepts = map[string]struct{ Import, Export bool }{
	"uid":               {true, true}, // import_identity_test.go / export_identity_test.go
	"prodid":            {true, true},
	"updated":           {true, true},
	"name.full":         {true, true}, // import_name_test.go / export_name_test.go
	"name.surname":      {true, true},
	"name.given":        {true, true},
	"name.given2":       {true, true},
	"name.title":        {true, true},
	"name.credential":   {true, true},
	"nickname":          {true, true}, // import_nickname_test.go / export_nickname_test.go
	"org":               {true, true}, // import_org_test.go / export_org_test.go
	"org.unit":          {true, true},
	"title":             {true, true}, // import_titles_test.go / export_titles_test.go
	"role":              {true, true},
	"email":             {true, true}, // import_email_test.go / export_email_test.go
	"phone":             {true, true}, // import_phone_test.go / export_phone_test.go
	"impp":              {true, true}, // import_onlineservices_test.go / export_onlineservices_test.go
	"social":            {true, true},
	"adr":               {true, true}, // import_adr_test.go / export_adr_test.go
	"anniversary.birth": {true, true}, // import_anniversary_test.go / export_anniversary_test.go
	"note":              {true, true}, // import_note_test.go / export_note_test.go
	"keywords":          {true, true}, // import_keywords_test.go / export_keywords_test.go
	"photo":             {true, true}, // import_resources_test.go / export_resources_test.go
	"logo":              {true, true},
	"sound":             {true, true},
	"calendar":          {true, true},
	"freebusy":          {true, true},
	"caladruri":         {true, true},
	"key":               {true, true},
	"source":            {true, true},
	"link":              {true, true},
	"pt.vcard":          {true, true}, // import_passthrough_test.go / export_passthrough_test.go
}

func TestCoverage_EveryMappedV3ConceptHasImportAndExportTests(t *testing.T) {
	for _, row := range correspondence.Load() {
		if row.V3Prop == "-" {
			continue
		}
		got, ok := testedConcepts[row.ConceptID]
		if !ok {
			t.Errorf("concept %q (v3_prop=%q) has no entry in testedConcepts — add an import/export test", row.ConceptID, row.V3Prop)
			continue
		}
		if !got.Import {
			t.Errorf("concept %q (v3_prop=%q) has no import test", row.ConceptID, row.V3Prop)
		}
		if !got.Export {
			t.Errorf("concept %q (v3_prop=%q) has no export test", row.ConceptID, row.V3Prop)
		}
	}
}

// TestCoverage_NoStaleEntries guards the other direction: every entry in
// testedConcepts must still correspond to a real, non-"-" V3Prop row, so a
// correspondence-table change (row removed/re-scoped) can't leave a stale,
// no-longer-meaningful entry behind.
func TestCoverage_NoStaleEntries(t *testing.T) {
	byConcept := correspondence.ByConcept()
	for concept := range testedConcepts {
		row, ok := byConcept[concept]
		if !ok {
			t.Errorf("testedConcepts has stale entry %q: no such concept in correspondence.tsv", concept)
			continue
		}
		if row.V3Prop == "-" {
			t.Errorf("testedConcepts has stale entry %q: v3_prop is now \"-\" (degradation, not a mapped test target)", concept)
		}
	}
}
