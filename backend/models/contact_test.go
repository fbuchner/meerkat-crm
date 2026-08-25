package models

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&Contact{}, &EmploymentHistory{}); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	return db
}

func TestContactSaveCreatesOpenEmploymentHistoryEntry(t *testing.T) {
	db := setupTestDB(t)

	contact := Contact{UserID: 1, Firstname: "Jane", Organization: "Acme", JobTitle: "Engineer"}
	if err := db.Create(&contact).Error; err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	var entries []EmploymentHistory
	db.Where("contact_id = ?", contact.ID).Find(&entries)
	if len(entries) != 1 {
		t.Fatalf("expected 1 employment history entry, got %d", len(entries))
	}
	if entries[0].Organization != "Acme" || entries[0].JobTitle != "Engineer" || entries[0].EndDate != "" {
		t.Errorf("unexpected entry: %+v", entries[0])
	}
}

func TestContactSavePatchesExistingOpenEntry(t *testing.T) {
	db := setupTestDB(t)

	contact := Contact{UserID: 1, Firstname: "Jane", Organization: "Acme"}
	db.Create(&contact)

	// Simulate an org change coming from a CardDAV sync
	contact.Organization = "Globex"
	if err := db.Save(&contact).Error; err != nil {
		t.Fatalf("failed to save contact: %v", err)
	}

	var entries []EmploymentHistory
	db.Where("contact_id = ?", contact.ID).Find(&entries)
	if len(entries) != 1 {
		t.Fatalf("expected still 1 employment history entry (patched, not duplicated), got %d", len(entries))
	}
	if entries[0].Organization != "Globex" {
		t.Errorf("expected patched org 'Globex', got %q", entries[0].Organization)
	}
}

func TestContactSavePicksMostRecentStartDateAmongOpenEntries(t *testing.T) {
	db := setupTestDB(t)

	contact := Contact{UserID: 1, Firstname: "Jane"}
	db.Create(&contact)

	db.Create(&EmploymentHistory{UserID: 1, ContactID: contact.ID, Organization: "Old Co", StartDate: "2015"})
	db.Create(&EmploymentHistory{UserID: 1, ContactID: contact.ID, Organization: "New Co", StartDate: "2022"})

	// Any save with non-empty scalars re-triggers reconciliation; use the
	// value that should already be authoritative (the 2022 entry) to prove
	// the "most recent StartDate wins" tie-break patches the right row.
	contact.Organization = "New Co Renamed"
	db.Save(&contact)

	var newer EmploymentHistory
	db.Where("contact_id = ? AND start_date = ?", contact.ID, "2022").First(&newer)
	var older EmploymentHistory
	db.Where("contact_id = ? AND start_date = ?", contact.ID, "2015").First(&older)

	if newer.Organization != "New Co Renamed" {
		t.Errorf("expected the 2022 entry to be patched, got %q", newer.Organization)
	}
	if older.Organization != "Old Co" {
		t.Errorf("expected the 2015 entry untouched, got %q", older.Organization)
	}
}
