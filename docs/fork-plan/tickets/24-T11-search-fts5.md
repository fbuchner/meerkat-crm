# T11 — WP-86 search synonyms, household scope, FTS5 full-text

| | |
|---|---|
| **Rating** | **5 — practically necessary at scale** |
| **Size** | L |
| **Depends on** | [T10](23-T10-graph-traversal.md), [T1](09-T1-households.md) |
| **Alpha** | after — deliberately, see below |
| **Source** | `92.2` |

## Why it is rated 5 but still post-alpha

Today's search is `LIKE`-based over contact fields only (`applyContactSearch` in
`contact_controller.go`) and **does not search notes at all**. Once you have a few hundred contacts and
years of notes, "where did I write that thing" becomes a daily need — this is the #1 retrieval feature.

But its value **scales with data volume**, and alpha will not have volume. Searching a few weeks of notes
is not the problem; searching five years of them is. That is why deferring search is principled while
deferring [T19](20-T19-cadence.md) (whose value is immediate) is not.

## What to build

Three distinct halves — they can ship separately:

1. **FTS5 full-text over contacts, notes, and interactions.**
   No FTS5 exists anywhere in this codebase today (verified). This means: `CREATE VIRTUAL TABLE … USING
   fts5(…)`, triggers to keep it in sync with the base tables on insert/update/delete, and a one-time
   index backfill. All of it is **derived data** — rebuildable from source at any time, which is exactly
   why it is safe to add post-alpha: a rebuild is a re-runnable index job, not a destructive migration.
2. **Search synonyms/aliases from the type registry** — `mom`/`mother` → `parent_of`.
   `models/relationship_type_registry.go` already carries a `Synonyms` field, extended during WP-81 and
   currently with **no live consumer**. This is what it was built for.
3. **Household-scoped queries** — "everyone in the Smith household". Genuinely depends on
   [T1](09-T1-households.md); households are unreachable until then, so this half has nothing to query.

## Traps

- **FTS5 triggers and soft deletes.** Most tables here use `gorm.Model`'s soft delete, so a "deleted" row
  is still present. Your triggers and queries must respect `deleted_at IS NULL`, or search returns
  deleted contacts and notes.
- **Sensitivity (`91.13`).** A `secret` relationship, tag, or custom field must not be findable by
  full-text search. Filter in the query, consistent with how `projectCustomFields` already does it.
- **User scoping.** FTS tables have no natural `user_id` join — carry it in the indexed row or join back
  to the base table, and make certain a search cannot cross users. This is the highest-risk correctness
  issue in the ticket; test it explicitly.
- Rebuilding the index must be an available operation, not something that only happens at migration time
  — you will want it after any bulk import.
- SQLite build must actually include FTS5. Verify against the `glebarez/sqlite` driver in use before
  designing around it.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- Tests prove: a note's body is findable; a synonym query resolves through the registry; a household
  query returns its members; a **cross-user** search returns nothing; a soft-deleted row is not findable;
  a `secret`-sensitivity item is not findable.
- Hand-verified: remove the user-scoping condition, confirm the cross-user test fails, restore.
- Index rebuild proven idempotent and re-runnable against a populated DB.
- Frontend search surfaces notes and interactions, not just contacts — otherwise the feature is invisible.

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
