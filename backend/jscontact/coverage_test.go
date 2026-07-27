package jscontact

import (
	"testing"

	"mycorrhizal/correspondence"
)

// importCoverage / exportCoverage are populated by each import_*_test.go /
// export_*_test.go file's init(), one entry per correspondence concept_id it
// exercises. init() (rather than inside a Test func body) so registration
// happens unconditionally whenever the test binary starts, independent of
// any -run filter — this is what makes TestCoverage_* mechanically reliable
// rather than aspirational (40-testing.md §40.5).
var (
	importCoverage = map[string]bool{}
	exportCoverage = map[string]bool{}
)

func registerImportCoverage(conceptIDs ...string) {
	for _, id := range conceptIDs {
		importCoverage[id] = true
	}
}

func registerExportCoverage(conceptIDs ...string) {
	for _, id := range conceptIDs {
		exportCoverage[id] = true
	}
}

// TestCoverage_AllMappedConceptsHaveImportAndExportTests iterates
// correspondence.Load() and asserts that every concept with a JSContact home
// (js_ptr not "" and not the "-" no-mapping sentinel, meaning the row
// explicitly records that this concept has no JSContact property at all) has
// been exercised by at least one import test and one export test in this
// package. ("created" used to be such a "-" row and was excluded here; a
// later correspondence-table audit corrected it to js_ptr "/created" — RFC
// 9553 §2.1.3 defines Card.created as a real, optional property — and it is
// now mapped and registered like any other identity/meta concept; see
// import_identity_test.go / export_identity_test.go.)
func TestCoverage_AllMappedConceptsHaveImportAndExportTests(t *testing.T) {
	for _, row := range correspondence.Load() {
		if row.JSPtr == "" || row.JSPtr == "-" {
			continue
		}
		if !importCoverage[row.ConceptID] {
			t.Errorf("concept %q (js_ptr %q) has no registered import test coverage", row.ConceptID, row.JSPtr)
		}
		if !exportCoverage[row.ConceptID] {
			t.Errorf("concept %q (js_ptr %q) has no registered export test coverage", row.ConceptID, row.JSPtr)
		}
	}
}
