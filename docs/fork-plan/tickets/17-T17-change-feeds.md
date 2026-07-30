# T17 — WP-92 change feeds + cursor pagination

| | |
|---|---|
| **Rating** | 2 (**4 if a mobile client is real**) |
| **Size** | M |
| **Depends on** | [T8](16-T8-openapi.md) — but read the coordination note below |
| **Alpha** | **before** — breaking API contract change |
| **Source** | `92.5` |

## Why this is pre-alpha

Not a data risk — an **API-contract** risk. Every list endpoint returns an offset-shaped envelope today:

```json
{ "contacts": [...], "total": 412, "page": 1, "limit": 20 }
```

`92.5` explicitly replaces that with cursors ("not offset — for large histories/timelines"). Breaking
that contract is free while we are the only consumer and costs a versioning scheme once anything external
depends on it.

It also interacts with T8: publishing the OpenAPI spec and *then* flipping pagination means publishing
the contract twice.

**Coordination call to make first:** T8 → T17 (document, then revise) or T17 → T8 (change, then document
once). The second is cleaner if you are confident in the cursor design; the first is safer if T17 might
slip. Decide explicitly rather than discovering the conflict mid-ticket.

## What exists today

- `GetPaginationParams(c)` in `backend/controllers/helpers.go` — the shared offset/limit parser used by
  every list handler.
- The offset envelope above, emitted by `GetContacts` and every other list endpoint.
- `Activity.UUID` and the UUID-PK entities give stable per-row identity; `UpdatedAt` exists everywhere.
- Frontend: `Pagination` components (e.g. `NotesPage.tsx`) built around page numbers.

## What to build

1. **A cursor scheme.** Needs a total order that is stable under concurrent writes — `(updated_at, id)`
   is the usual answer; a bare `updated_at` is not unique and will drop or duplicate rows at page
   boundaries. Encode it opaquely (base64 of the tuple), so the shape can change later without another
   contract break.
2. **A shared helper** alongside `GetPaginationParams`, so every list endpoint moves the same way rather
   than each inventing its own.
3. **Migrate the list endpoints.** Decide whether `total` survives — an exact count is expensive on large
   tables and is the usual thing cursor pagination gives up. If you drop it, the frontend's page-number
   UI has to become infinite-scroll or next/prev, which is real frontend work; scope it in.
4. **Change feeds** — the `?since=<cursor>` half: "what changed since I last synced," which is what makes
   this useful for a mobile client rather than just a pagination refactor. Deletes need to be
   representable; a client that only ever sees creates and updates cannot converge. Soft-deleted rows
   (`gorm.Model.DeletedAt`) can carry this — check each entity, since the UUID-PK entities deliberately
   have **no** soft delete.
5. **Index support** — a composite index on `(user_id, updated_at, id)` per paginated table, or the
   cursor query degrades to a scan.

## Traps

- **Deletes are the hard part of any change feed.** Entities like `RelationshipEdge`, `CircleMember`, and
  `ContactTag` hard-delete by design (matching `ContactSyncLink`'s precedent). Decide how a feed reports
  those before writing code — a tombstone table, or accepting that clients must periodically full-sync.
- `GetContacts` has a `?vcard_uid=` batch branch that **deliberately bypasses pagination entirely** and
  returns everything requested. Leave it alone; it is not a list view.
- The count query in `GetContacts` duplicates the main query's filters. If `total` goes away, both go.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- Tests prove cursor pagination is stable across a concurrent insert at a page boundary — no dropped or
  duplicated row. This is the bug this scheme exists to prevent, so it is the test that matters.
- A change-feed test proves `?since=` returns only rows changed after that cursor.
- Frontend updated and `npx tsc --noEmit` / `npx vitest run` green.
- OpenAPI updated (or T8 sequenced to follow).
