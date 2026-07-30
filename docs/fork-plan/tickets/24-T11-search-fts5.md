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
