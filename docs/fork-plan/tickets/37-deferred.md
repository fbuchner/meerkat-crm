# P1b / P2 / P3 / P4 — deferred, need design passes

| | |
|---|---|
| **Rating** | 1–2 |
| **Alpha** | after |
| **Source** | Tier 5, `92.7`, `90` D1, `80` |

These are **not implementation-ready and are not meant to be.** Each needs its own design pass before it
can be broken into work packages. This file records what they are and what a design pass would have to
settle, so nobody starts one by accident.

## P1b — Contact sharing: standing/live share + permission model

**XL. Depends on [P1](31-P1-contact-sharing.md).**

Everything Tier 5's section describes beyond the one-time copy: persistence for a *live* share that
re-syncs, a shared-vs-private field model, a real permission model, and **re-confirmation when a field is
newly marked sensitive after the share was created**.

A design pass must settle:
- Does a standing share re-apply the field-selection default on every sync, or only at creation time?
  (Tier 5 flags this as an open question, not a decision.)
- What is the permission model — read-only, read-write, revocable, time-bounded?
- What happens to already-shared data when a share is revoked?
- How does the recipient's own editing interact with incoming updates? This is the same reconciliation
  problem as [T13](36-T13-two-way-calendar.md), with the same trap.

**Do not start this as part of P1.** P1 is deliberately a one-time copy; conflating them is what produced
the original XL estimate.

## P2 — Other integrations

**Depends on [T14](32-T14-external-link-substrate.md).**

Dawarich/GeoPulse, Jellyfin, Audiobookshelf, Paperless-ngx, Nextcloud. Each is a
`93-integration-spec-template.md` instance on top of the T14 substrate — mostly level 1–2, API-based, no
upstream dependencies.

Explicitly **pulled in only when a concrete need arises**, one at a time. If building one requires
changing T14's substrate, that is a signal the substrate is wrong — fix it there rather than
special-casing.

## P3 — AI / Ollama layer

**Rating 1.** Summarization, entity/relationship/life-event extraction, timeline synthesis,
memory-curator suggestions.

Gated on two things: everything structured existing first, and the **propose-then-approve** workflow —
which is already the pattern used by household inference ([T1](09-T1-households.md)) and its
suggested-edge review surface. Any AI output must land as a *suggestion* a human confirms, never as fact.

**`90` D1 is explicit: this is not an AI-first project.** Would revisit D1's storage decision *only* if
vector search proves necessary, and then via an external sidecar — never a primary-store migration.

## P4 — Local-model code-gen pilot

**Rating 1.** See `80-local-model-pilot.md`. Deferred on its own terms; re-enters when **mobile client
work begins**, independent of this backend roadmap.

Note this connects to the open question flagged in `95-backlog-and-priorities.md`: whether a mobile client
is real at all. That answer re-rates [T8](16-T8-openapi.md), [T12a](14-T12a-etag-primitives.md), and
[T17](17-T17-change-feeds.md) from 2 to 4 — so it is worth settling before those tickets rather than
before this one.

### Post-alpha note
This ticket is post-alpha — real production data exists. Changes that modify schemas or data must be additive and non-destructive. Migration files must be hand-written SQL up/down pairs. Test against `database.InitDB`, not `AutoMigrate`. For integrations: SSRF protection via `httputil.SafeDialContext` is mandatory for any outbound requests.

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
