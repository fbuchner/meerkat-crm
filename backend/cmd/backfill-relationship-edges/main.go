// Command backfill-relationship-edges is the one-shot WP-81 data migration
// (docs/fork-plan/92-delivery-roadmap.md): migrate every legacy
// models.Relationship row (backend/models/relationship.go) into the
// relationship graph (models.RelationshipEdge, built in WP-80), promoting
// name-only rows (no RelatedContactID) into new thin Contacts along the way
// — docs/fork-plan/90-vision-and-reconciliation.md D3's explicit migration
// consequence: "no edge is ever left referencing a bare string."
//
// This does NOT touch the legacy table, its API (controllers/
// relationship_controller.go), or any of its other consumers
// (graph_controller.go, birthday_service.go, export_controller.go, cascade
// deletes) — they keep reading models.Relationship exactly as before.
// Cutting them over to the graph is a separate, later step.
//
// Same safety discipline as cmd/backfill-contact-records (WP-70's backfill,
// the explicitly-cited precedent):
//   - Dry-run by default. Without -write, this command performs every read
//     and resolution step (so the report line reflects a real run exactly)
//     but makes zero database writes.
//   - Idempotent. Rows already migrated (tracked via
//     RelationshipEdge.LegacyRelationshipID) are skipped, unless -force is
//     passed.
//   - Fail-fast: any unexpected error aborts the run immediately. Rows that
//     are individually unmigratable (self-reference, empty name, a dangling
//     related_contact_id pointer) are skipped and reported, not fatal.
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
	force := flag.Bool("force", false, "reprocess relationships even if already migrated")
	flag.Parse()

	db, err := gorm.Open(sqlite.Open(*dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("failed to open database %q: %v", *dbPath, err)
	}

	var legacyRelationships []models.Relationship
	if err := db.Find(&legacyRelationships).Error; err != nil {
		log.Fatalf("failed to load legacy relationships: %v", err)
	}

	mode := "DRY RUN"
	if *write {
		mode = "WRITE"
	}
	fmt.Printf("backfill-relationship-edges: mode=%s force=%v db=%s relationships=%d\n\n", mode, *force, *dbPath, len(legacyRelationships))

	var migrated, skipped int
	for _, legacy := range legacyRelationships {
		outcome, err := migrateOneRelationship(db, legacy, *write, *force)
		if err != nil {
			log.Fatalf("failed to migrate relationship id=%d: %v", legacy.ID, err)
		}

		if outcome.Skipped != "" {
			skipped++
			relatedContactID := "none (name-only)"
			if legacy.RelatedContactID != nil {
				relatedContactID = fmt.Sprintf("%d", *legacy.RelatedContactID)
			}
			fmt.Printf("SKIP  id=%d contact_id=%d related_contact_id=%s reason=%q\n",
				legacy.ID, legacy.ContactID, relatedContactID, outcome.Skipped)
			continue
		}

		migrated++
		action := "PLAN "
		if *write {
			action = "WRITE"
		}

		target := outcome.SourceID
		if outcome.CreatedThinContact {
			if *write {
				target = fmt.Sprintf("new thin contact %q (vcard_uid=%s)", outcome.ThinContactName, outcome.SourceID)
			} else {
				target = fmt.Sprintf("new thin contact %q (not yet created)", outcome.ThinContactName)
			}
		}

		typeNote := outcome.MatchedType
		if outcome.UsedFallback {
			typeNote = fmt.Sprintf("%s (fallback — legacy type %q unmatched)", outcome.MatchedType, legacy.Type)
		}

		fmt.Printf("%s id=%d source=%s target=%s type=%s edge_id=%q\n",
			action, legacy.ID, target, outcome.TargetID, typeNote, outcome.EdgeID)
	}

	fmt.Printf("\n%s complete: %d migrated, %d skipped, %d total\n", mode, migrated, skipped, len(legacyRelationships))
	if !*write {
		fmt.Println("No database changes were made (dry run). Re-run with -write to persist.")
	}

	os.Exit(0)
}
