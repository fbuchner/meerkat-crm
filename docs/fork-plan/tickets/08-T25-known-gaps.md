# T25 — Known small functional gaps sweep

| | |
|---|---|
| **Rating** | 3 — nice to have, likely used |
| **Size** | S |
| **Depends on** | — |
| **Alpha** | before — it closes a real data-loss bug |
| **Source** | Tier 0 notes in `95-backlog-and-priorities.md` |

## Why this exists

Small, real, individually-not-worth-a-ticket issues found in passing and recorded rather than fixed. One
is confirmed data loss, which is why this sits before alpha.

## The confirmed one — address component kinds

`AddressFields` / `toLegacyContact` (`frontend/src/api/contacts.ts`) round-trip only **five** address
component kinds: `street`, `locality`, `region`, `postcode`, `country`.

JSContact/vCard allow more (`apartment`, `floor`, `district`, `building`, `room`, `landmark`, …). A
CardDAV-imported or JSContact-imported address using any of those has them **silently dropped the next
time the contact is edited and saved through the UI** — the round-trip through the flat editing shape
loses them.

Narrow (only affects externally-imported addresses with non-standard structure) but real, and it is
genuine data loss.

**The fix belongs in the adapter (`api/contacts.ts`), not in the components.** Either preserve unknown
component kinds through the flat editing shape (carry them alongside and re-emit on save), or render them
as additional fields. Preserving-without-rendering is the smaller change and stops the bleeding.

## Also sweep for

While here, look for the same *class* of bug — anything that narrows a rich neutral-model structure into
a flat editing shape and writes it back:

- Other `Card` sections with more structure than the editing UI exposes (check `contactmodel.Card`
  against what `ContactInformation.tsx` / `AddContactDialog.tsx` actually render).
- `Passthrough.VCard` — verify it genuinely survives an edit-and-save cycle.
- `Contact.VCardExtra` — its own doc comment says `Passthrough` supersedes it "in spirit"; confirm
  nothing reads it as authoritative. (Full audit is [T22](19-T22-legacy-audit.md); just note findings.)

## Done when

- A test proves an address with a non-standard component kind survives a full
  import → edit → save → export round trip.
- Hand-verified: revert the fix, confirm the test fails, restore.
- `npx tsc --noEmit` clean, `npx vitest run` green; `go test ./...` green if the backend was touched.
- Anything else found is either fixed here (if trivial) or written up in
  `95-backlog-and-priorities.md` — do not fix silently and do not drop findings.
