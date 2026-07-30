# T2 — Circle/Tag user-assisted triage migration

| | |
|---|---|
| **Rating** | 4 — strong, frequent use |
| **Size** | M |
| **Depends on** | — |
| **Alpha** | before |
| **Source** | WP-84c-i, `91.5` |

## Why this exists

`Contact.Circles` is a flat `[]string` JSON column. WP-84 built real `Circle` and `Tag` entities and
WP-84c built their CRUD API, but **no data was ever migrated**, so the entities are unreachable and the
flat field is still what the app uses.

This is also the half of **P5's acceptance gate that has never closed** (`92.1`: "legacy relationship +
circle data migrated (dry-run-verified)" — the relationship half closed with §3d).

`91.5` is explicit that classifying a string as a Circle (a group you belong to) versus a Tag (a label) is
**"a light user-assisted step"** — an automated heuristic fails this ticket.

## What exists today

- `models.Circle` + `models.CircleMember`, `models.Tag` + `models.ContactTag`
  (`backend/models/circle.go`, `tag.go`) — UUID PKs; members keyed by `Contact.VCardUID`.
- `backend/controllers/circle_controller.go`, `tag_controller.go` — full CRUD **plus** nested membership
  sub-resources: `POST/DELETE /circles/:id/members`, `POST/DELETE /tags/:id/contacts`. A duplicate add
  returns `409`.
- `Tag` already projects into `Card.Keywords` via `projectTags` in `models/contact_record.go`.
- `Contact.Circles []string` — still populated and still what every reader uses.
- Real-DB verified in WP-84: `CircleMember`'s unique constraint is enforced.

## What to build

A triage flow, driven from the frontend against the **existing** CRUD API — no new backend endpoints
should be needed.

1. **Collect the distinct strings.** `GET /contacts/circles` (`GetCircles`) already returns the distinct
   set across the user's contacts. Use it.
2. **Present each for classification** — Circle, Tag, or skip/discard. Show how many contacts carry each
   string so the user can judge. Allow renaming while classifying (flat strings are often inconsistent).
3. **Preview before writing.** Show exactly what will be created and how many memberships each will get.
   Every migration in this repo is dry-run-first; keep that.
4. **Apply**: create each `Circle`/`Tag`, then add every carrying contact as a member via the nested
   sub-resource endpoints.
5. **Idempotent + resumable.** A 409 on a duplicate add is success, not failure — the user will run this
   twice. Do not fail the whole run on one row.
6. **Do not delete `Contact.Circles` yet.** [T3](06-T3-circle-tag-backend.md) moves the readers and
   [T4](07-T4-circle-tag-frontend.md) retires the field. Leaving both in place briefly is deliberate.

## Traps

- Membership is keyed by **`Contact.VCardUID`**, not the numeric contact ID.
- `CircleMember`/`ContactTag` have unique constraints — a repeat run must tolerate them.
- The flat field can contain near-duplicates (`"Work"` vs `"work"`). Offer merge-while-classifying or the
  user ends up with junk entities from the start.

## Done when

- Verified in a real browser end to end against a seeded set of messy circle strings: classify, preview,
  apply, re-run and confirm nothing is duplicated.
- `npx tsc --noEmit` clean, `npx vitest run` green.
- A component test covers the classify → preview → apply state machine.
- The resulting `Circle`/`Tag` rows and memberships confirmed directly against a real DB.

## Note

This ticket only **creates** the data. The app still reads the flat field until T3 and T4 land — expect
the UI to look unchanged afterwards. That is correct, not a bug.
