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
