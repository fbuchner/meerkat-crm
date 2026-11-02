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

func TestContactSaveDoesNotResurrectEndedJob(t *testing.T) {
	db := setupTestDB(t)

	contact := Contact{UserID: 1, Firstname: "Jane"}
	db.Create(&contact)
	db.Create(&EmploymentHistory{UserID: 1, ContactID: contact.ID, Organization: "Acme", EndDate: "2023-01-01"})

	// Stale CardDAV sync pushes the old job back.
	contact.Organization = "Acme"
	contact.RevertStaleEmployment = true
	if err := db.Save(&contact).Error; err != nil {
		t.Fatalf("save failed: %v", err)
	}

	var open int64
	db.Model(&EmploymentHistory{}).Where("contact_id = ? AND end_date = ''", contact.ID).Count(&open)
	if open != 0 {
		t.Errorf("expected no open entry, got %d", open)
	}
	var got Contact
	db.First(&got, contact.ID)
	if got.Organization != "" {
		t.Errorf("expected scalars reverted, got %q", got.Organization)
	}
}

// Only CardDAV opts into RevertStaleEmployment: an import is a deliberate write
// (e.g. rehired at the same company), so it reopens the ended job.
func TestContactSaveWithoutRevertFlagReopensEndedJob(t *testing.T) {
	db := setupTestDB(t)

	contact := Contact{UserID: 1, Firstname: "Jane"}
	db.Create(&contact)
	db.Create(&EmploymentHistory{UserID: 1, ContactID: contact.ID, Organization: "Acme", EndDate: "2023-01-01"})

	contact.Organization = "Acme"
	if err := db.Save(&contact).Error; err != nil {
		t.Fatalf("save failed: %v", err)
	}

	var open int64
	db.Model(&EmploymentHistory{}).Where("contact_id = ? AND end_date = ''", contact.ID).Count(&open)
	if open != 1 {
		t.Errorf("expected one open entry, got %d", open)
	}
}

func TestContactSaveRevertsAgainstLatestEndedEntryAcrossPrecisions(t *testing.T) {
	db := setupTestDB(t)

	contact := Contact{UserID: 1, Firstname: "Jane"}
	db.Create(&contact)
	// "2020" ends 2020-12-31, later than "2020-05", though it sorts before it as a string.
	db.Create(&EmploymentHistory{UserID: 1, ContactID: contact.ID, Organization: "Old", EndDate: "2020-05"})
	db.Create(&EmploymentHistory{UserID: 1, ContactID: contact.ID, Organization: "Acme", EndDate: "2020"})

	contact.Organization = "Acme"
	contact.RevertStaleEmployment = true
	if err := db.Save(&contact).Error; err != nil {
		t.Fatalf("save failed: %v", err)
	}

	var got Contact
	db.First(&got, contact.ID)
	if got.Organization != "" {
		t.Errorf("expected scalars reverted, got %q", got.Organization)
	}
}

// A stale push of the old job must not overwrite the job that replaced it.
func TestContactSaveStalePushKeepsNewerCurrentJob(t *testing.T) {
	db := setupTestDB(t)

	contact := Contact{UserID: 1, Firstname: "Jane"}
	db.Create(&contact)
	db.Create(&EmploymentHistory{UserID: 1, ContactID: contact.ID, Organization: "OldCo", EndDate: "2023-01-01"})
	current := EmploymentHistory{UserID: 1, ContactID: contact.ID, Organization: "NewCo", StartDate: "2023-02"}
	db.Create(&current)

	contact.Organization = "OldCo"
	contact.RevertStaleEmployment = true
	if err := db.Save(&contact).Error; err != nil {
		t.Fatalf("save failed: %v", err)
	}

	var entry EmploymentHistory
	db.First(&entry, current.ID)
	if entry.Organization != "NewCo" {
		t.Errorf("current entry overwritten with %q", entry.Organization)
	}
	var got Contact
	db.First(&got, contact.ID)
	if got.Organization != "NewCo" {
		t.Errorf("expected scalars reset to the current job, got %q", got.Organization)
	}
}

// New values from the same stale-guarded path still update the current entry.
func TestContactSaveRevertFlagStillAppliesNewValues(t *testing.T) {
	db := setupTestDB(t)

	contact := Contact{UserID: 1, Firstname: "Jane"}
	db.Create(&contact)
	db.Create(&EmploymentHistory{UserID: 1, ContactID: contact.ID, Organization: "OldCo", EndDate: "2023-01-01"})
	current := EmploymentHistory{UserID: 1, ContactID: contact.ID, Organization: "NewCo"}
	db.Create(&current)

	contact.Organization = "RenamedCo"
	contact.RevertStaleEmployment = true
	if err := db.Save(&contact).Error; err != nil {
		t.Fatalf("save failed: %v", err)
	}

	var entry EmploymentHistory
	db.First(&entry, current.ID)
	if entry.Organization != "RenamedCo" {
		t.Errorf("expected current entry updated, got %q", entry.Organization)
	}
}

// Blanking the scalars doesn't end the job, so the contact must keep showing it.
func TestContactSaveBlankScalarsRederivedFromCurrentEntry(t *testing.T) {
	db := setupTestDB(t)

	contact := Contact{UserID: 1, Firstname: "Jane", Organization: "Acme", JobTitle: "Engineer"}
	db.Create(&contact)

	contact.Organization, contact.JobTitle = "", ""
	if err := db.Save(&contact).Error; err != nil {
		t.Fatalf("save failed: %v", err)
	}

	var open int64
	db.Model(&EmploymentHistory{}).Where("contact_id = ? AND end_date = ''", contact.ID).Count(&open)
	if open != 1 {
		t.Errorf("expected the entry to stay open, got %d", open)
	}
	if contact.Organization != "Acme" {
		t.Errorf("expected in-memory scalars restored, got %q", contact.Organization)
	}
	var got Contact
	db.First(&got, contact.ID)
	if got.Organization != "Acme" || got.JobTitle != "Engineer" {
		t.Errorf("expected scalars restored, got %q / %q", got.Organization, got.JobTitle)
	}
}
