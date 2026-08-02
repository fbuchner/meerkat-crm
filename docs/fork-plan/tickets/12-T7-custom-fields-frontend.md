# T7 — Custom fields v2: frontend, backfill run, retire v1

| | |
|---|---|
| **Rating** | 3 — nice to have, likely used |
| **Size** | L |
| **Depends on** | [T6](11-T6-custom-fields-api.md) |
| **Alpha** | before |
| **Source** | WP-84b, `94.6` |

## Why this exists

v1 is what actually runs today; v2 is the unreachable parallel implementation. Leaving both is exactly
the "layer a new representation on top, bridge them, defer removal" pattern that
[T22](19-T22-legacy-audit.md) exists to clean up — so finish the cutover rather than adding to the pile.

## What exists today

**v1, live**, with these consumers:
- `backend`: `User.CustomFieldNames []string`, `Contact.CustomFields map[string]string`, and the CSV
  export (which appends custom field names as extra header columns and neutralises them against formula
  injection — see `csvSafeRecord`).
- `frontend`: `components/CustomFieldsSettings.tsx`, `components/ContactInformation.tsx`,
  `components/AddContactDialog.tsx`, `ContactsPage.tsx`, `api/users.ts`, `api/contacts.ts`.

**v2**: model, validation, projection, `cmd/backfill-custom-fields`, and (after T6) the API.

## What to build

1. **`frontend/src/api/fieldDefinitions.ts`** (+ values) — model on `api/relationshipEdges.ts`.
2. **Per-type input rendering.** v2 is typed, so the UI must render per `FieldDefinition.Type` (text,
   number, date, enum, email, phone, url, …) rather than v1's single text box, and handle
   `FieldConstraints.Multi` as an add/remove list. Reuse `MultiValueField.tsx`'s existing add/remove row
   pattern rather than inventing one.
3. **Replace `CustomFieldsSettings.tsx`** with definition management (create/edit/delete a definition,
   set type, constraints, sensitivity, and standards projection target).
4. **Replace the value editors** in `ContactInformation.tsx`, `AddContactDialog.tsx`, and the
   `ContactsPage.tsx` display.
5. **Run `cmd/backfill-custom-fields` for real** — dry run first, then `-write`, then re-run to confirm
   zero further writes.
6. **Retire v1**: `User.CustomFieldNames`, `Contact.CustomFields`, their DTOs, the
   `/users/custom-fields` routes, the frontend consumers, and the CSV export path — replaced by the v2
   equivalent, not just deleted (the export must still emit custom fields).
7. **i18n** in all five locale files.

## Traps

- **The backfill's dry-run has a documented two-pass limitation**: it runs definitions first, then values,
  so a dry run against a fresh DB reports "no field definition found" for every value, because pass 1 made
  no real writes to look up. That is expected, not a bug. `-force` exists only for the values pass.
- The CSV export's custom-field **header row** is user-authored and is deliberately run through
  `csvSafe` — preserve that when you rewrite the export path, or you reintroduce the formula-injection
  finding Tier 1 fixed.
- Sensitivity must be honoured in the UI too, not just the API — a `secret` field should not render in a
  context that gets exported or shared.
- Component tests need explicit `afterEach(cleanup)`; MUI appends `" *"` to required labels.

## Done when

- `npx tsc --noEmit` clean, `npx vitest run` green, with component tests for at least the enum and
  `Multi` rendering paths.
- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green after v1 removal.
- The backfill has actually been run and re-run, with output recorded in the commit message.
- Verified in a real browser: define a typed field, set a value on a contact, confirm validation rejects a
  bad value, confirm it appears in CSV export, confirm a `secret` one does not project to vCard.
- `grep -rn "CustomFieldNames\|custom_fields" backend frontend/src` returns only v2 references.
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
- `CustomFieldsSettings.tsx` is the v1 settings page — you're replacing it entirely, not modifying it
- Per-type rendering: the dialog must render different inputs for text/number/date/enum/email/phone/url. Study `MultiValueField.tsx` for the add/remove row pattern (it already handles typed lists)
- The backfill: `cmd/backfill-custom-fields` has a documented two-pass limitation — dry run shows "no definition found" for values because definitions are created in pass 1. That's expected, not a bug. `-force` exists for the values pass.
- CSV export: v1 appends custom field names as extra header columns. v2 must do the same but from FieldDefinitions. Preserve `csvSafeRecord` (formula injection neutralization).
- Sensitivity in UI: a `secret` field should not render in contexts that get exported/shared
- v1 retirement checklist: `User.CustomFieldNames`, `Contact.CustomFields`, their DTOs, `/users/custom-fields` routes, all frontend consumers, CSV export path
- `grep -rn "CustomFieldNames|custom_fields" backend frontend/src` must return only v2 references when done
