// Command backfill-custom-fields is the one-shot WP-84b data migration
// (docs/fork-plan/92-delivery-roadmap.md, docs/fork-plan/94-custom-fields.md
// §94.6): migrate the existing untyped v1 custom fields (models.User.
// CustomFieldNames + models.Contact.CustomFields) into the typed v2 system
// (models.FieldDefinition + models.FieldValue, built this WP).
//
// This does NOT touch User.CustomFieldNames/Contact.CustomFields or any of
// their live call sites (controllers/user_controller.go's
// GetCustomFieldNames/UpdateCustomFieldNames, controllers/export_controller.
// go's CSV columns, the CustomFieldsSettings/AddContactDialog frontend
// pages) -- they keep working exactly as before. Cutting them over to the
// new system is a separate, later step.
//
// Runs in two passes, since a FieldValue references its FieldDefinition:
// first every user's CustomFieldNames becomes a FieldDefinition, then every
// contact's CustomFields becomes a FieldValue looked up against those
// definitions.
//
// Same safety discipline as cmd/backfill-relationship-edges (WP-81, the
// explicitly-cited precedent):
//   - Dry-run by default. Without -write, this command performs every read
//     and lookup step (so the report line reflects a real run exactly) but
//     makes zero database writes.
//   - Idempotent. A name/value pair already migrated is skipped. -force
//     applies only to the values pass (a FieldValue can drift from v1 after
//     an initial migration; a FieldDefinition never does, so it has no
//     -force override -- see migrate.go's migrateUserFieldDefinitions doc).
//   - Fail-fast: any unexpected error aborts the run immediately. Rows that
//     are individually unmigratable (empty name, a value with no matching
//     definition) are skipped and reported, not fatal.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"mycorrhizal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	dbPath := flag.String("db", "mycorrhizal.db", "path to the SQLite database file")
	write := flag.Bool("write", false, "actually persist changes (default: dry run, report only, zero writes)")
	force := flag.Bool("force", false, "reprocess field values even if already migrated (definitions are never reprocessed, see migrate.go)")
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

	var users []models.User
	if err := db.Find(&users).Error; err != nil {
		log.Fatalf("failed to load users: %v", err)
	}

	fmt.Printf("backfill-custom-fields: mode=%s db=%s users=%d\n\n", mode, *dbPath, len(users))

	fmt.Println("-- pass 1: User.CustomFieldNames -> FieldDefinition --")
	var defsCreated, defsSkipped int
	for _, user := range users {
		for _, name := range user.CustomFieldNames {
			outcome, err := migrateUserFieldDefinitions(db, user.ID, name, *write)
			if err != nil {
				log.Fatalf("failed to migrate field definition user_id=%d name=%q: %v", user.ID, name, err)
			}
			if outcome.Skipped != "" {
				defsSkipped++
				fmt.Printf("SKIP  user_id=%d name=%q reason=%q\n", user.ID, name, outcome.Skipped)
				continue
			}
			defsCreated++
			action := "PLAN "
			if *write {
				action = "WRITE"
			}
			fmt.Printf("%s user_id=%d name=%q definition_id=%q\n", action, user.ID, name, outcome.DefinitionID)
		}
	}
	fmt.Printf("pass 1 complete: %d created, %d skipped\n\n", defsCreated, defsSkipped)

	var contacts []models.Contact
	if err := db.Find(&contacts).Error; err != nil {
		log.Fatalf("failed to load contacts: %v", err)
	}

	fmt.Println("-- pass 2: Contact.CustomFields -> FieldValue --")
	var valsCreated, valsSkipped int
	for _, contact := range contacts {
		for key, value := range contact.CustomFields {
			outcome, err := migrateContactFieldValue(db, contact, key, value, *write, *force)
			if err != nil {
				log.Fatalf("failed to migrate field value contact_id=%d key=%q: %v", contact.ID, key, err)
			}
			if outcome.Skipped != "" {
				valsSkipped++
				fmt.Printf("SKIP  contact_id=%d key=%q reason=%q\n", contact.ID, key, outcome.Skipped)
				continue
			}
			valsCreated++
			action := "PLAN "
			if *write {
				action = "WRITE"
			}
			fmt.Printf("%s contact_id=%d key=%q value_id=%d\n", action, contact.ID, key, outcome.ValueID)
		}
	}
	fmt.Printf("pass 2 complete: %d created, %d skipped\n\n", valsCreated, valsSkipped)

	fmt.Printf("%s complete: %d definitions, %d values\n", mode, defsCreated, valsCreated)
	if !*write {
		fmt.Println("No database changes were made (dry run). Re-run with -write to persist.")
	}

	os.Exit(0)
}
