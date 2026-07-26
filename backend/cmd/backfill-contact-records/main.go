// Command backfill-contact-records is the one-shot P1 *data* migration for
// WP-70 (docs/fork-plan/50-integration-and-rebrand.md): populate each
// existing Contact row's new card/crm/passthrough/fn/org columns (added by
// database/migrations/000023_neutral_contact_model.up.sql) from that row's
// current legacy flat/array fields, using the exact same
// models.RecordFromContact mapping that Contact.BeforeSave now applies to
// every future save — there is one Contact->Record mapping in this codebase,
// not two.
//
// Deliberately named differently from cmd/migrate (the golang-migrate
// *schema*-migration CLI runner already in this repo): this command is the
// *data* transform, meant to be run once after the 000023 schema migration
// has applied.
//
// Safety (binding, per WP-70's risk/guard note — the data migration is the
// irreversible part):
//   - Dry-run by default. Without -write, this command only reports what it
//     would change; it makes zero database writes.
//   - Idempotent. Rows whose `card` column is already populated (i.e.
//     already migrated) are skipped, unless -force is passed.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"

	"meerkat/contactmodel"
	"meerkat/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	dbPath := flag.String("db", "meerkat.db", "path to the SQLite database file")
	write := flag.Bool("write", false, "actually persist changes (default: dry run, report only, zero writes)")
	force := flag.Bool("force", false, "reprocess rows even if their card column is already populated")
	flag.Parse()

	db, err := gorm.Open(sqlite.Open(*dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("failed to open database %q: %v", *dbPath, err)
	}

	var contacts []models.Contact
	if err := db.Find(&contacts).Error; err != nil {
		log.Fatalf("failed to load contacts: %v", err)
	}

	mode := "DRY RUN"
	if *write {
		mode = "WRITE"
	}
	fmt.Printf("backfill-contact-records: mode=%s force=%v db=%s contacts=%d\n\n", mode, *force, *dbPath, len(contacts))

	var processed, skipped int
	for i := range contacts {
		c := &contacts[i]

		if !isZeroCard(c.Card) && !*force {
			skipped++
			fmt.Printf("SKIP  id=%d vcard_uid=%s (card already populated; pass -force to reprocess)\n", c.ID, c.VCardUID)
			continue
		}

		record := models.RecordFromContact(c)
		proj := contactmodel.DeriveProjection(record)

		action := "PLAN "
		if *write {
			action = "WRITE"
		}
		fmt.Printf("%s id=%d vcard_uid=%s firstname=%q lastname=%q fn=%q org=%q email=%q phone=%q birthday=%q emails=%d phones=%d addresses=%d impp=%d links=%d anniversaries=%d passthrough_vcard=%d\n",
			action, c.ID, c.VCardUID,
			proj.Firstname, proj.Lastname, proj.FN, proj.Org,
			proj.PrimaryEmail, proj.PrimaryPhone, proj.Birthday,
			len(record.Card.Emails), len(record.Card.Phones), len(record.Card.Addresses),
			len(record.Card.ImppAddresses), len(record.Card.Links), len(record.Card.Anniversaries),
			len(record.Passthrough.VCard),
		)

		processed++

		if !*write {
			continue
		}

		c.Card = record.Card
		c.CRM = record.Envelope
		c.Passthrough = record.Passthrough
		c.FN = proj.FN
		c.Org = proj.Org
		// Same "only overwrite when there's something to sync" guard as
		// Contact.BeforeSave (see models/contact.go) — keeps this backfill's
		// write path behaviorally identical to what a normal save would do.
		if proj.Firstname != "" {
			c.Firstname = proj.Firstname
		}
		if proj.Lastname != "" {
			c.Lastname = proj.Lastname
		}
		if proj.PrimaryEmail != "" {
			c.Email = proj.PrimaryEmail
		}
		if proj.PrimaryPhone != "" {
			c.Phone = proj.PrimaryPhone
		}
		if proj.Birthday != "" {
			c.Birthday = proj.Birthday
		}

		// SkipHooks: BeforeSave would recompute the exact same
		// Card/CRM/Passthrough/projection we just computed (harmless but
		// redundant), and AfterSave would bump ETag for every historical row
		// even though nothing about its sync state actually changed. Saving
		// through the normal struct-based path (not a raw map) still applies
		// each field's GORM serializer correctly.
		if err := db.Session(&gorm.Session{SkipHooks: true}).Save(c).Error; err != nil {
			log.Fatalf("failed to save contact id=%d: %v", c.ID, err)
		}
	}

	fmt.Printf("\n%s complete: %d processed, %d skipped (already populated), %d total\n", mode, processed, skipped, len(contacts))
	if !*write {
		fmt.Println("No database changes were made (dry run). Re-run with -write to persist.")
	}

	os.Exit(0)
}

// isZeroCard reports whether c is the zero-value Card, i.e. nothing has
// been mapped into it yet (contacts row not yet migrated). NULL, "", and
// "{}" in the `card` column all deserialize to this same zero value, so all
// three are treated identically as "not yet populated".
func isZeroCard(c contactmodel.Card) bool {
	return reflect.DeepEqual(c, contactmodel.Card{})
}
