// Command backfill-preferences is the one-shot T20a data migration
// (docs/fork-plan/tickets/10-T20a-preferences.md, docs/fork-plan/
// 91-envelope-data-model.md §91.9): migrate each non-empty
// Contact.FoodPreference (the legacy free-text field, being retired this
// ticket) into a structured food-category Preference row, keyed to the same
// contact by VCardUID.
//
// Requires migration 000040 (the preferences table) to have been applied —
// this command opens the DB directly (same as backfill-custom-fields) and
// does not run migrations itself.
//
// The legacy contacts.food_preference column is read via raw SQL, not the
// models.Contact struct: the Go field is retired this ticket, so the
// backfill is deliberately the LAST reader of the column (which itself stays
// in the schema — a future cleanup migration may drop it once this has
// demonstrably run everywhere).
//
// The per-contact migration logic lives in
// services.MigrateContactFoodPreference so the command and the real-DB test
// exercise the exact same function. Its safety discipline (same as
// cmd/backfill-custom-fields, its explicitly-cited precedent):
//   - Dry-run by default. Without -write, this command performs every read
//     and lookup step (so the report line reflects a real run exactly) but
//     makes zero database writes.
//   - Idempotent. A (entity, category, value) triple already migrated is
//     skipped, matching existing rows INCLUDING soft-deleted ones
//     (Unscoped), so a re-run after a delete is never misread as needing
//     re-creation.
//   - Fail-fast: any unexpected error aborts the run immediately.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"mycorrhizal/services"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	dbPath := flag.String("db", "mycorrhizal.db", "path to the SQLite database file")
	write := flag.Bool("write", false, "actually persist changes (default: dry run, report only, zero writes)")
	flag.Parse()

	db, err := gorm.Open(sqlite.Open(*dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("failed to open database %q: %v", *dbPath, err)
	}

	mode := "DRY RUN"
	if *write {
		mode = "WRITE"
	}

	var contacts []services.LegacyFoodContact
	if err := db.Raw(`
		SELECT id, user_id, vcard_uid, food_preference
		FROM contacts
		WHERE food_preference IS NOT NULL AND food_preference != ''
	`).Scan(&contacts).Error; err != nil {
		log.Fatalf("failed to load contacts: %v", err)
	}

	fmt.Printf("backfill-preferences: mode=%s db=%s contacts_with_food_preference=%d\n\n", mode, *dbPath, len(contacts))

	var created, skipped int
	for _, contact := range contacts {
		outcome, err := services.MigrateContactFoodPreference(db, contact, *write)
		if err != nil {
			log.Fatalf("failed to migrate food preference contact_id=%d: %v", contact.ID, err)
		}
		if outcome.Skipped != "" {
			skipped++
			fmt.Printf("SKIP  contact_id=%d vcard_uid=%q reason=%q\n", contact.ID, contact.VCardUID, outcome.Skipped)
			continue
		}
		created++
		action := "PLAN "
		if *write {
			action = "WRITE"
		}
		fmt.Printf("%s contact_id=%d vcard_uid=%q food=%q preference_id=%q\n",
			action, contact.ID, contact.VCardUID, contact.FoodPreference, outcome.PreferenceID)
	}

	fmt.Printf("\n%s complete: %d preferences, %d skipped\n", mode, created, skipped)
	if !*write {
		fmt.Println("No database changes were made (dry run). Re-run with -write to persist.")
	}

	os.Exit(0)
}
