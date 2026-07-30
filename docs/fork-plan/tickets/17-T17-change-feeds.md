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
   this useful for a mobile client rather than just a pagination refactor. See **Delete handling** below;
   that decision is already made.
5. **Index support** — a composite index on `(user_id, updated_at, id)` per paginated table, or the
   cursor query degrades to a scan.

## Delete handling — decided 2026-07-30

**No tombstone table.** `deleted_at` provides tombstones where soft delete already exists; entities that
hard-delete are handled by **periodic full resync of those collections**, not by tracking their deaths.
Rationale: do not retain a marker for something we deliberately hard-delete.

This works cleanly because of how the split falls out — verified by auditing which models actually embed
`gorm.Model`:

| | Entities | Volume per user | Sync strategy |
|---|---|---|---|
| **Soft delete** (`deleted_at`, free tombstone) | Contact, Note, Activity, Reminder, ReminderCompletion, User, ApiToken, Webhook, WebhookDelivery, ContactSubscription, CalendarSubscription, **LifeEvent** (added by [T5](03-T5-lifeevent-frontend.md) step 0) | **Large and growing** — notes and activities accumulate for years | Incremental. `?since=` returns updated *and* soft-deleted rows; the client applies the deletion. |
| **Hard delete** (no soft delete) | RelationshipEdge, Circle, CircleMember, Tag, ContactTag, Household, HouseholdMember, FieldDefinition, FieldValue, ContactSyncLink, CalendarEventLink | **Bounded small** — hundreds to low thousands | **Full resync of the collection.** Cheap enough to just re-pull. |

**The split is favourable, which is why this works:** every big, unboundedly-growing table already has
soft delete, and every hard-delete table is small. A client can pull all of a user's relationship edges,
tags, and life events in a handful of requests. Resyncing those on app foreground (or daily, or on
demand) costs far less than maintaining a tombstone table and its retention policy.

**Do not add soft delete to the hard-delete entities just to get tombstones.** Their lack of it is
deliberate — `RelationshipEdge`'s own doc comment records the reasoning, matching
`ContactSyncLink`/`CalendarEventLink`'s precedent for edge- and join-shaped rows.

### Two things this leaves open

1. **The timeline is derived, so it needs nothing.** Verified: there is no persisted timeline table —
   `ContactTimeline.tsx` composes it client-side from notes and activities, **both of which are
   soft-delete**. So the timeline stays correct under pure incremental sync, with no special handling.

2. **`LifeEvent` was the one exception — resolved 2026-07-30: it gets a soft delete.** T5 puts life
   events *into* the timeline, and `LifeEvent` hard-deleted, so a replica would have shown a deleted life
   event while notes and activities vanished immediately.

   The "no soft delete" precedent was set for **edge/join-shaped rows**; `LifeEvent` is not one — it is
   first-class user-authored content, the same shape as `Note`. Adding `deleted_at` is additive, cheap
   pre-alpha, and buys a free tombstone.

   **[T5](03-T5-lifeevent-frontend.md) owns the work** (its step 0), including the trap that `LifeEvent`
   has a UUID string PK so `gorm.Model` must **not** be embedded. By the time this ticket is picked up,
   every timeline input soft-deletes and the table above is accurate as written. The container entities
   (Circle, Tag, Household) and the join rows stay hard-delete and are genuinely fine on resync.

### ⚠ The retention window is the sync horizon

[T26](08b-T26-delete-semantics.md) adds a purge job that hard-deletes soft-deleted rows after a retention
window (default 30 days). **That window is therefore the maximum age of a usable cursor**: a client
offline longer than it has missed tombstones that no longer exist, and cannot converge by incremental
sync alone — it must full-resync everything, not just the hard-delete collections.

Design for this explicitly rather than discovering it: the feed needs a *"your cursor is older than the
retention window — full resync required"* response, and the retention setting and the sync horizon must
be the same number, not two independently-configured ones that can drift apart.

### What the client contract must say

Whatever you build, the feed has to tell a client **which collections are incremental and which need
resync**, plus the "your cursor is too old" answer above. A client cannot infer any of it.

## Traps

- **Soft-deleted rows must actually appear in the feed.** GORM excludes them from queries by default —
  an incremental feed needs `Unscoped()` plus an explicit `deleted_at IS NOT NULL` marker in the
  response, or deletes silently never propagate and the whole scheme above fails quietly. Test this
  specifically; it is the single easiest thing to get wrong here.
- `GetContacts` has a `?vcard_uid=` batch branch that **deliberately bypasses pagination entirely** and
  returns everything requested. Leave it alone; it is not a list view.
- The count query in `GetContacts` duplicates the main query's filters. If `total` goes away, both go.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- Tests prove cursor pagination is stable across a concurrent insert at a page boundary — no dropped or
  duplicated row. This is the bug this scheme exists to prevent, so it is the test that matters.
- A change-feed test proves `?since=` returns only rows changed after that cursor, **and that a
  soft-deleted row is returned as a deletion** rather than silently vanishing.
- The response makes clear which collections are incremental and which require full resync.
- Frontend updated and `npx tsc --noEmit` / `npx vitest run` green.
- OpenAPI updated (or T8 sequenced to follow).
