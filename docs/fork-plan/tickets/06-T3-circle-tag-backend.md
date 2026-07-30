# T3 — Circle/Tag backend call-site rewiring

| | |
|---|---|
| **Rating** | 4 — strong, frequent use |
| **Size** | S–M |
| **Depends on** | [T2](05-T2-circle-tag-triage.md) — the data must exist before readers move to it |
| **Alpha** | before |
| **Source** | WP-84c-ii |

## Why this exists

With real `Circle`/`Tag` rows created by T2, the backend sites still reading and writing the flat
`Contact.Circles` string array move onto the entities.

## The call sites

1. **`controllers/contact_controller.go` → `GetCircles`** — returns the distinct set of flat strings.
   Should return real `Circle` rows. Note T2's triage UI consumes this endpoint; if you change its shape,
   check T2 still works (or version it).
2. **`controllers/contact_controller.go` → circle filtering in `GetContacts`** — currently:
   ```go
   query.Where("EXISTS (SELECT 1 FROM json_each(contacts.circles) WHERE json_each.value = ?)", circle)
   ```
   appearing **twice** (the main query and the count query — keep them in sync, that is a real trap).
   Becomes a join through `circle_members` on `Contact.VCardUID`.
3. **`services/import_service.go` → circles/tags/groups/labels synonym mapping.** This is the one that
   needs actual thought, not a find-and-replace: it currently maps **all four** vocabularies onto the
   single flat `circles` field. Now that `Tag` exists as a distinct destination, **the mapping has to
   split by target** — "groups" is Circle-shaped, "labels" is Tag-shaped — rather than just changing
   where it writes. Decide the mapping explicitly and document it in the function's doc comment.

## What to build

- Move each site above onto `Circle`/`Tag` + their membership tables.
- Keep `Contact.Circles` populated for now if anything else still reads it (T4 retires the field) —
  **or**, if T4 will land immediately after, coordinate and drop it once. Say which you chose.
- Add/adjust tests for each moved site.

## Traps

- Membership is keyed by **`Contact.VCardUID`**, not the numeric ID.
- The `json_each` filter exists in **two** places in `GetContacts`; missing the count query gives correct
  results with a wrong total.
- SQLite FK enforcement is on, and `DeleteCircle`/`DeleteTag` rely on DB-level `CASCADE` for their member
  rows (this was found and fixed in Tier 3c item 8) — do not add manual member deletion that would double
  up.
- Import merge policy is pinned by tests (`TestMergeImportedContact_*`); if the circles field changes
  shape, those tests change with it — update them deliberately, do not just make them pass.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- A **real-DB test** proves contact filtering by circle returns the same results through the new join as
  the old `json_each` query did.
- A test covers the import synonym split: a "groups" column and a "labels" column land in Circle and Tag
  respectively.
- Hand-verified: break the join condition, confirm the filter test fails, restore.
