package jscontact

import "fmt"

// Small literal-construction helpers shared by the import_*/export_*/roundtrip
// test files in this package (std `testing` only, no testify — 0.6).

func intPtr(v int) *int    { return &v }
func boolPtr(v bool) *bool { return &v }

// sprintfPointer builds a JSON pointer string, e.g. sprintfPointer("/name/components/%d/kind", 2).
func sprintfPointer(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}
