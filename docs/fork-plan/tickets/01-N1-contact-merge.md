# N1 — Contact merge / dedupe for existing contacts

| | |
|---|---|
| **Rating** | 5 — practically necessary |
| **Size** | M |
| **Depends on** | — |
| **Alpha** | before |
| **Source** | New (gap found in the 2026-07-30 product review vs. Monica) |

## Why this exists

Duplicate detection exists **only at import time**. There is no way to merge two contacts already in the
database. Duplicates arrive continuously from CardDAV sync, repeat imports, and manual entry, and today
the only remedy is deleting one by hand — which destroys everything attached to it.

Pre-alpha not because building it gets harder later, but because *not having it* during alpha produces
duplicate sprawl across real data that then has to be untangled manually. The cost lands on the data.

## What exists today

- `services.DetectDuplicate(db, userID, firstname, lastname, email, phone) *models.DuplicateMatch`
  (`backend/services/import_service.go`) — finds a likely existing match. Reusable as-is.
- `services.MergeImportedContact(existing, incoming *models.Contact)`
  (`import_service.go:999`) — field-level merge with a pinned policy: **incoming wins if non-empty,
  existing survives if blank**, uniform across scalar and array fields. Verified and locked by
  `TestMergeImportedContact_*` in `services/import_service_merge_test.go`.
- `services.CreateMergeNote(...)` (`import_service.go:1088`) — writes an audit note recording what a
  merge changed. Reuse it so a merge is traceable.
- `controllers.DeleteContact` (`backend/controllers/contact_controller.go`) — enumerates **every**
  dependent table. This is your checklist for what must be re-pointed.

## What to build

1. **`POST /contacts/:id/merge`** (or `/contacts/merge` with both IDs in the body) taking a **keep** ID
   and a **merge** (loser) ID. Both must be scoped to `user_id`; reject if equal or either is missing.
2. **Field resolution** — see **Merge semantics** below; it is the heart of this ticket. Persist via
   `ApplyRecordToContact` so `BeforeSave` runs (see `/CLAUDE.md` trap 2). Do **not** mutate `Card`/`CRM`
   directly.
3. **Re-point every association off the loser onto the keeper, inside one `db.Transaction`:**
   - `notes.contact_id`
   - `activity_contacts.contact_id` (many-to-many — **dedupe**: if the keeper is already on that
     activity, delete the loser's row instead of re-pointing, or you violate the join's uniqueness)
   - `reminders.contact_id`, `reminder_completions.contact_id`
   - `relationship_edges` — **both** `source_id` and `target_id`, matched on `Contact.VCardUID`, not the
     numeric ID. Then drop any edge that has become a self-loop (`source_id == target_id`) and any exact
     duplicate edge the merge created.
   - `household_members.member_vcard_uid`, `circle_members.member_vcard_uid`,
     `contact_tags.contact_vcard_uid` — all keyed by VCardUID; dedupe against their unique constraints.
   - `field_values.entity_id` (VCardUID)
   - `contact_sync_links.contact_id` — genuine FK. **Decide:** two sync links pointing at one contact
     from the same subscription is invalid; likely delete the loser's link and let the next sync
     reconcile.
4. **Delete the loser** using the existing `DeleteContact` cleanup path *after* re-pointing, so nothing
   is left orphaned.
5. **Write a merge note** on the keeper via `CreateMergeNote` so the merge is auditable.
6. **Frontend**: a merge entry point (contacts list multi-select, or a "find duplicates" view driven by
   `DetectDuplicate`), plus a preview showing which value wins per field before confirming.

## Merge semantics

**The guiding principle: in a merge, prefer keeping too much over losing something.** A stale extra email
is one click to delete afterwards; a value silently discarded during the merge is unrecoverable. Every
rule below follows from that.

### ⚠ Do not reuse `MergeImportedContact` wholesale

Its policy — *"incoming wins if non-empty, existing survives if blank"* — was written for **imports**,
where one side is authoritative. For a user-driven merge it is wrong in a specific way: applied to
multi-valued fields it **overwrites** the array, so merging a contact that has `home@` with one that has
`work@` would leave you with only one of them. That is data loss, and it is the exact opposite of what a
user means by "merge."

Write merge-specific field resolution. `MergeImportedContact` stays untouched for the import path; its
tests (`services/import_service_merge_test.go`, pinned by Tier 3c item 11c) must stay green.

### Resolution by field class

| Class | Fields | Rule | User choice? |
|---|---|---|---|
| **Multi-valued** | `Emails`, `Phones`, `Addresses`, `URLs`, `IMPPs`, and their `Card` equivalents | **Union**, de-duplicated on (type, value) — case-insensitively for email | **No** |
| **Associations** | notes, activities, reminders, relationship edges, circle/tag/household memberships, life events, field values | **Union** (re-pointed in step 3) | **No** |
| **Scalars, one side empty** | any of `Firstname`, `Lastname`, `Nickname`, `Gender`, `Birthday`, `Anniversary`, `Org`/`Department`/`JobTitle`/`Role`, `HowWeMet`, `FoodPreference`, `WorkInformation`, `ContactInformation`, prefix/middle/suffix | Take the non-empty one — **regardless of which contact it came from** | **No** |
| **Scalars, both non-empty and equal** | as above | Keep it; not a conflict | **No** |
| **Scalars, both non-empty and different** | as above | **This is the conflict set** — user picks | **Yes** |
| **Passthrough / unmapped** | `Passthrough.VCard`, `VCardExtra` | Keep the keeper's; merging opaque vendor data is not safe to guess at | No — but say so in the UI |

**Why this keeps the ticket M:** the conflict set is computed, not the whole model. In a real duplicate
pair it is typically a handful of fields and often zero — because duplicates usually arise from one
sparse record and one full one, where every field falls into the "one side empty" row above. The picker
renders a short list, not a side-by-side of the entire nested `Card`.

### The API shape this implies

The merge is **two calls**, not one:

1. `GET`/`POST` a **preview** returning: the computed merge result, plus an explicit `conflicts` array —
   one entry per genuinely-conflicting scalar with both candidate values and the field's display label.
2. The **commit** call takes the keep/loser IDs plus a resolution map (`field → chosen value`) for those
   conflicts. Reject the commit if a conflict is unresolved rather than silently defaulting, so a
   race (someone edits a contact between preview and commit) cannot quietly pick for the user.

### Explicitly out of scope

A full side-by-side "review and choose every field" UI. If that turns out to be wanted, it is a separate
follow-up (N1b) — the conflict-only picker is what keeps this ticket M, and it delivers the same control
for every case that actually needs it.

## Traps

- `LifeEvent.RelatedEntityIDs` is a **JSON array** of VCardUIDs, not a join table — a plain SQL update
  will not reach it. Either handle it explicitly or document it as a known limitation (Tier 3c item 1
  accepted the same limitation for deletes).
- **De-duplicate on union.** Merging two records that both hold the same phone number should not produce
  it twice. Compare on normalised value, not raw string.
- **Relationship edges need semantic de-duplication, not just row de-duplication** — after re-pointing,
  the keeper may hold two edges expressing the same fact, and one may be the *inverse* of the other
  (`A parent_of B` and `B child_of A`). Only one direction is ever stored, so use
  `relationship_type_registry.go`'s `InverseRelationType` to spot these, not string equality.
- Re-pointing before deduping will hit unique constraints on `circle_members`, `contact_tags`, and
  `household_members`. Dedupe first or use `INSERT OR IGNORE` semantics.
- SQLite FK enforcement is **on** (`database.openDSN`'s `foreign_keys(1)`), so ordering matters.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- A **real-DB test** (`database.InitDB` against a `t.TempDir()` file, not `AutoMigrate`) seeds a contact
  pair with *every* association type above, merges them, and asserts: zero orphan rows in every table,
  no self-loop or duplicate edges, the keeper holds the union of associations, and the loser is gone.
- Tests for each row of the resolution table: multi-valued fields **union rather than overwrite** (the
  regression this ticket exists to avoid); a scalar present on only the loser is kept; identical scalars
  are not reported as conflicts; genuinely differing scalars **are**; a commit with an unresolved
  conflict is rejected.
- A test proving an edge and its inverse do not both survive the merge.
- Hand-verified: change the multi-valued rule from union to overwrite, confirm that test fails, restore.
  That is the specific mistake reusing `MergeImportedContact` would have caused.
- Frontend `npx tsc --noEmit` clean and `npx vitest run` green; merge exercised in a real browser.

## Open decision

**What wins on conflict.** `MergeImportedContact`'s "incoming wins" policy was written for imports. For a
user-driven merge, per-field choice in the preview UI is friendlier and arguably correct. Decide before
building the preview; if you keep the automatic policy, say so in the UI so it isn't surprising.
