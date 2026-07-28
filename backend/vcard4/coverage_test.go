package vcard4

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
// correspondence.Load() and asserts that every concept with a vCard4 home
// (v4_prop not "-") has been exercised by at least one import test and one
// export test in this package (40-testing.md §40.5).
func TestCoverage_AllMappedConceptsHaveImportAndExportTests(t *testing.T) {
	for _, row := range correspondence.Load() {
		if row.V4Prop == "-" {
			continue
		}
		if !importCoverage[row.ConceptID] {
			t.Errorf("concept %q (v4_prop %q) has no registered import test coverage", row.ConceptID, row.V4Prop)
		}
		if !exportCoverage[row.ConceptID] {
			t.Errorf("concept %q (v4_prop %q) has no registered export test coverage", row.ConceptID, row.V4Prop)
		}
	}
}
