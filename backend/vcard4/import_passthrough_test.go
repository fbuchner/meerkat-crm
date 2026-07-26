package vcard4

import (
	"encoding/json"
	"strings"
	"testing"

	"meerkat/internal/rfctest"
)

// Concepts: pt.vcard, pt.jscontact.
// Golden fixture: rfc6350-baseline.v4.vcf carries a CLIENTPIDMAP property,
// which has no correspondence row and must be preserved verbatim via
// Passthrough.VCard (RFC 9555 "vCardProps" shape). (language.v4.vcf's
// LANGUAGE property used to be exercised here too, back when the "language"
// row's v4_prop was wrongly "-"; the table has since been corrected — RFC
// 9554 §3.3 LANGUAGE is a real 1:1 mapping to Card.Language — so that
// fixture now belongs to import_identity_test.go/export_identity_test.go
// instead of here.)
func init() {
	registerImportCoverage("pt.vcard", "pt.jscontact")
}

func TestImport_PassthroughUnknownProperty(t *testing.T) {
	raw := rfctest.LoadFixture("rfc6350-baseline.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	found := false
	for _, p := range rec.Passthrough.VCard {
		if strings.EqualFold(p.Name, "CLIENTPIDMAP") {
			found = true
			var val string
			_ = json.Unmarshal(p.Value, &val)
			if val != "1;urn:uuid:53e374d9-337e-4727-8803-a1e9c14e0556" {
				t.Errorf("CLIENTPIDMAP value = %q", val)
			}
		}
	}
	if !found {
		t.Errorf("Passthrough.VCard = %+v, want a CLIENTPIDMAP entry", rec.Passthrough.VCard)
	}
}

func TestImport_PassthroughJSProp(t *testing.T) {
	raw := []byte("BEGIN:VCARD\r\nVERSION:4.0\r\nUID:jsprop-example\r\nFN:Test\r\nJSPROP;JSPTR=/custom/field:\"hello\"\r\nEND:VCARD\r\n")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	raw2, ok := rec.Passthrough.JSContact["/custom/field"]
	if !ok {
		t.Fatalf("Passthrough.JSContact = %+v, want key /custom/field", rec.Passthrough.JSContact)
	}
	if string(raw2) != `"hello"` {
		t.Errorf("JSContact[/custom/field] = %s, want \"hello\"", raw2)
	}
}
