# T14 — WP-89 external-link substrate (ExternalIdentity / ExternalActivity)

| | |
|---|---|
| **Rating** | 2 alone — invisible infrastructure that enables 3s |
| **Size** | M |
| **Depends on** | — |
| **Alpha** | after (all-new tables; nothing existing changes) |
| **Source** | `92.4`, `91.12` |

## Why this exists, and why it comes first

The generic substrate is built **before** the first concrete integration, deliberately, **so that no
integration grows bespoke tables.** That ordering is the entire point of this ticket — keep it even
post-alpha. If Immich is built first, Immich gets its own columns and the next integration gets its own
too, and you have the sprawl this design exists to prevent.

Rated 2 on its own because a user sees nothing from it; it is the enabler for
[T15/T16](33-T15-T16-immich.md) and later `92.7` integrations.

## What to build

Per `91.12`:

- **`ExternalIdentity`** — this contact *is* this thing in that external system. Contact `VCardUID`,
  system identifier (e.g. `immich`), external ID, optional deep-link URL, last-synced timestamp.
- **`ExternalActivity`** — something that happened in an external system, linkable into the timeline.
  System, external ID, type, occurred-at, payload/summary, and the contact(s) it concerns.
- **A generic link/enrichment API** — CRUD over both, plus the ability to attach an `ExternalActivity`
  into a contact's timeline.

## What already anticipates this

- **`Activity.ExternalRef`** (`backend/models/activity.go`) — added in WP-84 specifically to link an
  Interaction to an `ExternalActivity`, and it has had no consumer since. Use it rather than inventing a
  parallel link.
- `93-integration-spec-template.md` — the per-integration template each future integration instantiates.
  Read it; this substrate should satisfy what that template assumes.

## Traps

- **Resist system-specific fields.** The moment `ExternalIdentity` grows an `immich_person_id` column,
  the abstraction has failed. System-specific data goes in a JSON payload field — follow
  `RelationshipEdge.Metadata`'s `map[string]interface{}` + `serializer:json` pattern.
- Keyed by `Contact.VCardUID`, not the numeric ID — the graph invariant.
- **Cascade delete**: add both tables to `DeleteContact` and `DeleteUser`, and to the real-DB cascade
  test. Every WP-80+ entity is scoped by `VCardUID` there.
- Unique constraint on (system, external ID, user) so a re-sync does not duplicate.
- External URLs are user-influenced — the SSRF guard (`httputil/fetch.go`, dialer-level) applies to
  anything this fetches.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- A **real-DB test** covers the round trip, the unique constraint, and cascade cleanup on contact delete.
- The API is demonstrably generic — sketch how a second, unrelated integration (say Paperless-ngx) would
  use it without schema changes. If it cannot, the abstraction is wrong and Immich will prove it later.

### Ticket-specific
- Build the generic substrate BEFORE any concrete integration — this ordering is the point of the ticket
- `ExternalIdentity`: this contact IS this thing in that system. Keyed by Contact.VCardUID, not numeric ID.
- `ExternalActivity`: something that happened externally, linkable into the timeline. Keyed by system + external ID.
- `Activity.ExternalRef` in `models/activity.go` already exists to link an Interaction to an ExternalActivity — use it, don't add a parallel link.
- Unique constraint: `(system, external_id, user_id)` — prevents duplicate sync. Hard delete per T26 (edge/join-shaped).
- System-specific data goes in a JSON payload field — follow `RelationshipEdge.Metadata`'s `map[string]interface{}` + `serializer:json` pattern
- SSRF guard: external URLs are user-influenced — `httputil/fetch.go`'s dialer-level guard applies to any fetch this does
- Cascade delete: add both tables to `DeleteContact` and `DeleteUser` cascade lists
- Test: the API must be demonstrably generic — sketch how Paperless-ngx would use it without schema changes

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
