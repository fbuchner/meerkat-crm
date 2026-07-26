package vcard4

import (
	"bytes"

	vcard "github.com/emersion/go-vcard"
)

// Small literal-construction helpers shared by the import_*/export_*/
// roundtrip test files in this package (std `testing` only, no testify —
// 00-overview.md §0.6).

func intPtr(v int) *int { return &v }

// parseVCardForTest decodes raw with the same go-vcard decoder the adapter
// uses, for tests that need to inspect field Group/Params directly (beyond
// what rfctest.AssertVCardLine's single-property-line assertion supports —
// e.g. checking that two different properties share a GROUP prefix).
func parseVCardForTest(raw []byte) (vcard.Card, error) {
	return vcard.NewDecoder(bytes.NewReader(raw)).Decode()
}
