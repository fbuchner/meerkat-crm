package jscontact

import (
	"testing"

	"meerkat/internal/rfctest"
)

// Concepts: org, org.unit, title, role.
// Rows: org         Card.Organizations[].Name        /organizations/{id}/name       org_units
//
//	org.unit    Card.Organizations[].Units[].Name /organizations/{id}/units      org_units
//	title       Card.Titles[kind=title].Name      /titles/{id}/name              identity
//	role        Card.Titles[kind=role].Name       /titles/{id}/name              identity
func init() {
	registerImportCoverage("org", "org.unit", "title", "role")
}

func TestImport_TitleRoleOrg(t *testing.T) {
	// title-role.jscontact.json (RFC 9555 §2.9.6 golden fixture).
	raw := rfctest.LoadFixture("title-role.jscontact.json")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Titles) != 2 {
		t.Fatalf("len(Titles) = %d, want 2", len(rec.Card.Titles))
	}
	if rec.Card.Titles[0].ID != "TITLE-1" || rec.Card.Titles[0].Kind != "title" || rec.Card.Titles[0].Name != "Research Scientist" {
		t.Errorf("Titles[0] = %+v", rec.Card.Titles[0])
	}
	if rec.Card.Titles[1].ID != "TITLE-2" || rec.Card.Titles[1].Kind != "role" || rec.Card.Titles[1].Name != "Project Leader" {
		t.Errorf("Titles[1] = %+v", rec.Card.Titles[1])
	}
	if rec.Card.Titles[1].OrganizationID != "ORG-1" {
		t.Errorf("Titles[1].OrganizationID = %q, want ORG-1", rec.Card.Titles[1].OrganizationID)
	}
	if len(rec.Card.Organizations) != 1 || rec.Card.Organizations[0].ID != "ORG-1" || rec.Card.Organizations[0].Name != "ABC, Inc." {
		t.Errorf("Organizations = %+v", rec.Card.Organizations)
	}
}

func TestImport_OrgUnit(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "org-unit-example",
		"organizations": {
			"ORG-1": {
				"@type": "Organization", "name": "ABC, Inc.",
				"units": [
					{ "@type": "OrgUnit", "name": "North American Division" },
					{ "@type": "OrgUnit", "name": "Marketing" }
				]
			}
		}
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Organizations) != 1 {
		t.Fatalf("len(Organizations) = %d, want 1", len(rec.Card.Organizations))
	}
	units := rec.Card.Organizations[0].Units
	if len(units) != 2 || units[0].Name != "North American Division" || units[1].Name != "Marketing" {
		t.Errorf("Units = %+v", units)
	}
}
