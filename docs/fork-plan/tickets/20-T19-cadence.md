# T19 — WP-94 CadencePolicy + derived relationship health

| | |
|---|---|
| **Rating** | **5 — practically necessary.** This is the product's reason to exist |
| **Size** | L |
| **Depends on** | [T5](03-T5-lifeevent-frontend.md) (timeline) |
| **Alpha** | after by the risk rules — **but see the note below** |
| **Source** | `92.6`, `91.10` (has a real field table — read it) |

## Why this matters more than its position suggests

"Who have I not talked to in too long" is the entire reason personal-relationship CRMs exist. Everything
else in this model is scaffolding for this question.

It sits post-alpha only because the alpha line was drawn by *risk* and this ticket is purely additive.
`95-backlog-and-priorities.md` records a standing recommendation to **pull it before alpha**: its value is
immediate on day one of real use, and its only dependency already precedes alpha. Check whether that
decision has been made before assuming this is post-alpha work.

## The spec (`91.10`)

| Field | Notes |
|---|---|
| `id` | UUID |
| `entity_id` (or `relationship_id`) | who the cadence is about |
| `target_interval_days` | e.g. 30 |
| `qualifying_types` | which Interaction types reset it (call/visit/meal…); everything else ignored |

**Derived, never stored:** relationship health (`next_due`, `overdue_by`) is *computed* from the timeline
— find the most recent **qualifying** interaction, add the interval. Do not persist it.

**The rule that is easy to get wrong:** *recording a qualifying interaction resets cadence; completing a
generated task does not.* The vision doc is explicit — the meaningful interaction is the source of truth.
Overdue cadence **may** emit a task to an external manager (Vikunja) via the existing webhook mechanism,
but the CRM owns the cadence *state*, and that task's completion is not a reset signal.

## What exists today

- **`Activity.Qualifying()`** (`backend/models/activity.go`) — added in WP-84 for exactly this and
  **has had no consumer since**. Start there; do not write a second definition of "qualifying."
- `Activity.Type` with the `InteractionType*` constants (`call`, `video_call`, `visit`, `meal`, `gift`,
  `photo`, `message`, `shared_activity`).
- The webhook system (`services/webhook_service.go`) with signing, retry, and delivery — the
  Vikunja emission path needs no new infrastructure.
- `services/reminder_service.go`'s job-lock pattern (`acquireJobLock`/`releaseJobLock` with a
  `minInterval`) for anything that runs on a cron tick.

## What to build

1. **`CadencePolicy` entity + migration** per the table above. UUID PK, follow `Circle`/`LifeEvent`'s
   template. `qualifying_types` is a JSON array column (see `RelationshipEdge.Metadata` for the
   serializer pattern).
2. **Derived health computation** — a service that, for a contact, finds the most recent qualifying
   interaction and returns `next_due` / `overdue_by`. Pure function over the timeline; no writes.
3. **CRUD + surfacing** — set a cadence on a contact; show health on the contact page and as an
   "overdue" list (which is the screen people will actually live in).
4. **Overdue → webhook** emission, respecting the job-lock pattern so a multi-instance deploy does not
   double-fire.
5. **Frontend** — cadence editor, health indicator, overdue view.

## Traps

- Do **not** reset cadence on task completion. It is the single most likely misimplementation here.
- A contact with no qualifying interaction ever has no "last" — decide whether cadence counts from the
  contact's creation date or is simply undefined until the first interaction. Say which.
- Health is derived; if you find yourself adding a `next_due` column, stop.
- Timezone: "overdue by N days" near midnight is a classic off-by-one. `DaysUntilBirthday` already solved
  the adjacent problem (including the Dec 31 → Jan 1 wrap) — reuse its approach rather than reinventing.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- Tests prove: a qualifying interaction resets cadence; a **non**-qualifying one does not; task
  completion does not; overdue computation is correct across a year boundary.
- Hand-verified: make a non-qualifying type reset cadence, confirm the test fails, restore.
- A **real-DB test** for the policy round trip.
- Verified in a real browser: set a cadence, record a qualifying interaction, watch health update.

### Ticket-specific
- `Activity.Qualifying()` in `models/activity.go` is the single definition of "qualifying interaction" — use it, don't reimplement
- The InteractionType constants: `call`, `video_call`, `visit`, `meal`, `gift`, `photo`, `message`, `shared_activity` — defined in `models/activity.go`
- `qualifying_types` is a JSON array column — study `RelationshipEdge.Metadata` for the `gorm:"serializer:json"` pattern
- Health is DERIVED, never stored. If you find yourself adding a `next_due` column, stop — you're doing it wrong.
- The single biggest misimplementation risk: resetting cadence on task completion. The rule is: recording a qualifying INTERACTION resets cadence; completing a generated TASK does not.
- Timezone: `DaysUntilBirthday` in `services/birthday_service.go` already solved the adjacent problem (including Dec 31 → Jan 1 wrap) — reuse its approach
- Overdue → webhook emission: follow `services/webhook_service.go`'s pattern. Use the job-lock pattern (`acquireJobLock`/`releaseJobLock` with minInterval) — same as reminders.
- For new entities: decide soft vs hard delete per T26's rule. `CadencePolicy` is user-authored content → soft delete.

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
