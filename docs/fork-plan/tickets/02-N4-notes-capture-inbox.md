# N4 — Notes: dead-end journal → capture inbox

| | |
|---|---|
| **Rating** | 4 (was **1** as currently built) |
| **Size** | S–M |
| **Depends on** | — |
| **Alpha** | before |
| **Source** | New (re-scope from the 2026-07-30 product review) |

## Why this exists

The standalone notes page is inherited from Meerkat and, as built, rates **1**:
`frontend/src/NotesPage.tsx` creates notes via `createUnassignedNote` and passes `allContacts={[]}` to
its dialog, so **a note created there can never be attached to a contact from the UI**. It is a dead-end
diary inside a relationship CRM — it feeds nothing in the graph, timeline, or cadence.

The "notes about a group" defence does not hold either: `Activity` already supports multiple contacts and
is the correct home for that.

**The re-scope that makes it a 4:** turn the same page into a **capture inbox** — jot now, file onto a
contact later. That targets the real friction in a personal CRM: the useful detail always arrives when
you are nowhere near the contact record. This is a UI change; the backend already supports it.

Pre-alpha because shipping the dead-end version generates exactly the pile of unfilable orphan notes the
inbox exists to prevent.

## What exists today

- `models.Note` (`backend/models/note.go`): `ContactID *uint` — **nullable**, so unassigned notes are
  already representable. (It carries a `validate:"required"` tag that the controller does not enforce for
  the unassigned path; tidy that up so the model tells the truth.)
- `controllers.UpdateNote` (`backend/controllers/note_controller.go:~223`) already does
  `note.ContactID = updatedNote.ContactID` and validates the target contact belongs to the user —
  **assignment already works over the API**.
- `frontend/src/NotesPage.tsx` — timeline-style list, search, date filter, pagination.
- `frontend/src/api/notes.ts` — has `createUnassignedNote`, `updateNote`, `deleteNote`.
- `frontend/src/components/AddNoteDialog.tsx` and `EditTimelineItemDialog.tsx` — both already accept an
  `allContacts` prop; `NotesPage` just passes `[]`.

## What to build

1. **Reframe the page as an inbox.** Retitle (i18n key change across all 5 locales), and show a count of
   unfiled notes so it reads as a queue rather than an archive.
2. **Enable filing.** Pass real contacts into the dialogs instead of `[]` — reuse the debounced
   `Autocomplete` contact-picker pattern from `RelationshipEdgeDialog.tsx` rather than loading every
   contact up front.
3. **Filed notes leave the inbox.** Default the list to `contact_id IS NULL`; a note that gains a contact
   disappears from the inbox and appears on that contact's timeline (which already renders notes).
4. **Backend**: make sure listing can filter to unassigned only — add an `?unassigned=true` (or
   `?contact_id=none`) query param to `GetNotes` if it cannot already express that.
5. **Quick capture**: keep creation frictionless — content + date, contact optional. The whole point is
   that filing is deferred.
6. **i18n**: new/changed strings in all five locale files with real translations.

## Traps

- `NotesPage` currently imports `createUnassignedNote` specifically. If you generalise creation, make
  sure the "no contact" path still works — do not make `contact_id` required.
- Component tests here need explicit `afterEach(cleanup)` (see `/CLAUDE.md` frontend trap 1).
- Watch for i18n key collisions when adding a heading near existing chip/label text — a duplicate string
  broke a `getByText` assertion during §3d WP3.

## Done when

- `npx tsc --noEmit` clean, `npx vitest run` green.
- A component test proves the inbox lists only unassigned notes, and that assigning a contact removes a
  note from it.
- Verified in a real browser: create an unfiled note, file it to a contact, confirm it leaves the inbox
  and appears on that contact's timeline.
- Backend `go test ./...` green if `GetNotes` changed.

## Open decision

**Should the inbox nag?** A badge/count in the nav makes it a real queue; silence makes it a shoebox. An
inbox nobody triages is the same dead end in a different shape — but a permanent unread badge is its own
kind of annoying. Pick one deliberately.
