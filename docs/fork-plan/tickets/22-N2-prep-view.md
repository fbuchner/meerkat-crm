# N2 — Prep view / person briefing screen

| | |
|---|---|
| **Rating** | **5 — practically necessary.** The difference between a database and a relationship OS |
| **Size** | M — read-side composition, almost no new persistence |
| **Depends on** | [T5](03-T5-lifeevent-frontend.md), [T19](20-T19-cadence.md), [T21](21-T21-conversation-agenda.md) |
| **Alpha** | after (by dependency, not by value) |
| **Source** | New (gap found in the 2026-07-30 product review) |

## Why this exists

Every ingredient is planned and **nothing assembles them**. There is no screen answering the question you
actually have five minutes before seeing someone: *what do I need to remember about this person right
now?*

That gap is structural — each ticket built its own slice and none owned the composition. This ticket is
the composition.

## What it shows

One screen, scannable in under a minute:

| Block | Source |
|---|---|
| Last interaction: when, what type, what was discussed | `Activity` timeline + notes |
| **How overdue this relationship is** | [T19](20-T19-cadence.md)'s derived health |
| **Open agenda items** — things to bring up | [T21](21-T21-conversation-agenda.md) |
| Household and close relationships — partner's and kids' names | `RelationshipEdge` + `Household` |
| Recent and upcoming life events | [T5](03-T5-lifeevent-frontend.md) |
| Upcoming dates — birthday, anniversary | existing birthday service |
| Food preferences, key custom fields | [T20a](10-T20a-preferences.md), custom fields v2 |

## What to build

1. **A composition endpoint or a composed frontend fetch.** Prefer a single backend endpoint
   (`GET /contacts/:id/briefing`) over six frontend round trips — this screen is latency-sensitive by
   nature, and a mobile client would want the same thing.
2. **The surface.** Either a distinct tab on the contact page or a dedicated "prep" mode. A tab is less
   work and more discoverable; a dedicated mode can be denser. Pick one.
3. **Graceful degradation.** Every block must render sensibly when its source is empty or its feature is
   not yet built. This is what lets a reduced version ship as soon as T19 lands, before T21.
4. **Optional: an "upcoming" list** — everyone you are due to see soon, each with their briefing one click
   away. That turns this from a lookup into a workflow, and is arguably where the real value is.

## Traps

- **Do not persist a briefing.** Everything here is derived; a cached briefing is a staleness bug waiting
  to happen. If it is slow, fix the queries or add a request-scoped cache, not a table.
- Relationship labels must respect direction — use `getEffectiveType`/`getDisplayLabel` from
  `api/relationshipEdges.ts`, which already handle the inverse-token logic. Do not re-derive it.
- Only `status: confirmed` edges belong here. A suggested edge is not fact.
- Respect `91.13` sensitivity: a `secret` relationship or field should not surface on a screen likely to
  be open in front of the person it concerns.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- A test proves the briefing composes correctly and that each block degrades cleanly when empty.
- `npx tsc --noEmit` clean, `npx vitest run` green.
- Verified in a real browser against a contact with a full picture — relationships, life events, agenda
  items, an overdue cadence — and against a nearly-empty contact.

## Note

Sized M despite rating 5 because it is composition over entities that already exist. If it is growing new
tables, the scope has drifted.

### Ticket-specific
- "Everything you want to know about someone before you see them" — composite view of timeline, relationships, life events, notes, cadence health, upcoming reminders
- This is a READ-ONLY aggregation page — no writes, no new backend entities (it reads from existing ones)
- Backend: a single new endpoint that aggregates data. Follow the graph controller pattern (`controllers/graph_controller.go`) for the aggregation query approach.
- The view should include: recent timeline items, active relationships, life events, recent notes, upcoming reminders, cadence health if T19 is done
- Frontend: a new page component (`PrepViewPage.tsx`) navigable from contact detail (button: "Prep View" / "Briefing")
- No new model needed — this is a query-only endpoint

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
