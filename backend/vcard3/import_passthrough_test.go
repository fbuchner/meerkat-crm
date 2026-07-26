package vcard3

import (
	"encoding/json"
	"testing"
)

// Concept covered: pt.vcard.
func init() {
	registerImportCoverage("pt.vcard")
}

const passthroughImportVCF = "BEGIN:VCARD\n" +
	"VERSION:3.0\n" +
	"FN:Frank Dawson\n" +
	"X-CUSTOM-PROP:hello\n" +
	"END:VCARD\n"

func TestImport_PassthroughVCard(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(passthroughImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	var found bool
	for _, jp := range rec.Passthrough.VCard {
		if jp.Name == "x-custom-prop" {
			found = true
			var s string
			if uerr := json.Unmarshal(jp.Value, &s); uerr == nil && s != "hello" {
				t.Errorf("passthrough value = %q, want %q", s, "hello")
			}
		}
	}
	if !found {
		t.Errorf("Passthrough.VCard = %+v, want an x-custom-prop entry", rec.Passthrough.VCard)
	}
}
