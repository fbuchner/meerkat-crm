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
