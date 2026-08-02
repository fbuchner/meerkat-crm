# T24 — Non-critical test-coverage expansion

| | |
|---|---|
| **Rating** | 2 — rarely user-visible |
| **Size** | M |
| **Depends on** | — |
| **Alpha** | before |
| **Source** | Tier 6, `45-test-coverage-closure.md` |

## Why this exists

`45-test-coverage-closure.md`'s Phase 3 (the security-sensitive and critical-path half) is **fully
closed** — that work landed in Tier 3c items 9 and 10. What was never scoped at all is the low-risk
remainder, which is why it sits here rather than in the security tier.

## The actual numbers (as of 2026-07-29)

| Package | Coverage |
|---|---|
| `config` | 24.2% |
| `database` | 37.8% |
| `routes` | 0.0% |
| `errors` | 0.0% |
| `i18n` | had 0%; now has its first tests from the item-12 fix |
| `logger` | 0.0% |
| `cmd/backfill-custom-fields` | 33.7% |
| `cmd/backfill-contact-records` | 0.0% |
| `cmd/migrate` | 0.0% |

Re-measure before starting — several have moved:

```bash
cd backend && go test ./... -cover
```

## What to build

**This ticket starts with a scoping pass, not with tests.** Decide per package whether coverage is worth
buying, and write the decision down. Genuine candidates to *decline*:

- `cmd/migrate` — a thin CLI wrapper around a library call; testing it in isolation tests nothing real.
- `cmd/backfill-contact-records` and `cmd/backfill-custom-fields` — one-shot tools that have already run.
  [T22](19-T22-legacy-audit.md) may delete them outright, in which case covering them now is waste.
  **Check T22's disposition before writing a line here.**
- `logger` — mostly a zerolog configuration surface.

Likelier to be worth it:

- `config` — env parsing and `ValidateOrPanic`'s rules are real logic with real failure modes, and a
  misparse breaks boot. Highest value in this list.
- `database` — `openDSN`'s pragma handling (WAL + `foreign_keys`) already has targeted tests; the rest is
  migration plumbing.
- `errors` — the `AppError` constructors and HTTP-status mapping are simple but widely relied on.
- `routes` — a registration smoke test has real value now that [T8](16-T8-openapi.md) adds a
  spec/route drift test; consider building them together.

## Explicit instruction

**Do not chase the percentage.** A test that asserts a getter returns what was set is negative value —
it adds maintenance cost and catches nothing. If a package's honest answer is "not worth covering," write
that in the commit message and in `95-backlog-and-priorities.md` and move on. Partial completion with
reasons is the expected outcome here, not a failure.

## Done when

- `go test ./... -cover` re-measured, with before/after numbers recorded.
- Each package in the list has an explicit decision: covered, or declined with a reason.
- Every new test hand-verified — break the code, confirm failure, restore. This matters more than usual
  here, because coverage-driven tests are the easiest kind to write vacuously.

### Ticket-specific
- This ticket STARTS with a scoping pass — decide per package whether coverage is worth buying. Write decisions down.
- Decline candidates: `cmd/migrate`, `cmd/backfill-*` (one-shot tools), `logger` (config surface)
- Likely worth it: `config` (ValidateOrPanic rules), `database` (pragma handling), `errors` (already done), `routes` (registration smoke test)
- Run `cd backend && go test ./... -cover` before and after — record numbers
- Every new test must be hand-verified: break the code, confirm test fails, restore. Coverage-driven tests are the easiest to write vacuously.
- Partial completion with explicit reasons is EXPECTED, not a failure

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
