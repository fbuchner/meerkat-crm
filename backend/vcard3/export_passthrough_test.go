package vcard3

import (
	"encoding/json"
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

// Concept covered: pt.vcard.
func init() {
	registerExportCoverage("pt.vcard")
}

func TestExport_PassthroughVCard(t *testing.T) {
	val, _ := json.Marshal("hello")
	rec := &contactmodel.Record{Passthrough: contactmodel.Passthrough{
		VCard: []contactmodel.JCardProp{{Name: "x-custom-prop", Type: "text", Value: val}},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "X-CUSTOM-PROP", nil, "hello")
}

// De-dup guard (20.5): a passthrough entry whose name is already produced by
// a mapped row (e.g. UID) must not be re-emitted a second time.
func TestExport_PassthroughVCard_DedupGuard(t *testing.T) {
	val, _ := json.Marshal("should-not-appear")
	rec := &contactmodel.Record{
		Card: contactmodel.Card{UID: "real-uid"},
		Passthrough: contactmodel.Passthrough{
			VCard: []contactmodel.JCardProp{{Name: "uid", Type: "text", Value: val}},
		},
	}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropUID, nil, "real-uid")
}
