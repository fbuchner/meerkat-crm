package services

import (
	"meerkat/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

// findMapping returns the mapping for a given CSV column, or fails the test.
func findMapping(t *testing.T, mappings []models.ColumnMapping, column string) models.ColumnMapping {
	t.Helper()
	for _, m := range mappings {
		if m.CSVColumn == column {
			return m
		}
	}
	t.Fatalf("no mapping found for column %q", column)
	return models.ColumnMapping{}
}

func TestSuggestColumnMappings_GoogleStyle(t *testing.T) {
	headers := []string{
		"First Name", "Last Name", "Middle Name", "Name Prefix", "Name Suffix",
		"Organization Name", "Organization Title", "Organization Department",
		"E-mail 1 - Label", "E-mail 1 - Value", "E-mail 2 - Label", "E-mail 2 - Value",
		"Phone 1 - Label", "Phone 1 - Value",
		"Address 1 - Street", "Address 1 - City", "Address 1 - Region",
		"Address 1 - Postal Code", "Address 1 - Country", "Address 1 - Label",
		"Website 1 - Value", "Labels",
	}

	mappings := SuggestColumnMappings(headers)

	cases := []struct {
		column string
		field  string
		group  int
	}{
		{"First Name", "firstname", 0},
		{"Last Name", "lastname", 0},
		{"Middle Name", "middle_name", 0},
		{"Name Prefix", "prefix", 0},
		{"Name Suffix", "suffix", 0},
		{"Organization Name", "organization", 0},
		{"Organization Title", "job_title", 0},
		{"Organization Department", "department", 0},
		{"E-mail 1 - Label", "email_label", 0},
		{"E-mail 1 - Value", "email", 0},
		{"E-mail 2 - Label", "email_label", 1},
		{"E-mail 2 - Value", "email", 1},
		{"Phone 1 - Label", "phone_label", 0},
		{"Phone 1 - Value", "phone", 0},
		{"Address 1 - Street", "address_street", 0},
		{"Address 1 - City", "address_city", 0},
		{"Address 1 - Region", "address_region", 0},
		{"Address 1 - Postal Code", "address_postal", 0},
		{"Address 1 - Country", "address_country", 0},
		{"Address 1 - Label", "address_label", 0},
		{"Website 1 - Value", "url", 0},
		{"Labels", "circles", 0},
	}

	for _, tc := range cases {
		m := findMapping(t, mappings, tc.column)
		assert.Equal(t, tc.field, m.ContactField, "field for %q", tc.column)
		assert.Equal(t, tc.group, m.Group, "group for %q", tc.column)
	}
}

func TestSuggestColumnMappings_FlatHeaders(t *testing.T) {
	headers := []string{"Firstname", "Email", "Phone", "Vorname", "Unknown Column"}
	mappings := SuggestColumnMappings(headers)

	assert.Equal(t, "firstname", findMapping(t, mappings, "Firstname").ContactField)
	assert.Equal(t, "email", findMapping(t, mappings, "Email").ContactField)
	assert.Equal(t, "phone", findMapping(t, mappings, "Phone").ContactField)
	assert.Equal(t, "firstname", findMapping(t, mappings, "Vorname").ContactField)
	assert.Equal(t, "", findMapping(t, mappings, "Unknown Column").ContactField)
}

func TestBuildContactFromRow_MultiValue(t *testing.T) {
	headers := []string{
		"First Name", "Last Name",
		"E-mail 1 - Label", "E-mail 1 - Value", "E-mail 2 - Label", "E-mail 2 - Value",
		"Phone 1 - Label", "Phone 1 - Value",
		"Address 1 - Street", "Address 1 - City", "Address 1 - Postal Code", "Address 1 - Country", "Address 1 - Label",
		"Website 1 - Value", "Labels",
	}
	row := []string{
		"Ada", "Lovelace",
		"Home", "ada@home.example", "Work", "ada@work.example",
		"Mobile", "+44 20 7946 0000",
		"12 Baker St", "London", "NW1", "UK", "Home",
		"https://ada.example", "Friends ::: Math",
	}
	mappings := SuggestColumnMappings(headers)

	c := BuildContactFromRow(7, headers, row, mappings)

	assert.Equal(t, uint(7), c.UserID)
	assert.Equal(t, "Ada", c.Firstname)
	assert.Equal(t, "Lovelace", c.Lastname)

	// Two emails, in order, with their types.
	assert.Len(t, c.Emails, 2)
	assert.Equal(t, models.ContactEmail{Type: "home", Value: "ada@home.example"}, c.Emails[0])
	assert.Equal(t, models.ContactEmail{Type: "work", Value: "ada@work.example"}, c.Emails[1])
	assert.Equal(t, "ada@home.example", c.Email) // primary scalar synced

	// One phone, "Mobile" normalized to "cell".
	assert.Len(t, c.Phones, 1)
	assert.Equal(t, "cell", c.Phones[0].Type)
	assert.Equal(t, "+44 20 7946 0000", c.Phones[0].Value)

	// One structured address.
	assert.Len(t, c.Addresses, 1)
	assert.Equal(t, models.ContactAddress{
		Type: "home", Street: "12 Baker St", City: "London", Postal: "NW1", Country: "UK",
	}, c.Addresses[0])

	// One website.
	assert.Len(t, c.URLs, 1)
	assert.Equal(t, "https://ada.example", c.URLs[0].Value)

	// Circles parsed.
	assert.ElementsMatch(t, []string{"Friends", "Math"}, c.Circles)
}

func TestBuildContactFromRow_FlatSingleValue(t *testing.T) {
	headers := []string{"First Name", "Email", "Phone", "Address"}
	row := []string{"Bob", "bob@example.com", "555-123-4567", "1 Main St"}
	mappings := SuggestColumnMappings(headers)

	c := BuildContactFromRow(1, headers, row, mappings)

	// Single email/phone get default types and sync the legacy scalars.
	assert.Len(t, c.Emails, 1)
	assert.Equal(t, "home", c.Emails[0].Type)
	assert.Equal(t, "bob@example.com", c.Email)
	assert.Len(t, c.Phones, 1)
	assert.Equal(t, "cell", c.Phones[0].Type)
	assert.Equal(t, "555-123-4567", c.Phone)
	assert.Len(t, c.Addresses, 1)
	assert.Equal(t, "1 Main St", c.Addresses[0].Street)
	assert.Equal(t, "1 Main St", c.Address)
}

func TestBuildContactFromRow_DuplicateValueColumnsBump(t *testing.T) {
	// Two distinct columns manually mapped to "email" both default to group 0; the
	// builder bumps the second to a new group so both values survive.
	headers := []string{"First Name", "Personal Email", "Work Email"}
	row := []string{"Carol", "carol@a.example", "carol@b.example"}
	mappings := []models.ColumnMapping{
		{CSVColumn: "First Name", ContactField: "firstname"},
		{CSVColumn: "Personal Email", ContactField: "email", Group: 0},
		{CSVColumn: "Work Email", ContactField: "email", Group: 0},
	}

	c := BuildContactFromRow(1, headers, row, mappings)

	assert.Len(t, c.Emails, 2)
	assert.Equal(t, "carol@a.example", c.Emails[0].Value)
	assert.Equal(t, "carol@b.example", c.Emails[1].Value)
}

func TestValidateImportedContact(t *testing.T) {
	// Missing first name.
	assert.Contains(t, ValidateImportedContact(&models.Contact{}), "First name is required")

	// A bad email among several is flagged.
	c := models.Contact{
		Firstname: "Ada",
		Emails: []models.ContactEmail{
			{Type: "home", Value: "ada@example.com"},
			{Type: "work", Value: "not-an-email"},
		},
	}
	assert.Contains(t, ValidateImportedContact(&c), "Invalid email format")

	// All valid -> no errors.
	ok := models.Contact{
		Firstname: "Ada",
		Emails:    []models.ContactEmail{{Type: "home", Value: "ada@example.com"}},
		Birthday:  "1958-06-29",
	}
	assert.Empty(t, ValidateImportedContact(&ok))
}
