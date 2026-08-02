# T1 — Household CRUD + suggestion trigger + review wiring

| | |
|---|---|
| **Rating** | 3 — nice to have, likely used |
| **Size** | M |
| **Depends on** | — |
| **Alpha** | before |
| **Source** | WP-83, §3d WP3, `91.3`/`91.4` |

## Why this exists

`Household` is fully built on the backend and **completely unreachable**: no routes, no frontend, and
`services.GenerateHouseholdSuggestions` has **zero callers anywhere**.

The consequence is worth understanding: §3d WP3 shipped a suggestion-review UI (Accept/Reject on
`status: suggested` relationship edges) that **can never fire**, because nothing in a running app can
produce a suggested edge. This ticket is the missing connective tissue between three pieces that all
already exist and are tested.

It is also the project's first live use of the propose-then-approve pattern that `92.7` says the eventual
AI layer must reuse.

## What exists today

- `models.Household` + `models.HouseholdMember` (`backend/models/household.go`) — UUID PKs; members keyed
  by `Contact.VCardUID`; `HouseholdMember.Role`; `Household.Type`.
- `services.GenerateHouseholdSuggestions(db, household) ([]models.RelationshipEdge, error)`
  (`backend/services/household_service.go:59`) — re-scans membership and **idempotently** returns the
  suggested edges for every applicable pair. Classification uses `HouseholdMember.Role` +
  `Contact.CRM.Kind` only (no birthday/age inference — deliberate, since thin entities often lack
  birthdays). Fully tested.
- `RelationshipEdge` with `Status: suggested` and `Source: household-inferred`, plus
  `PATCH /relationship-edges/:id/accept` and `DELETE` (which doubles as reject) — all built in §3d WP1.
- `RelationshipEdgeList.tsx` already renders a suggested section with Accept/Reject buttons, currently
  provably dead code (covered only by a component test with mocked data).

## What to build

1. **`backend/controllers/household_controller.go`** — CRUD following `circle_controller.go`'s idiom
   **exactly**, including real nested membership sub-resources
   (`POST /households/:id/members`, `DELETE /households/:id/members/:vcard_uid`) rather than a
   bulk-replace field, and a checked `409 ErrAlreadyExists` on a duplicate add.
2. **A suggestion trigger endpoint** — e.g. `POST /households/:id/suggest-relationships`. Calls
   `GenerateHouseholdSuggestions` and **persists** the returned edges idempotently (the service returns
   them; it does not save them — check before assuming). Re-running must not duplicate edges.
3. **Routes** in `backend/routes/routes.go`, next to the circles/tags block.
4. **Frontend**: `api/households.ts`, a hook, and a management surface — create a household, set its
   type, add/remove members with roles — plus a visible way to run the suggestion trigger.
5. **Verify the review loop closes**: generated suggestions must appear in the existing
   `RelationshipEdgeList` suggested section on each member's contact page, and Accept/Reject must work
   against real (not mocked) data — the first time that path has ever run for real.
6. **i18n** in all five locale files.

## Traps

- **Do not set `Contact.CRM.Kind` by direct field mutation in tests** — `BeforeSave` will not run and
  every test contact silently classifies as an adult. Use `ApplyRecordToContact`. This exact bug hit
  WP-81 and WP-83.
- Members are keyed by `Contact.VCardUID`. `HouseholdMember.MemberVCardUID` carries an explicit
  `gorm:"column:member_vcard_uid"` tag because GORM otherwise derives `member_v_card_uid` — do not remove
  it, and test against a real migrated DB.
- Suggested edges must **never** be treated as fact: `GetGraph` filters to `status: confirmed`, and the
  standards projection excludes non-confirmed. Do not weaken either.
- `91.4`'s rule is "every **human** → household pet `owned_by`", not "every adult" — a child gets one
  too. This arithmetic error was caught in WP-83's own plan; do not reintroduce it.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- A **real-DB test** (`database.InitDB`) creates a household with mixed roles plus a pet, runs the
  trigger, asserts the expected suggested edges exist, re-runs it and asserts nothing duplicated.
- `npx tsc --noEmit` clean, `npx vitest run` green.
- Verified in a real browser end to end: build a household → trigger suggestions → see them on a member's
  contact page → accept one → confirm it becomes `confirmed` and appears in the graph → reject another →
  confirm it is gone.

## Note

That last browser check is the point of the ticket. It is the first time the suggestion-review surface
has ever run against real data.
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
- Controller pattern: copy `circle_controller.go` exactly — same error handling, same 409 check, same `currentUserID(c)` pattern, same `apperrors.AbortWithError`
- `services/household_service.go:59` already has `GenerateHouseholdSuggestions` — it RETURNS edges, does NOT save them. You must persist them in the controller.
- The suggestion trigger endpoint must check for existing edges before inserting (idempotent on re-run)
- `HouseholdMember.Role` is an open string (not validated oneof) — follow `Household.Type`'s precedent
- Test: create a household with a pet + humans, run trigger, assert `owned_by` edges for every human→pet, re-run, assert no duplicates
- The Accept/Reject buttons in `RelationshipEdgeList.tsx` already exist but have never run against real data — your test must prove they work end-to-end
