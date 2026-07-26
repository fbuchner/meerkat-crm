package vcard3

import (
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

// Concepts covered (coverage_test.go): org, org.unit.

func TestExport_Org(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Organizations: []contactmodel.Organization{{
			Name:  "Lotus Development Corporation",
			Units: []contactmodel.OrgUnit{{Name: "Sales"}, {Name: "East Region"}},
		}},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropOrg, nil, "Lotus Development Corporation;Sales;East Region")
}
