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
## Flash implementation notes

### Files to read first
- CLAUDE.md (repo conventions, traps, commands)
- Look at an existing implemented ticket (e.g. T5/LifeEvent) for the full pattern: model → controller → routes → api → hooks → dialog → list → ContactInformation wiring → ContactDetailPage wiring → i18n
- For households: study `circle_controller.go` and `circle_controller_test.go` — the household controller must follow this exact idiom

### Tests you must write before considering it done
- Backend: real-DB test (`database.InitDB`, not `AutoMigrate`) for the core round-trip
- Backend: controller tests covering create, update, delete, cross-user ownership rejection, 409 on duplicate member add
- Frontend: component test for the dialog and list (follow `MergeContactsDialog.test.tsx` pattern — `afterEach(cleanup)`, mock `fetch` with `vi.stubGlobal`)
- Frontend: the ticket's specific assertion (e.g. \"suggested edges appear in RelationshipEdgeList\")

### Self-verification checklist
1. `npx tsc --noEmit` clean
2. `npx vitest run` green (ALL tests, not just yours)
3. `cd backend && go build ./... && go vet ./... && gofmt -l . && go test ./...` green
4. Run `make migrate-up` to verify migrations apply cleanly
5. Hand-verify: break one assertion, confirm the test fails, restore

### Common traps (beyond CLAUDE.md)
- `gorm.Model` only works on uint-PK entities — UUID PK models need explicit `ID`/`CreatedAt`/`UpdatedAt`/`DeletedAt`
- Membership is keyed by `Contact.VCardUID`, not numeric ID — use `gorm:\"column:member_vcard_uid\"` tag or GORM derives `member_v_card_uid`
- Backend tests use `setupRouter()` from `activity_controller_test.go` (sets db + userID + cfg in context, uses AutoMigrate)
- Frontend component tests: `afterEach(cleanup)` is mandatory; MUI appends `\" *\"` to required field labels
- All 5 locale files (`de/es/fr/it/en`) need real translations, not English placeholders

### Ticket-specific
- The filter must run BEFORE the adapters, not inside them. `vcard3.Adapter.Export()`, `vcard4.Adapter.Export()`, and `jscontact.Adapter.Export()` all consume the same `contactmodel.Record` — one filter function, zero adapter changes.
- Call sites: `export_controller.go` → `ExportContactsAsVCF` and `ExportContactsAsJSContact`. NOT `ExportData` (CSV full backup — includes everything).
- `projectRelationshipEdges` in `contact_record.go` has an UNCONDITIONAL `WHERE sensitivity = 'normal'` filter. You need to add an override parameter for selective export. Same pattern for `projectTags` and `projectCustomFields` — decide consistently.
- Sensitivity: ordinary fields are opt-out (checked by default), sensitive are opt-in (unchecked by default). Sensitive items must be visually distinct AND behind an extra deliberate action before interactive.
- Frontend field picker: coarse-grained (whole sections like emails, phones, addresses), not per-value. Follow Google Contacts' reference.
- Test: same selection produces identical omissions across all three formats. Sensitive item excluded by default, included only with explicit opt-in. Component test: sensitive control not interactive until gating action taken.
