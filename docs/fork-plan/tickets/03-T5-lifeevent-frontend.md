# T5 — LifeEvent frontend + timeline surface

| | |
|---|---|
| **Rating** | 4 — strong, frequent use |
| **Size** | M |
| **Depends on** | — |
| **Alpha** | before |
| **Source** | WP-84, `91.6` (entity spec), `91.8` (timeline) |

## Why this exists

`LifeEvent` is one of the five entities that are **fully built on the backend and unreachable by any
user**. Model, migration, and CRUD routes all exist; there is no frontend at all. Life events ("new job",
"moved", "had a kid", bereavement) are the substance of knowing someone, and they unblock T19 (cadence),
T20b (gifts), T21 (agenda) and N2 (prep view), all of which read the timeline.

## What exists today

- `models.LifeEvent` (`backend/models/life_event.go`) — UUID PK, `UserID`, `EntityID`
  (a `Contact.VCardUID`), `Type`, `Date` (a `contactmodel.PartialDate`), `RelatedEntityIDs`
  (a **JSON array** of VCardUIDs covering both secondary participants — e.g. both spouses in a marriage —
  and the related entity, e.g. the new child).
- `backend/controllers/life_event_controller.go` — full CRUD, list filterable by `entity_id`.
- Routes: `GET/POST /life-events`, `GET/PUT/DELETE /life-events/:id`.
- Real-DB verified in WP-84: a year-only `PartialDate` and a `RelatedEntityIDs` entry round-trip exactly.
- **No** frontend: no `api/lifeEvents.ts`, no component, nothing in `ContactDetailPage`.

## What to build

### 0. Backend prerequisite — give `LifeEvent` a soft delete (decided 2026-07-30)

`LifeEvent` currently **hard-deletes**, but this ticket puts life events into the timeline alongside notes
and activities, which both soft-delete. Without this, a replica client (mobile, offline) would show a
deleted life event on the timeline while a deleted note vanishes immediately — and
[T17](17-T17-change-feeds.md)'s change feed has no way to tell it otherwise, since a hard-deleted row
leaves nothing to report.

The "no soft delete" precedent was set for **edge- and join-shaped rows** (`RelationshipEdge`'s doc
comment cites `ContactSyncLink`/`CalendarEventLink`). `LifeEvent` is not one of those — it is
first-class user-authored content, the same shape as `Note`. Adding soft delete is additive and cheap
while no production data exists, and it gets tombstones for free.

- **Migration** `000036_add_life_event_deleted_at` (hand-written up/down pair): add `deleted_at DATETIME`
  to `life_events` plus `CREATE INDEX idx_life_events_deleted_at`. Follow `000032_add_life_events`'s
  style.
- **⚠ Do NOT embed `gorm.Model` to get this.** `LifeEvent` has a **UUID string PK** (`ID string`) with its
  own explicit `CreatedAt`/`UpdatedAt`; embedding `gorm.Model` would add a conflicting `ID uint` and break
  the entity. Add the single field instead:
  ```go
  DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
  ```
  That is all GORM needs — `gorm.Model` is only a convenience wrapper around the same field.
- **⚠ This silently changes both cascade-delete paths.** `contact_controller.go`'s `DeleteContact`
  (`entity_id = ? AND user_id = ?`) and `admin_user_controller.go`'s `DeleteUser` (`user_id = ?`) both
  call plain `tx.Delete(&models.LifeEvent{})`, which becomes a *soft* delete the moment the field exists.
  For `DeleteContact` that is correct and consistent (the contact soft-deletes too). **`DeleteUser` is
  settled by [T26](08b-T26-delete-semantics.md)**, which converts that whole path to `Unscoped()` —
  so leave the plain `tx.Delete` here and do not special-case it. If T26 has already landed, follow what
  it established.
- `life_events` has **no unique constraint**, so soft delete introduces no index collision — unlike
  `contacts` and `users`, which T26 fixes.
- **⚠ The existing cascade test will not catch this either way.** `admin_user_controller_test.go`'s
  `assertGone` helper counts via `db.Model(...).Count()`, and **GORM excludes soft-deleted rows from
  that count** — so it stays green whether the rows are gone or merely marked. If you want the behaviour
  pinned, assert with `Unscoped()`.

### 1. Frontend

1. **`frontend/src/api/lifeEvents.ts`** — types + CRUD calls. Mirror the structure of
   `api/relationshipEdges.ts`, which is the most recent complete example of this pattern.
   Check the controller for the exact response envelope (create may wrap, update may not — that asymmetry
   exists elsewhere in this codebase and burned §3d WP3).
2. **`frontend/src/hooks/useLifeEvents.ts`** — mirror `useRelationshipEdges.ts`, including the
   mount-effect gotcha: pass the freshly-fetched contact's `uid` **directly** into the refresh function
   rather than reading it from state, or the first load silently fetches nothing.
3. **Add/edit dialog** — type selector, `PartialDate` input, optional related-entity picker (reuse the
   debounced contact `Autocomplete` from `RelationshipEdgeDialog.tsx`).
4. **Contact detail surface** — a life-events section on `ContactDetailPage.tsx` /
   `ContactInformation.tsx`, following how `RelationshipEdgeList` is wired in.
5. **Timeline integration** (`91.8`) — life events appear in the contact timeline alongside notes and
   activities, ordered by date.
6. **i18n** — event-type labels and all UI strings in all five locale files, real translations.

## Traps

- **`PartialDate` is not a `time.Time`.** Year-only (`1985`) and month-day-only (`--03-14`) are both
  legal and must render and round-trip correctly. Do not funnel it through a normal date picker that
  demands a full date. Look at how `Birthday`/`Anniversary` already handle partial dates in
  `api/contacts.ts` and the contact form.
- Event *types* are a hardcoded frontend mirror of the backend's accepted values — no dynamic list
  endpoint exists anywhere in this codebase. Add a comment saying it must stay in sync.
- `RelatedEntityIDs` is a JSON array of VCardUIDs with no nested contact data — resolve names via
  `getContactsByUid()` in `api/contacts.ts` (the `?vcard_uid=` batch filter added in §3d WP0).

## Done when

- `npx tsc --noEmit` clean, `npx vitest run` green, with component tests covering the dialog and list.
- Verified in a real browser: create a life event with a **year-only** date, edit it, delete it, and see
  it in the contact's timeline in date order.
- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green after step 0.
- A **real-DB test** (`database.InitDB`, not `AutoMigrate`) proves the new `deleted_at` column exists in
  the real migrated schema and that deleting a life event soft-deletes it — i.e. it disappears from a
  normal query but is still present under `Unscoped()`. That round trip is what T17's change feed will
  rely on.

## Next

**Do [04 · T5b](04-T5b-lifeevent-reminders.md) immediately after.** Without it, life-event dates generate
no reminders and the feature is inert — alpha cannot evaluate whether it is useful.
