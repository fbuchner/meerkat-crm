package jscontact

import (
	"testing"

	"mycorrhizal/contactmodel"
	"mycorrhizal/internal/rfctest"
)

func init() {
	registerExportCoverage("org", "org.unit", "title", "role")
}

func TestExport_TitleRoleOrg(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "title-role-example",
		Titles: []contactmodel.Title{
			{ID: "TITLE-1", Kind: "title", Name: "Research Scientist"},
			{ID: "TITLE-2", Kind: "role", Name: "Project Leader", OrganizationID: "ORG-1"},
		},
		Organizations: []contactmodel.Organization{
			{ID: "ORG-1", Name: "ABC, Inc."},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/titles/TITLE-1/kind", "title")
	rfctest.AssertJSONPointer(t, out, "/titles/TITLE-1/name", "Research Scientist")
	rfctest.AssertJSONPointer(t, out, "/titles/TITLE-2/kind", "role")
	rfctest.AssertJSONPointer(t, out, "/titles/TITLE-2/organizationId", "ORG-1")
	rfctest.AssertJSONPointer(t, out, "/organizations/ORG-1/name", "ABC, Inc.")
}

func TestExport_OrgUnit(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "org-unit-example",
		Organizations: []contactmodel.Organization{
			{ID: "ORG-1", Name: "ABC, Inc.", Units: []contactmodel.OrgUnit{
				{Name: "North American Division"},
				{Name: "Marketing"},
			}},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/organizations/ORG-1/units/0/name", "North American Division")
	rfctest.AssertJSONPointer(t, out, "/organizations/ORG-1/units/1/name", "Marketing")
}
