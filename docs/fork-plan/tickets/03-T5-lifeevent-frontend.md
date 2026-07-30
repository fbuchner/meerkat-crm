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
- Backend untouched (or `go test ./...` green if you had to adjust the controller).

## Next

**Do [04 · T5b](04-T5b-lifeevent-reminders.md) immediately after.** Without it, life-event dates generate
no reminders and the feature is inert — alpha cannot evaluate whether it is useful.
