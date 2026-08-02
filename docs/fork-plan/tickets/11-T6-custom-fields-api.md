# T6 — Custom fields v2: API surface

| | |
|---|---|
| **Rating** | 3 — nice to have, likely used |
| **Size** | M |
| **Depends on** | — |
| **Alpha** | before |
| **Source** | WP-84b, `94` (full spec — read it) |

## Why this exists

Custom fields v2 (`FieldDefinition` + typed `FieldValue`) is one of the five entities built but
unreachable: model, validation, standards projection, and a migration CLI all exist and are tested, but
there are **no routes and no frontend**. The untyped v1 (`User.CustomFieldNames` +
`Contact.CustomFields`) is still what actually runs.

## What exists today — none of this needs rebuilding

- `models.FieldDefinition` + `models.FieldValue` (`backend/models/field_definition.go`) — the
  schema/data two-part model from `94.3`. `FieldConstraints.Multi` makes any scalar type a validated
  list (it is not a separate type token).
- `services.ValidateFieldValue` — dispatches per `Type`, reusing `middleware.ValidateEmail` and the new
  `middleware.ValidateVar` primitive (which exposes the `phone`/`birthday`/`safeurl` custom validators
  plus the validator library's built-ins to a single runtime value rather than a tagged struct field).
- `projectCustomFields` in `models/contact_record.go` — projects `vcard:X-<NAME>`-mapped values into
  `Passthrough.VCard`, filtering `sensitivity='normal'` **in the query** per `91.13`. Only
  `internal-only` and `vcard:X-<NAME>` are implemented; a raw `jscontact:<pointer>` projection was
  deliberately **not** built (`Card.VCardProps` already carries `Passthrough.VCard` verbatim, so a
  `vcard:` projection already reaches vCard3, vCard4 *and* JSContact through one mechanism).
- `cmd/backfill-custom-fields` — the v1→v2 migration tool, dry-run/idempotent/fail-fast.
- Real-DB verified: an invalid `phone` is rejected, a valid one accepted, a `secret`-sensitivity
  `vcard:`-mapped field does not project.

## What to build

1. **`backend/controllers/field_definition_controller.go`** — CRUD for `FieldDefinition`, following
   `circle_controller.go`'s idiom. Ownership by `user_id`.
2. **`FieldValue` endpoints.** Decide the shape and say why in the doc comment: nested under the contact
   (`GET/PUT /contacts/:id/field-values`) reads more naturally for the frontend; a flat
   `/field-values?entity_id=` matches `life_event_controller.go`'s newer idiom. Either is defensible —
   pick one and be consistent.
3. **Wire validation into the write path** — every `FieldValue` write goes through
   `services.ValidateFieldValue` against its definition. A type mismatch is a `400`, not a 500.
4. **Respect sensitivity on read** — non-`normal` values must not leak into any response that leaves the
   instance. Follow how `projectCustomFields` already filters.
5. **Routes** in `backend/routes/routes.go`.
6. **Do not touch v1 yet** — [T7](12-T7-custom-fields-frontend.md) retires it. Both coexisting briefly is
   deliberate.

## Traps

- `FieldDefinition.Key` is the stable identifier; `Label` is display. Do not key values by label.
- `FieldValue.Value` is `json.RawMessage` — a `Multi` field is a JSON array, a scalar is a bare JSON
  value. Validate accordingly.
- Deleting a `FieldDefinition` must handle its values (SQLite FK enforcement is on and the FK is declared
  `CASCADE` — verify that is actually what you want, rather than blocking the delete).
- Test against the real migrated schema, not `AutoMigrate`.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- Controller tests cover CRUD, ownership scoping (cross-user access denied), type validation rejection,
  and the `Multi` list case.
- A **real-DB test** (`database.InitDB`) round-trips a definition + value through the HTTP handlers.
- Hand-verified: break the validation dispatch, confirm the rejection test fails, restore.
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
- `FieldConstraints.Multi` makes any scalar type a list — it is NOT a separate type token. Check `models/field_definition.go` for the exact struct.
- `FieldValue.Value` is `json.RawMessage` — a Multi field is a JSON array like `["a","b"]`, a scalar is `"value"`. Validate accordingly.
- `services.ValidateFieldValue` already dispatches per Type — reuse it, don't reimplement.
- Deleting a FieldDefinition: check `field_definition.go` for FK constraint. If CASCADE, values auto-delete. If not, you must clean up manually.
- Endpoints: pick between nested (`/contacts/:id/field-values`) or flat (`/field-values?entity_id=`). The ticket says either is defensible — pick one and STATE the choice in the controller's doc comment.
- `projectCustomFields` in `contact_record.go` already filters sensitivity in the query — study it before building the read path.
- Test: cross-user access denied, type validation rejection, Multi list round-trip, sensitivity filtering
