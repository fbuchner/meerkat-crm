---
name: add-contact-field
description: Add, rename or remove a stored property of a contact end to end — migration, GORM model, input DTO, validator, controller allowlist and assignments, Monica import mapping, frontend field registry, detail-page allowlist, form rendering, all five locale files, and tests on both sides. Use for any request that changes what a contact stores, such as "add a pronouns field", "expand the gender options", "add a second anniversary date", or "contacts need a preferred-language field".
---

# Adding a contact field

A contact field touches eleven places. Skipping one usually does not fail a test — it
loses data silently. Work the list in order and tick each item off explicitly.

The **deceased** field is the complete worked reference, including its mistakes:

```bash
git log --oneline 2aa841f~1..348a46f    # the feature, then four corrective commits
git show 000024_add_contact_deceased    # or: grep -rn "deceased" backend frontend/src
```

## Checklist

### Backend

1. **Migration** — `cd backend && make migrate-create NAME=add_contact_<field>`.
   Write both directions. SQLite via `ALTER TABLE`, booleans as `INTEGER DEFAULT 0 NOT NULL`,
   dates and text as `TEXT DEFAULT ''`:
   ```sql
   -- up
   ALTER TABLE contacts ADD COLUMN is_deceased INTEGER DEFAULT 0 NOT NULL;
   ALTER TABLE contacts ADD COLUMN deceased_date TEXT DEFAULT '';
   -- down
   ALTER TABLE contacts DROP COLUMN is_deceased;
   ```
   If existing rows need a value, that is a *second*, separate backfill migration — keep the
   schema change and the data change in different numbered pairs.

2. **`backend/models/contact.go`** — add the field to `Contact` with its `gorm`, `json` and
   `validate` tags. Match the surrounding style and keep it in the right commented group.

3. **`backend/models/dtos.go`** — add it to the contact *input* DTO with the same `json` and
   `validate` tags. Easy to forget; without it the API accepts the field and drops it.

4. **`backend/middleware/validation.go`** — only if it needs a rule the stock validators do
   not cover. Two edits, both required:
   - register it in the `validate.RegisterValidation(...)` block
   - add a `case "<tag>":` to the error-message switch, or the user sees a raw tag name

5. **`backend/controllers/contact_controller.go`** — **three** separate edits:
   - add the column to `contactSelectableFields` (see the trap below)
   - assign it in the create path
   - assign it in the update path

6. **`backend/services/monica_import_service.go`** — map it if Monica carries an equivalent
   (`backend/monica/types.go` lists what the Monica API returns).

7. **Other services** — grep for places that must respect it. A field like "deceased" has to be
   excluded from birthday reminders (`services/birthday_service.go`); a new flag may need the
   same treatment.

### Frontend

8. **`frontend/src/contactFields.ts`** — add the key to the `ContactFieldKey` union and an
   entry to `CONTACT_FIELDS` with its `labelKey` and `group`. Add it to
   `DEFAULT_ENABLED_CONTACT_FIELDS` only if existing users should see it without opting in.

9. **`frontend/src/ContactDetailPage.tsx`** — add the column to the `CONTACT_FIELDS` array.
   **This is the data-loss trap. See below.**

10. **`frontend/src/components/ContactInformation.tsx`** — render it. Dates go through
    `DateFormatProvider`, never `toLocaleDateString` or a hand-rolled format. Types live in
    `frontend/src/types/`.

11. **All five locales** — `en.json`, `de.json`, `fr.json`, `es.json`, `it.json` under
    `frontend/src/i18n/locales/`. English alone is an incomplete change; the `PostToolUse`
    hook reports any locale whose keys drift from `en.json`.

### Tests

Both sides, in the same change: `backend/middleware/validation_test.go` for a new validator,
`backend/controllers/` for create/update round-trips, and
`frontend/src/components/ContactInformation.test.tsx` for rendering. A round-trip test that
saves, re-fetches and asserts the value survived is the one that catches trap #1.

## The two-allowlist trap

There are two independent column allowlists with different names in different languages:

| Where | Name |
|---|---|
| `backend/controllers/contact_controller.go` | `contactSelectableFields` |
| `frontend/src/ContactDetailPage.tsx` | `CONTACT_FIELDS` |

The detail page sends its list as the `fields` query param and the backend returns only those
columns. Omit a field from **either** and the `GET` returns a zero value, which overwrites the
real value in local state, and the next save persists that wipe. Permanent data loss, with every
test still green. This shipped once (`348a46f`) — check both lists before you call the work done.

## Verify

```bash
cd backend   && go vet ./... && go test ./...
cd frontend  && yarn tsc --noEmit && CI=true yarn test
```

Then apply the migration and round-trip the field through the real app: create a contact with a
value, reload the detail page, edit an unrelated field, save, and confirm the value is still
there. That sequence is what the allowlist bug breaks and what unit tests miss.
