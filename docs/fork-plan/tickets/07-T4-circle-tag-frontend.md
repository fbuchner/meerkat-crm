# T4 — Circle/Tag frontend rewiring

| | |
|---|---|
| **Rating** | 4 — strong, frequent use |
| **Size** | L — the largest frontend surface left |
| **Depends on** | [T3](06-T3-circle-tag-backend.md) |
| **Alpha** | before |
| **Source** | WP-84c-iii |

## Why this exists

~18 frontend files consume `circles` as a flat `string[]`. Moving them onto the real `Circle`/`Tag`
entities is the last step of the WP-84c migration, and **closes P5's acceptance gate**.

## Find the work

```bash
cd frontend && grep -rn "circles" src --include=*.ts --include=*.tsx | grep -v node_modules
```

Expect roughly: contact chips, the contacts-list circle filter, `AddContactDialog`, `ContactInformation`,
`ContactDetailPage`, the network graph's node grouping/colouring, the dashboard, the import dialog, and
`api/contacts.ts`'s type definitions. Enumerate them first and work the list.

## What to build

1. **`frontend/src/api/circles.ts` and `api/tags.ts`** — CRUD + membership calls against the endpoints
   WP-84c built (`/circles`, `/circles/:id/members`, `/tags`, `/tags/:id/contacts`). Model them on
   `api/relationshipEdges.ts`.
2. **Hooks** — `useCircles` / `useTags` following `useRelationshipEdges.ts`.
3. **Replace every flat-string consumer.** Chips render a `Circle`/`Tag` (with its id), filters query by
   entity, the graph groups by real circle membership.
4. **Distinguish the two in the UI.** Circles and Tags are now different things; if they render
   identically the whole remodel is invisible to the user and T2's triage was pointless.
5. **Management surface** — create/rename/delete circles and tags, and add/remove members. WP-84c's
   nested endpoints already support this.
6. **Retire `Contact.Circles`** once nothing reads it — model field, DTOs, and the CSV export column
   (`export_controller.go` writes `strings.Join(contact.Circles, "; ")`). Coordinate with T3 on who drops
   it.
7. **i18n** — new strings in all five locale files, real translations.

## Traps

- Membership is keyed by **`Contact.VCardUID`**, not the numeric contact ID. `Contact.uid` is available
  on the summary shape (added in §3d WP0/WP3); use `getContactsByUid()` for batch name resolution.
- `NetworkGraph.tsx`/`NetworkLegend.tsx` use circles for node grouping — check them specifically, they
  are easy to miss in a grep because they read the field indirectly.
- Component tests need explicit `afterEach(cleanup)`; MUI appends `" *"` to required labels; do not nest
  a `<Chip>` inside `<Typography variant="body2">`. See `/CLAUDE.md`.
- The CSV export column change is user-visible — decide whether to emit circles, tags, or both columns.

## Done when

- `npx tsc --noEmit` clean, `npx vitest run` green.
- `grep -rn "circles" src` returns nothing meaningful outside the new API modules.
- Component tests cover the chip rendering and the filter.
- Verified in a real browser: create a circle and a tag, assign contacts to each, filter by circle,
  confirm the network graph groups correctly, confirm CSV export is right.
- Backend `go test ./...` green if you dropped the flat field.

## Milestone

Landing this closes **P5's acceptance gate** (`92.1`), open since WP-84c was split. Note that in the
commit message and update `95-backlog-and-priorities.md`.
