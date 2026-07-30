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
2. **Field resolution** — call `MergeImportedContact(keep, loser)`, then persist via
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

## Traps

- `LifeEvent.RelatedEntityIDs` is a **JSON array** of VCardUIDs, not a join table — a plain SQL update
  will not reach it. Either handle it explicitly or document it as a known limitation (Tier 3c item 1
  accepted the same limitation for deletes).
- Re-pointing before deduping will hit unique constraints on `circle_members`, `contact_tags`, and
  `household_members`. Dedupe first or use `INSERT OR IGNORE` semantics.
- SQLite FK enforcement is **on** (`database.openDSN`'s `foreign_keys(1)`), so ordering matters.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- A **real-DB test** (`database.InitDB` against a `t.TempDir()` file, not `AutoMigrate`) seeds a contact
  pair with *every* association type above, merges them, and asserts: zero orphan rows in every table,
  no self-loop or duplicate edges, the keeper holds the union of associations, and the loser is gone.
- Hand-verified: remove one re-point step, confirm the test fails, restore.
- Frontend `npx tsc --noEmit` clean and `npx vitest run` green; merge exercised in a real browser.

## Open decision

**What wins on conflict.** `MergeImportedContact`'s "incoming wins" policy was written for imports. For a
user-driven merge, per-field choice in the preview UI is friendlier and arguably correct. Decide before
building the preview; if you keep the automatic policy, say so in the UI so it isn't surprising.
