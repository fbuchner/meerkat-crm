package services

import (
	"meerkat/models"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing, with both
// Contact and EmploymentHistory models migrated.
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	// Migrate both Contact and EmploymentHistory models
	if err := db.AutoMigrate(&models.Contact{}, &models.EmploymentHistory{}); err != nil {
		t.Fatalf("failed to migrate models: %v", err)
	}

	return db
}

func TestRecomputeContactEmploymentScalars_PicksMostRecentOpenEntry(t *testing.T) {
	db := setupTestDB(t)

	contact := models.Contact{UserID: 1, Firstname: "Jane"}
	db.Create(&contact)

	db.Create(&models.EmploymentHistory{UserID: 1, ContactID: contact.ID, Organization: "Old Co", StartDate: "2015", EndDate: ""})
	db.Create(&models.EmploymentHistory{UserID: 1, ContactID: contact.ID, Organization: "New Co", StartDate: "2022", EndDate: ""})
	db.Create(&models.EmploymentHistory{UserID: 1, ContactID: contact.ID, Organization: "Closed Co", StartDate: "2023", EndDate: "2024"})

	if err := RecomputeContactEmploymentScalars(db, 1, contact.ID); err != nil {
		t.Fatalf("RecomputeContactEmploymentScalars failed: %v", err)
	}

	var updated models.Contact
	db.First(&updated, contact.ID)
	if updated.Organization != "New Co" {
		t.Errorf("expected 'New Co' (most recent open entry), got %q", updated.Organization)
	}
}

func TestRecomputeContactEmploymentScalars_ClearsWhenNoOpenEntries(t *testing.T) {
	db := setupTestDB(t)

	contact := models.Contact{UserID: 1, Firstname: "Jane", Organization: "Stale Co"}
	db.Create(&contact)
	db.Create(&models.EmploymentHistory{UserID: 1, ContactID: contact.ID, Organization: "Stale Co", StartDate: "2020", EndDate: "2021"})

	if err := RecomputeContactEmploymentScalars(db, 1, contact.ID); err != nil {
		t.Fatalf("RecomputeContactEmploymentScalars failed: %v", err)
	}

	var updated models.Contact
	db.First(&updated, contact.ID)
	if updated.Organization != "" {
		t.Errorf("expected empty organization when no open entries remain, got %q", updated.Organization)
	}
}

func TestSortEmploymentHistory_OpenEntriesFirst(t *testing.T) {
	entries := []models.EmploymentHistory{
		{Organization: "Closed", StartDate: "2023", EndDate: "2024"},
		{Organization: "OpenOlder", StartDate: "2015", EndDate: ""},
		{Organization: "OpenNewer", StartDate: "2022", EndDate: ""},
	}
	SortEmploymentHistory(entries)

	if entries[0].Organization != "OpenNewer" || entries[1].Organization != "OpenOlder" || entries[2].Organization != "Closed" {
		got := []string{entries[0].Organization, entries[1].Organization, entries[2].Organization}
		t.Errorf("unexpected order: %v", got)
	}
}
