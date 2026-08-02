# T20a — Preferences: migrate `Contact.FoodPreference`, project `hobby`

| | |
|---|---|
| **Rating** | 3 — nice to have, likely used |
| **Size** | M |
| **Depends on** | — |
| **Alpha** | **before** — migrates a live, populated field |
| **Source** | WP-95 (split), `92.6`, `91.9` |

## Why this exists — and why it is pre-alpha

WP-95 bundles "Preferences" with "Gift tracking". They are split because they have **opposite** alpha
risk: gifts ([T20b](28-T20b-gift-tracking.md)) is a brand-new entity and safe to defer, while Preferences
**migrates `Contact.FoodPreference`** — a live, populated, user-visible field.

That field spans ~13 files: the model, `contactmodel`'s envelope, `contact_record.go` and
`contact_record_reverse.go`, `import_service.go`, the CSV export, and five frontend files
(`contactFields.ts`, `ContactInformation.tsx`, `AddContactDialog.tsx`, `ContactDetailPage.tsx`,
`api/contacts.ts`).

Structurally this is **the same migration** as `Contact.Circles` → Circle/Tag, which is already pre-alpha
as [T2](05-T2-circle-tag-triage.md). Doing one before alpha and the other after would be incoherent.

## What exists today

- `Contact.FoodPreference string` — a free-text field, fully wired end to end.
- `91.9` defines the target Preferences shape — **read it before designing anything**.
- No Preferences entity exists yet.
- `Card.PersonalInfo` exists in the neutral model and is the projection target for `hobby`. Note
  `RecordForContact` already has projection helpers to copy from: `projectRelationshipEdges`,
  `projectTags`, `projectCustomFields` in `models/contact_record.go` are structurally identical to what
  you need.

## What to build

1. **Preferences entity + migration** per `91.9`. Hand-written SQL up/down pair; follow `Circle`/`Tag`'s
   template (UUID PK, `UserID`, keyed to the contact by `VCardUID`).
2. **Migrate the existing data** — a backfill turning each non-empty `Contact.FoodPreference` into a
   Preferences row. Follow `cmd/backfill-custom-fields`' template exactly: dry-run by default, `-write`
   to apply, idempotent, fail-fast. **Run it for real** as part of this ticket.
3. **`hobby` → `Card.PersonalInfo` projection** — a new `projectPreferences` in `contact_record.go`
   alongside the existing three. Respect `91.13` sensitivity **in the query**, the way
   `projectCustomFields` filters `sensitivity='normal'`.
4. **CRUD + frontend** — replace the free-text food-preference input with the structured equivalent
   across the five frontend files above.
5. **Retire `Contact.FoodPreference`** once nothing reads it: model field, DTOs, CSV export column,
   `import_service.go` mapping, and the frontend field registry.

## Traps

- The CSV export has a `"Food Preference"` column — dropping the field changes the export format.
  Decide and note it.
- `import_service.go` maps incoming columns onto `FoodPreference`; that mapping needs a new destination.
- Do not set `Card`/`CRM` by direct field mutation — use `ApplyRecordToContact` so `BeforeSave` runs.
- A dry run against a fresh DB cannot show pass-2 successes if you split the backfill into passes — this
  is documented, expected behaviour from `cmd/backfill-custom-fields`, not a bug.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- A **real-DB test** proves: a seeded `FoodPreference` migrates to a Preferences row; a `hobby`
  preference appears in `RecordForContact(...).Card.PersonalInfo`; a non-`normal` sensitivity preference
  does **not** project.
- The backfill has actually been run (dry-run → `-write` → re-run showing zero further writes).
- `npx tsc --noEmit` clean, `npx vitest run` green; verified in a real browser.
## Flash implementation notes

### Files to read first
- CLAUDE.md (repo conventions, traps, commands)
- Look at an existing implemented ticket (e.g. T5/LifeEvent) for the full pattern: model → controller → routes → api → hooks → dialog → list → ContactInformation wiring → ContactDetailPage wiring → i18n
- For households: study `circle_controller.go` and `circle_controller_test.go` — the household controller must follow this exact idiom

### Tests you must write before considering it done
- Backend: real-DB test (`database.InitDB`, not `AutoMigrate`) for the core round-trip
- Backend: controller tests covering create, update, delete, cross-user ownership rejection, 409 on duplicate member add
- Frontend: component test for the dialog and list (follow `MergeContactsDialog.test.tsx` pattern — `afterEach(cleanup)`, mock `fetch` with `vi.stubGlobal`)
- Frontend: the ticket's specific assertion (e.g. \"suggested edges appear in RelationshipEdgeList\")

### Self-verification checklist
1. `npx tsc --noEmit` clean
2. `npx vitest run` green (ALL tests, not just yours)
3. `cd backend && go build ./... && go vet ./... && gofmt -l . && go test ./...` green
4. Run `make migrate-up` to verify migrations apply cleanly
5. Hand-verify: break one assertion, confirm the test fails, restore

### Common traps (beyond CLAUDE.md)
- `gorm.Model` only works on uint-PK entities — UUID PK models need explicit `ID`/`CreatedAt`/`UpdatedAt`/`DeletedAt`
- Membership is keyed by `Contact.VCardUID`, not numeric ID — use `gorm:\"column:member_vcard_uid\"` tag or GORM derives `member_v_card_uid`
- Backend tests use `setupRouter()` from `activity_controller_test.go` (sets db + userID + cfg in context, uses AutoMigrate)
- Frontend component tests: `afterEach(cleanup)` is mandatory; MUI appends `\" *\"` to required field labels
- All 5 locale files (`de/es/fr/it/en`) need real translations, not English placeholders

### Ticket-specific
- The backfill tool pattern: copy `cmd/backfill-custom-fields/main.go` exactly — dry-run by default, `-write` to apply, idempotent, fail-fast
- Run it for real: `go run ./cmd/backfill-preferences` dry → verify output → `-write` → re-run → verify zero further writes
- `projectPreferences` in `models/contact_record.go`: follow `projectCustomFields`'s exact shape — sensitivity filter in the query, not in the caller
- CSV export change: `export_controller.go` currently writes `strings.Join(contact.Circles, "; ")` for circles and has a `"Food Preference"` column — you're replacing the column with preference data
- The 13 files that consume `FoodPreference` are listed in the ticket — grep for each one and update it
- `import_service.go` maps `"food:food_preference"` and `"dietary:food_preference"` — update these mappings
