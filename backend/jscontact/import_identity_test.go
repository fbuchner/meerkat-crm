package jscontact

import (
	"testing"

	"mycorrhizal/internal/rfctest"
)

// Concepts: uid, kind, prodid, created, updated, language.
// Rows (docs/fork-plan/20-correspondence.md §20.3 "Identity / meta"):
//
//	uid      Card.UID           /uid       identity
//	kind     Card.Kind          /kind      identity
//	prodid   Card.ProdID        /prodId    identity
//	created  Card.Created       /created   ts_rfc3339
//	updated  Card.Updated       /updated   ts_rfc3339
//	language Card.Language      /language  identity
//
// ("created" was previously excluded here: an earlier version of the
// correspondence table wrongly marked its row js_ptr "-". A later audit
// corrected the table — RFC 9553 §2.1.3 defines Card.created as a real,
// optional Card-level UTCDateTime — and importIdentity/exportIdentity
// (adapter.go) now map it exactly like Updated; see TestImport_Created
// below.)
func init() {
	registerImportCoverage("uid", "kind", "prodid", "created", "updated", "language")
}

func TestImport_UIDAndKind(t *testing.T) {
	// johndoe.jscontact.json is the RFC 9553 Fig. 6 golden fixture: uid =
	// "22B2C7DF-9120-4969-8460-05956FE6B065", kind = "individual".
	raw := rfctest.LoadFixture("johndoe.jscontact.json")
	rec, diags, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(diags) != 0 {
		t.Errorf("diags = %v, want none", diags)
	}
	if rec.Card.UID != "22B2C7DF-9120-4969-8460-05956FE6B065" {
		t.Errorf("Card.UID = %q", rec.Card.UID)
	}
	if rec.Card.Kind != "individual" {
		t.Errorf("Card.Kind = %q, want individual", rec.Card.Kind)
	}
}

func TestImport_ProdID(t *testing.T) {
	raw := []byte(`{"@type":"Card","version":"1.0","uid":"prodid-example","prodId":"-//ACME//AddressBook//EN"}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.ProdID != "-//ACME//AddressBook//EN" {
		t.Errorf("Card.ProdID = %q", rec.Card.ProdID)
	}
}

func TestImport_Updated(t *testing.T) {
	raw := []byte(`{"@type":"Card","version":"1.0","uid":"updated-example","updated":{"@type":"Timestamp","utc":"2021-08-24T18:30:00Z"}}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.Updated == nil || rec.Card.Updated.UTC != "2021-08-24T18:30:00Z" {
		t.Errorf("Card.Updated = %+v", rec.Card.Updated)
	}
}

func TestImport_Language(t *testing.T) {
	raw := []byte(`{"@type":"Card","version":"1.0","uid":"language-example","language":"de-AT"}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.Language != "de-AT" {
		t.Errorf("Card.Language = %q, want de-AT", rec.Card.Language)
	}
}

// TestImport_Created exercises the "created" row (js_ptr "/created",
// corrected from a wrongly-recorded "-"): a top-level "created" property in a
// JSContact document maps to Record.Card.Created, exactly like "updated"
// maps to Record.Card.Updated (TestImport_Updated, above). It must NOT be
// captured into Passthrough.JSContact — it now has a real neutral home, so
// treating it as an unmapped top-level key would be a (mapped-property)
// regression, not a degradation-policy concern.
func TestImport_Created(t *testing.T) {
	raw := []byte(`{"@type":"Card","version":"1.0","uid":"created-example","created":{"@type":"Timestamp","utc":"2021-08-24T18:30:00Z"}}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.Created == nil || rec.Card.Created.UTC != "2021-08-24T18:30:00Z" {
		t.Errorf("Card.Created = %+v, want UTC 2021-08-24T18:30:00Z", rec.Card.Created)
	}
	if _, ok := rec.Passthrough.JSContact["/created"]; ok {
		t.Errorf(`Passthrough.JSContact["/created"] should not be populated now that "created" is a mapped property`)
	}
}
