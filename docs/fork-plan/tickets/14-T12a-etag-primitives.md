# T12a — Activity/LifeEvent sync primitives (ETag)

| | |
|---|---|
| **Rating** | 2 (**4 if a mobile client is real**) — invisible infrastructure |
| **Size** | S |
| **Depends on** | — |
| **Alpha** | **before** — cheap additive column while the tables are empty |
| **Source** | Split out of WP-87 (`92.3`) during the 2026-07-30 alpha-cut-line pass |

## Why this exists

`ETag` exists **only** on `Contact` and `ContactSubscription`. `Activity` got a `UUID` in WP-84 but has no
ETag; `LifeEvent` has neither concern addressed. CalDAV clients need a per-resource ETag to sync and
cache, so [T12b](35-T12b-caldav-serve.md) cannot be built without one.

This ticket is split out so the **schema** half lands pre-alpha (an additive column plus a backfill,
trivial while the tables are empty) without dragging a whole CalDAV server implementation — sized L — in
front of alpha to get it.

## The pattern to copy

`Contact.ETag` (`backend/models/contact.go`):

```go
ETag string `gorm:"column:etag" json:"-"` // Sync conflict detection
```

generated in the create/save hooks as:

```go
c.ETag = fmt.Sprintf("e-%d-%d", c.ID, c.UpdatedAt.Unix())
```

refreshed on save only when it actually changed, via `tx.Model(c).UpdateColumn("etag", c.ETag)`. Read the
real implementation — it takes care to avoid a save loop.

## What to build

1. **Migration** `000036_add_activity_etag` (hand-written SQL up/down pair) adding `etag` to `activities`,
   plus `life_events` if it will be served too — decide and say which.
   **Backfill existing rows in the same migration**, exactly as `000030` did for `activities.uuid`. That
   is the proven precedent here.
2. **Model + hooks** mirroring `Contact`'s. Note `LifeEvent` has a **UUID string PK**, not a uint, so its
   ETag derivation differs slightly — base it on the UUID + `UpdatedAt`.
3. Add the explicit `gorm:"column:etag"` tag. **Do not** let GORM derive it — it produces `e_tag`, which
   is exactly how `ContactSyncLink.ETag` shipped broken and silently killed CardDAV incremental sync
   writes.

## Traps

- Migrations are hand-written SQL up/down pairs. Never add the column by editing the struct alone.
- The `e_tag` derivation bug above is the single most repeated mistake in this codebase. Explicit tag,
  real-DB test, every time.
- Changing `UpdatedAt` semantics affects anything ordering by it — check before touching save hooks.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- A **real-DB test** (`database.InitDB`, not `AutoMigrate`) proves: a new Activity gets an ETag; updating
  it changes the ETag; a row whose ETag was cleared is recovered by the migration's backfill (mirror
  `000030`'s own test for `activities.uuid`).
- Hand-verified: remove the explicit column tag, confirm the real-DB test fails, restore. That failure is
  the whole point of the test.
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
- Study `Contact.ETag` in `models/contact.go` for the exact pattern — hooks, column tag, save-loop avoidance
- `Activity` uses `gorm.Model` (uint PK) — ETag format: `"e-{ID}-{UpdatedAt.Unix()}"`
- `LifeEvent` has UUID string PK — ETag format: `"e-{UUID}-{UpdatedAt.Unix()}"`. Use `c.UpdatedAt.Unix()` on `time.Time`.
- **Critical: `gorm:"column:etag"` tag is mandatory.** Without it, GORM derives `e_tag` and the column is named `etag` in the migration → silent mismatch. This has shipped broken before.
- Migration: include backfill for existing rows (like `000030` did for `activities.uuid`). Add column, then `UPDATE SET etag = printf('e-%d-%d', id, unixepoch(updated_at))` in the same migration file.
- Real-DB test: create an Activity → assert ETag exists → update it → assert ETag changed. Prove with `database.InitDB`, not `AutoMigrate`.
- Hand-verify: remove the `gorm:"column:etag"` tag, confirm the real-DB test fails (it will show `e_tag` column not found), restore.
