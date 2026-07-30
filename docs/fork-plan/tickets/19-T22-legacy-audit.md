# T22 — Legacy-representation / dead-code audit + migration squash

| | |
|---|---|
| **Rating** | 3 — health, not features |
| **Size** | L |
| **Depends on** | everything above — it sweeps up their debt too |
| **Alpha** | **before, and last** — the last ticket inside the no-production-data window |
| **Source** | Tier 6 |

## Why this exists, and why it is last

Prompted by §3d: what started as a small `GetGraph` rewire turned out to be a whole legacy subsystem
(`models.Relationship`) still doing real work long after its replacement existed — because this fork's
build pattern across WP-80→84c was consistently *"layer a new representation on top of the old one,
bridge them, defer full removal."*

The migration squash in particular is safe **only** while no deployment needs a stepwise upgrade path
preserved — true right up until alpha, false forever after. Placing this at the very end of the pre-alpha
run puts it as late as possible while still inside that window, so it also sweeps up debt created by
every ticket before it.

**This ticket is the audit itself.** Methodology mirrors Tier 3c item 11: identify candidates, then decide
keep / remove / defer **per candidate** with a written reason. It is not a blind deletion pass.

## Known starting candidates

Found in passing rather than by a dedicated sweep — expect to find more.

1. **`Contact.VCardExtra`** — its own doc comment (`models/contact_record_reverse.go`) says `Passthrough`
   supersedes it "in spirit," but it was never removed or confirmed dead. Check what, if anything, still
   reads it as authoritative versus `Passthrough`.
2. **`RelationshipEdge.LegacyRelationshipID`** — WP-81 migration bookkeeping, vestigial since §3d WP5
   deleted the `models.Relationship` it referenced. Its own doc comment already says no application code
   reads it. Removing it is a migration (drop column + its unique index from `000028`).
3. **`cmd/backfill-custom-fields` and `cmd/backfill-contact-records`** — one-shot tools whose migrations
   have run. (`cmd/backfill-relationship-edges` was already removed in §3d WP5, out of compile necessity
   rather than choice.) If [T20a](10-T20a-preferences.md) added another backfill, it joins this list.
   **Check [T24](15-T24-test-coverage.md) did not just add coverage to something you are deleting.**
4. **Migration squash** — ~35 incremental migration files for a repo with zero production data. Squashing
   to a single clean baseline is normal pre-release hygiene *specifically because* there is no live
   deployment needing a stepwise path. Note the tradeoff flips permanently at alpha.
5. **Dead/duplicate scaffolding** — one confirmed instance already found and removed (`types/index.ts` /
   `types/api.ts` carried an unused duplicate `Relationship` type). Do a real sweep rather than assuming
   that was the only one: a Go dead-code tool plus a frontend unused-export check.

## Doing the squash safely

- The squashed baseline must produce a schema **byte-for-byte equivalent** to running all ~35 in order.
  Prove it: build a DB both ways and diff `sqlite_master` (plus `PRAGMA table_info` per table). Do not
  eyeball it.
- Preserve the `schema_migrations` bookkeeping so a DB that already ran the old chain is not re-migrated.
  If that is not cleanly possible, say so — with no prod data, "recreate from scratch" is an acceptable
  answer, but it must be a stated decision.
- Keep `database/migrate_test.go`'s existing guards green: migrations apply to an empty DB, WAL is on,
  `foreign_keys` is on and enforced, and the legacy `relationships` table stays dropped.

## Done when

- Every candidate has a written disposition (removed / kept with reason / deferred with reason), recorded
  in `95-backlog-and-priorities.md`.
- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- The squash is proven equivalent by schema diff, not inspection.
- `npx tsc --noEmit` clean, `npx vitest run` green after any frontend dead-code removal.
- A real end-to-end smoke test against a freshly migrated DB — register, create a contact with
  relationships and life events, export, re-import — because this ticket touches the schema foundation.

## After this ticket

**Alpha.** Real data starts existing, and the cheap-cleanup window closes. Anything not done here gets
materially more expensive from this point on.
