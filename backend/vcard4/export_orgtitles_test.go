package vcard4

import (
	"testing"

	"mycorrhizal/contactmodel"
	"mycorrhizal/internal/rfctest"
)

func init() {
	registerExportCoverage("org", "org.unit", "title", "role")
}

func TestExport_OrgUnits(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Organizations: []contactmodel.Organization{{Name: "Example Corp", Units: []contactmodel.OrgUnit{{Name: "Sales"}, {Name: "East"}}}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "ORG", nil, "Example Corp;Sales;East")
}

func TestExport_TitleRoleOrganizationID(t *testing.T) {
	// Reproduces golden fixture title-role.v4.vcf's GROUP-linking mechanism
	// (RFC 9555 §2.9.6): a Title with Kind=role and a matching
	// OrganizationID must share a synthetic GROUP prefix with its ORG.
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Organizations: []contactmodel.Organization{{ID: "ORG-1", Name: "ABC, Inc."}},
		Titles: []contactmodel.Title{
			{Name: "Research Scientist", Kind: "title"},
			{Name: "Project Leader", Kind: "role", OrganizationID: "ORG-1"},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "TITLE", nil, "Research Scientist")

	dec, err := parseVCardForTest(out)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	roleField := dec["ROLE"]
	orgField := dec["ORG"]
	if len(roleField) != 1 || len(orgField) != 1 {
		t.Fatalf("ROLE=%v ORG=%v", roleField, orgField)
	}
	if roleField[0].Group == "" || roleField[0].Group != orgField[0].Group {
		t.Errorf("ROLE.Group = %q, ORG.Group = %q, want equal and non-empty", roleField[0].Group, orgField[0].Group)
	}
}
