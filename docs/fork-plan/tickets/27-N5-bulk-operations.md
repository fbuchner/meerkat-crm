# N5 — Bulk operations

| | |
|---|---|
| **Rating** | 3 |
| **Size** | M |
| **Depends on** | [T4](07-T4-circle-tag-frontend.md) — so it operates on real Circle/Tag entities |
| **Alpha** | after |
| **Source** | New (gap found in the 2026-07-30 product review) |

## Why this exists

Once the contact list runs to hundreds, per-contact editing stops scaling. Assigning fifty contacts to a
circle one at a time is the kind of friction that makes people stop maintaining their data.

## What exists today

- `frontend/src/types/api.ts` declares a `BatchOperationResponse` type — **check whether anything
  actually implements it.** It may be aspirational scaffolding inherited from Meerkat, in which case it
  belongs in [T22](19-T22-legacy-audit.md)'s dead-code sweep rather than being built around.
- `ContactsPage.tsx` — the list, with search, circle filter, sort, and pagination. No multi-select.
- Circle/Tag membership endpoints (`POST/DELETE /circles/:id/members`, `/tags/:id/contacts`) already
  exist and are per-contact.
- `ArchiveContact`/`UnarchiveContact` and `DeleteContact` exist per-contact.

## What to build

1. **Multi-select on the contacts list** — checkbox column, select-all-on-page, a clear selection count,
   and a visible way to clear it.
2. **Bulk actions**: add/remove circle, add/remove tag, archive/unarchive, delete.
3. **Backend**: either a batch endpoint per action, or loop the existing per-contact endpoints from the
   frontend. **Prefer a batch endpoint** — a 200-contact loop is 200 round trips and 200 chances to
   half-fail. If you build one, give it partial-success semantics (report which succeeded and which
   failed) rather than all-or-nothing; `import_session.go`'s partial-success handling is the precedent
   here.
4. **Confirmation proportional to blast radius.** Bulk delete needs a real confirmation naming the count;
   bulk tag does not.

## Traps

- **Bulk delete is the most dangerous action in the app.** `DeleteContact` cascades across ~14 tables.
  Doing it 200 times must be transactional per contact and must not leave the DB half-cleaned. Consider
  restricting bulk delete to archive-only, and requiring individual deletes for the real thing.
- Selection must survive pagination, or "select all" means "select this page" and will surprise someone
  destructively.
- Membership is keyed by `Contact.VCardUID`, not the numeric ID.
- A duplicate membership add returns `409` — in bulk that is success, not failure. Do not surface it as
  an error.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- If a batch endpoint exists: tests cover partial success, ownership scoping (a contact belonging to
  another user is rejected, not silently skipped), and duplicate-add tolerance.
- `npx tsc --noEmit` clean, `npx vitest run` green, with a component test for selection state across
  pagination.
- Verified in a real browser: select across two pages, bulk-add a tag, confirm every selected contact got
  it and nothing else did.

### Post-alpha note
This ticket is post-alpha — real production data exists. Changes that modify schemas or data must be additive and non-destructive. Migration files must be hand-written SQL up/down pairs. Test against `database.InitDB`, not `AutoMigrate`.

## Flash implementation notes

### Files to read first
- `/CLAUDE.md` at repo root (conventions, recurring traps, commands)
- Study an existing fully-implemented feature for the pattern: model → controller → routes → api → hooks → dialog → list → page wiring → i18n
- Common pattern references: `circle_controller.go` + test (newer idiom), `api/relationshipEdges.ts` + hook, `RelationshipEdgeDialog.tsx` + test, the `ContactInformation.tsx` tab + `ContactDetailPage.tsx` wiring

### Tests you must write before considering it done
- Backend: controller tests covering CRUD, ownership scoping, error states (not found, cross-user, 409 duplicate)
- Backend: real-DB test (`database.InitDB`, not `AutoMigrate`) for the core round-trip + any migration-dependent behavior
- Frontend: component test (`afterEach(cleanup)`, mock `fetch` with `vi.stubGlobal`) for dialog and list
- Hand-verify EVERY new test: break the code, confirm the test fails, restore. A test that has never failed has proven nothing.

### Self-verification checklist
1. `npx tsc --noEmit` — clean
2. `npx vitest run` — green (ALL tests, not just yours)
3. `cd backend && go build ./... && go vet ./... && gofmt -l . && go test ./...` — green
4. New migrations: run `make migrate-up` to verify they apply cleanly
5. All 5 locale files (`de/es/fr/it/en`) — real translations for any new strings

### Common traps (beyond CLAUDE.md)
- `gorm.Model` only works on uint-PK entities — UUID PK models need explicit `ID`/`CreatedAt`/`UpdatedAt` fields
- Backend tests use `setupRouter()` from `activity_controller_test.go` (sets `db`, `userID`, `cfg` in Gin context, uses AutoMigrate)
- Frontend component tests: `afterEach(cleanup)` mandatory; MUI appends `" *"` to required field `getByLabelText`
- Migration files: hand-written SQL up/down pairs — never add a column by editing the struct alone
- `gorm:"column:xxx"` tag is mandatory for acronyms/compound words — GORM silently derives wrong names
- New entities: decide soft vs hard delete per T26's rule (user-authored content → soft, edge/join rows → hard)
- Delete cascade: add new entities to `deleteContactAssociations` in `contact_controller.go` and `DeleteUser` in `admin_user_controller.go`
