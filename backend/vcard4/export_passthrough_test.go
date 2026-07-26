package vcard4

import (
	"encoding/json"
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

func init() {
	registerExportCoverage("pt.vcard", "pt.jscontact")
}

func TestExport_PassthroughUnknownProperty(t *testing.T) {
	valJSON, _ := json.Marshal("1;urn:uuid:53e374d9-337e-4727-8803-a1e9c14e0556")
	rec := &contactmodel.Record{
		Card: contactmodel.Card{Name: &contactmodel.Name{Full: "Test"}},
		Passthrough: contactmodel.Passthrough{
			VCard: []contactmodel.JCardProp{{Name: "clientpidmap", Type: "text", Value: valJSON}},
		},
	}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "CLIENTPIDMAP", nil, "1;urn:uuid:53e374d9-337e-4727-8803-a1e9c14e0556")
}

func TestExport_PassthroughDeDupGuard(t *testing.T) {
	// 20.5's de-dup guard: a passthrough entry whose name collides with a
	// mapped property must NOT also be re-emitted verbatim.
	valJSON, _ := json.Marshal("stray-uid-should-not-appear")
	rec := &contactmodel.Record{
		Card: contactmodel.Card{UID: "real-uid", Name: &contactmodel.Name{Full: "Test"}},
		Passthrough: contactmodel.Passthrough{
			VCard: []contactmodel.JCardProp{{Name: "uid", Type: "text", Value: valJSON}},
		},
	}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	dec, err := parseVCardForTest(out)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if len(dec["UID"]) != 1 || dec["UID"][0].Value != "real-uid" {
		t.Errorf("UID fields = %+v, want exactly one field with value real-uid", dec["UID"])
	}
}

func TestExport_PassthroughJSProp(t *testing.T) {
	rec := &contactmodel.Record{
		Card: contactmodel.Card{Name: &contactmodel.Name{Full: "Test"}},
		Passthrough: contactmodel.Passthrough{
			JSContact: map[string]json.RawMessage{"/custom/field": json.RawMessage(`"hello"`)},
		},
	}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "JSPROP", map[string]string{"JSPTR": "/custom/field"}, `"hello"`)
}
