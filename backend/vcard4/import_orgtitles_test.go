package vcard4

import (
	"testing"

	"meerkat/internal/rfctest"
)

// Concepts: org, org.unit, title, role.
// Golden fixture: title-role.v4.vcf (RFC 9555 §2.9.6) — organizationId
// derived from a shared vCard property GROUP (20.7).
func init() {
	registerImportCoverage("org", "org.unit", "title", "role")
}

func TestImport_OrgUnits(t *testing.T) {
	raw := rfctest.LoadFixture("org-unit.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Organizations) != 1 {
		t.Fatalf("Organizations = %+v", rec.Card.Organizations)
	}
	o := rec.Card.Organizations[0]
	if o.Name != "Example Corp" {
		t.Errorf("Name = %q, want Example Corp", o.Name)
	}
	if len(o.Units) != 2 || o.Units[0].Name != "Sales" || o.Units[1].Name != "East" {
		t.Errorf("Units = %+v, want [Sales East]", o.Units)
	}
}

func TestImport_TitleRoleOrganizationID(t *testing.T) {
	raw := rfctest.LoadFixture("title-role.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Organizations) != 1 || rec.Card.Organizations[0].Name != "ABC, Inc." {
		t.Fatalf("Organizations = %+v", rec.Card.Organizations)
	}
	orgID := rec.Card.Organizations[0].ID
	if len(rec.Card.Titles) != 2 {
		t.Fatalf("Titles = %+v", rec.Card.Titles)
	}
	var foundTitle, foundRole bool
	for _, ti := range rec.Card.Titles {
		switch ti.Kind {
		case "title":
			foundTitle = true
			if ti.Name != "Research Scientist" {
				t.Errorf("title.Name = %q, want Research Scientist", ti.Name)
			}
			if ti.OrganizationID != "" {
				t.Errorf("TITLE has no group in the fixture; OrganizationID should be empty, got %q", ti.OrganizationID)
			}
		case "role":
			foundRole = true
			if ti.Name != "Project Leader" {
				t.Errorf("role.Name = %q, want Project Leader", ti.Name)
			}
			if ti.OrganizationID != orgID {
				t.Errorf("ROLE.OrganizationID = %q, want %q (shared GROUP with ORG)", ti.OrganizationID, orgID)
			}
		}
	}
	if !foundTitle || !foundRole {
		t.Errorf("Titles = %+v, want one kind=title and one kind=role", rec.Card.Titles)
	}
}
