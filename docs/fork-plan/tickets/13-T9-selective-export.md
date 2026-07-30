# T9 — WP-97 selective field export + sensitivity gating

| | |
|---|---|
| **Rating** | 3 — nice to have, likely used |
| **Size** | L |
| **Depends on** | — (P0 + WP-73, both long done) |
| **Alpha** | before |
| **Source** | `92.6b` — **unusually well specified; read it in full first** |

## Why this exists

Google Contacts' "choose which fields to export" as the reference feature, at the user's request. It also
builds the field-selection model and UI that contact sharing ([P1](31-P1-contact-sharing.md)) is meant to
**reuse rather than reinvent**.

`92.6b` carries the full scope plus two user clarifications. This file summarises and adds the code-level
findings; it does not replace it.

## What to build

1. **A field-selection representation** over `contactmodel.Card`'s top-level sections (emails, phones,
   addresses, organizations, media/photo, personal info, related-to, …) — **coarse-grained like Google's
   picker, not per-value.**
2. **Apply it by filtering the `Record`/`Card` *before* it reaches an exporter**, never inside each
   exporter. `vcard3.Adapter`, `vcard4.Adapter`, and `jscontact.Adapter` all already consume the same
   neutral `Card`, so **one filter function makes the selection apply to all three identically, with zero
   changes to any adapter.** Call sites: `ExportContactsAsVCF` and `ExportContactsAsJSContact` in
   `backend/controllers/export_controller.go`.
3. **A field-picker UI** wired into the existing export flow: checkboxes per section, sensible
   "select all" default for ordinary fields.

## The sensitivity rules — binding, not decoration

Per `91.13` and the two clarifications in `92.6b`:

- **Ordinary (`normal`) categories are opt-*out*** — on by default, can be excluded.
- **Sensitive items are opt-*in*** — off by default, can be included. Same control, opposite default, not
  a second mechanism.
- **An unchecked box is explicitly NOT sufficient gating.** A sensitive item must be (a) visually
  distinct — a warning colour/icon, not merely unchecked — **and** (b) behind a deliberate extra action
  before its control is even interactive. Acceptable mechanisms: a per-item "unlock to include" toggle, a
  confirmation prompt on check, or a single "reveal sensitive fields" step for the whole group. The exact
  choice is yours; "gated behind an extra action" is the requirement. This is foot-gun prevention.

## The non-obvious backend change

**`projectRelationshipEdges` in `backend/models/contact_record.go` enforces `Sensitivity: normal` as an
unconditional SQL filter with no override parameter at all** — confirmed by reading it, not assumed.

So "include these sensitive edges just this once" cannot be done with a Card filter in front of an
otherwise-unchanged projection: `RecordForContact` (or an equivalent path this ticket adds) needs a way
to say *also include these specific sensitive items this time*. Design that explicitly; it is small but
it is real, and discovering it mid-implementation is what `92.6b` was trying to prevent.

The same pattern applies to `projectTags` and `projectCustomFields` if their sensitivity filters should
also be overridable — decide consistently.

## Traps

- Do not add a per-format code path. The whole design value is one filter, three formats.
- The CSV data export (`ExportData`) is a *different* thing — the user's own full backup, which
  deliberately includes everything regardless of sensitivity (see the comment in its RELATIONSHIPS
  section). Do not accidentally apply the picker to it.
- Component tests need explicit `afterEach(cleanup)`.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- Tests prove the same selection produces consistent omissions across **all three** export formats.
- A test proves a sensitive item is excluded by default and included only when explicitly opted in.
- `npx tsc --noEmit` clean, `npx vitest run` green; a component test proves the sensitive control is not
  interactive until the gating action is taken.
- Verified in a real browser across vCard 3, vCard 4, and JSContact.
