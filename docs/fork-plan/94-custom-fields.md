# 94 — User-defined custom fields (schema extensibility)

> A way for a **user** to add typed, validated, optionally-standards-mapped properties to entities
> **without a code fork**. The motivating insight: if Meerkat had this, gender identity and pronouns
> (the whole reason this fork started, issue #193) could have been added by a user *defining a field*
> rather than us *forking the schema*. Extensibility like this is what keeps the project from needing a
> fork every time someone wants to track one more thing.

## 94.1 Relationship to what exists (this is a generalization, not a new bolt-on)

Meerkat already ships an **untyped v1** of exactly this:
- `models.User.CustomFieldNames []string` — a per-user list of custom field *names* (untyped strings).
- `models.Contact.CustomFields map[string]string` — per-contact key→value, **string-only**, no
  validation, no standards mapping.

This doc's system is that v1, **generalized** to: typed values, per-field validation/constraints, and an
optional **standards-projection rule**. Existing data migrates directly (94.6).

## 94.2 Where it sits architecturally (native vs. custom — do not conflate)

Two tiers, deliberately kept distinct:
- **Native fields** — RFC-modeled, first-class, in `contactmodel.Card` (name, emails, pronouns,
  anniversaries, …). These are the concepts the *standards* define; we model them natively and richly.
  **Do not reimplement native fields as custom fields** — that would throw away the RFC-native modeling
  P0 was built for.
- **Custom fields** — the user's escape hatch for concepts the standards *don't* define but a particular
  user needs. Envelope-side (CRM extension), user-scoped.

This is the same hub-vs-envelope split from `90`, one level up: native modeling for standardized
concepts, user-defined fields for user-specific ones.

**Clean reuse — custom fields and Passthrough are two ends of one spectrum.** P0's
`Record.Passthrough.VCard []JCardProp` already round-trips *unknown* vCard properties opaquely. A custom
field mapped to a vCard `X-` property is essentially **"a passthrough property the user has named, typed,
and promoted to first-class"** instead of leaving it opaque. So a custom field's standards projection
rides the *same* JCardProp/passthrough machinery we already built — no new export plumbing. (A nice future
Level-2 feature falls out: on import, an unrecognized `X-` property could be *offered* to the user as a
candidate field definition — out of scope here, but the mechanism supports it.)

## 94.3 Two-part model

**FieldDefinition** (the schema — user-defined, scoped by `UserID`):

| Field | Notes |
|---|---|
| `id` | UUID |
| `user_id` | owner (definitions are per-user, like `CustomFieldNames` today). |
| `label` | human name, e.g. "Gender Identity". |
| `key` | machine name, e.g. `gender_identity` (stable; the map key / API field name). |
| `target` | which entity the field attaches to — `contact` initially; model allows `relationship`/`household`/… later. |
| `type` | one of the value types in 94.4. |
| `constraints` | JSON, type-dependent: `{min,max}` for number; `{values:[…]}` for enum; `{maxLength}` for string; `{multi:true}` for list-of; `required?`. |
| `projection` | the standards-mapping rule (94.5), or `internal-only`. |
| `sensitivity` | optional; custom fields are subject to the cross-cutting sensitivity rule (`91.13`) — a `gender_identity` or `hiv_status` field a user marks sensitive is excluded-by-default from export/external/shared surfaces. |
| `display` | optional UX hints (group, order, help text) the P3 field-path registry consumes. |

**FieldValue** (the data — per entity):

| Field | Notes |
|---|---|
| `id` | UUID |
| `field_definition_id` | which definition. |
| `entity_id` | the Contact (or other target entity). |
| `value` | the typed value, stored as JSON (so number/bool/enum/list keep their type, not stringified). |

## 94.4 Type system (reuse existing validators)

Keep the type set small and lean on the validators already in `middleware/validation.go`
(`phone`, `email`, `safeurl`, `birthday`) plus the validator library's built-ins:

- `string` (with optional `maxLength`, `pattern`)
- `text` (long free text)
- `number` (int or decimal; optional `min`/`max`)
- `boolean`
- `date` (reuse `birthday`/partial-date validation) / `datetime`
- `uri` (reuse `safeurl`), `email` (reuse `email`), `phone` (reuse `phone`)
- `enum` (with an allowed `values` list; the pronouns/gender-identity case — a constrained vocabulary)
- `list<T>` — a multi-valued variant of any scalar type above (`{multi:true}`)

Validation on write is driven by the definition's `type` + `constraints`, routed to the matching
validator — so a user's `number` field with `{min:0,max:10}` is enforced server-side, not just in the UI.

## 94.5 Standards-projection rule

Each definition declares how (or whether) its value maps out:
- **`internal-only`** — default; never leaves the CRM (its own DB + internal API only).
- **`vcard: X-<NAME>`** — projects to a vCard `X-` extension property (RFC 6350 permits arbitrary `X-`
  properties). Emitted via the P0 `JCardProp`/passthrough path on vCard export; parsed back on import.
- **`jscontact: <pointer>`** — projects to a JSContact vendor/custom property (via the RFC 9555
  `JSPROP`/`vCardProps` escape hatches P0 already implements in `Passthrough`).

Projection is **subject to sensitivity** (94.3 / `91.13`): a field marked sensitive does not project even
if it has a mapping rule, unless the export context explicitly includes sensitive data.

**Worked example (the #193 case, no fork required):** a user defines
`{label:"Pronouns", key:"pronouns", type:list<string>, projection:"vcard: PRONOUNS"}` — and now pronouns
round-trip through vCard with zero code changes. (In our actual fork pronouns are a *native* field, richer
than this — but the point stands: a user could have added them themselves.)

## 94.6 Migration from the untyped v1

- Each `User.CustomFieldNames` entry → a `FieldDefinition{type:string, constraints:{}, projection:internal-only}`.
- Each `Contact.CustomFields[key]` value → a `FieldValue{value:<string>}` under the matching definition.
- Lossless and mechanical; users can later *upgrade* a definition's type/constraints/projection in place.

## 94.7 Roadmap placement

Backend (definitions + typed values + validation + projection) is independent of the relationship graph
and can land with the P5 core-model work (`92`). The **full UX** depends on P3's data-driven field-path
registry (`50-integration-and-rebrand.md` WP-72) — custom-field definitions make that registry partly
*data-driven* (user-defined fields appear alongside native ones). See `92` for the WP slot.
