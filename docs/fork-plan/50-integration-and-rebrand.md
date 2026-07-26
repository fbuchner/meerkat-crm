# 50 — Integration phases P1–P4 + rebrand (file-level work packages)

P0 (docs 10–40) delivers the isolated, tested converter core. P1–P4 wire it into the app. These are
specified at file granularity; each WP is still one sub-agent task with a compile+test gate.

## WP-70 · P1 — Persistence swap + migration  (effort L)

**Goal:** store the neutral `Record` as JSON alongside the existing flat fields; migrate existing rows
losslessly into it. **P1 does NOT remove or rename any existing `Contact` field, and does not touch
`backend/carddav`, `backend/controllers`, or `backend/services`.**

**Scope correction (binding — resolved before implementation, not after).** An earlier pass of this
section said "replace the flat field set." That is unsafe as literally read: at least 16 existing files
(`carddav/vcard_mapper.go`, `carddav/backend.go`, `controllers/{contact,export,activity,note,photo,
relationship,reminder,admin_user,graph}_controller.go`, `services/{import_service,import_session,
birthday_service,calendar_sync_service,mailer,reminder_service}.go`) read or write `Contact`'s current
granular fields (`Emails`, `Phones`, `Addresses`, `URLs`, `IMPPs`, `Prefix`, `MiddleName`, `Suffix`,
`Organization`, `Department`, `JobTitle`, `Role`, `Anniversary`, `Gender`, `Nickname`, `HowWeMet`,
`FoodPreference`, `WorkInformation`, `ContactInformation`, `Circles`, `CustomFields`, `VCardExtra`)
directly, by name, today. Removing those fields in this WP would break every one of those files to
compile — pulling P2's controller rewrite and P4's CardDAV rewire into what is supposed to be an
effort-L, models-and-migrations-only WP. **The actual, safe design: P1 is purely additive**, exactly like
P0 was — every existing field stays exactly as it is, fully functional, untouched. The new `Card`/`CRM`/
`Passthrough` columns are added *alongside* them and populated by the migration + an updated
`BeforeSave`, as a second, parallel representation of the same data. Nothing currently reading the old
fields needs to change or even notice this WP happened. The old fields (and `VCardExtra`, which
`Passthrough` supersedes in spirit but does not yet replace in code) become genuinely dead **only** once
P2 rewrites the controllers/services to read/write the neutral model instead — removing them is a later,
separate migration once `git grep` for each old field name in `controllers`/`services`/`carddav` comes
up empty. Do not remove a single existing field in this WP.

**Shared mapping function (binding — avoids two divergent implementations of the same logic).**
`BeforeSave` (runs per-save, needs a `Record` to call `DeriveProjection` on) and the migration backfill
tool (runs per-row at migration time, needs a `Record` to serialize into `card`/`crm`/`passthrough`)
both need to turn a `*Contact`'s *current, existing* fields into a `*contactmodel.Record`. Write this
**once** — `func RecordFromContact(c *Contact) *contactmodel.Record` in `backend/models/contact.go` (or
a small new file in the same package) — and have both `BeforeSave` and the migration tool call it. Do
not let the migration tool grow its own separate Contact→Record mapping; that is exactly the
"two-competing-code-paths" mistake this WP must avoid. Port the field-by-field logic from
`carddav/vcard_mapper.go`'s Contact↔VCard mapping (read-only reference — that file targets vCard, not
`contactmodel.Record`, so this is a new mapping, not a call into carddav) for the actual field
assignments (`Firstname`/`Lastname`→`Name.Components`, `Emails`→`Card.Emails`, etc.), guided by
`docs/fork-plan/20-correspondence.md`'s neutral-path column for what each concept's home is.

- `backend/models/contact.go` — **add** (do not remove anything):
  - `Card contactmodel.Card \`gorm:"column:card;type:text;serializer:json"\`` (and equivalently-tagged
    `CRM contactmodel.CRMEnvelope` / `Passthrough contactmodel.Passthrough`) — **use the existing
    `type:text;serializer:json` GORM pattern already used for `Circles`/`Emails`/`Phones`/etc. in this
    same file, not `gorm.io/datatypes.JSON`**: that package is not a current dependency (`go.mod` has no
    reference to it) and would be an unnecessary new dependency when the established in-repo convention
    already does exactly this job.
  - An updated `BeforeSave` that, **in addition to** its existing sync logic (Email/Phone/Address from
    the first array entry), also calls `RecordFromContact` + `contactmodel.DeriveProjection` and writes
    the result into the **existing** projected scalars
    (`Firstname`, `Lastname`, `Email`, `Phone`, `Birthday`) — i.e. `DeriveProjection` becomes the single
    source of truth for what the existing sync logic already does, not a second competing sync path.
    Also add two new nullable columns, `FN` and `Org` (both new `contactmodel.Projection` fields with no
    existing analog), populated the same way — additive, nothing currently reads them, no risk in adding
    them now rather than deferring.
- `backend/models/dtos.go` — **do not touch.** `ContactInput`/`ContactResponse` expressing the neutral
  `Record` shape is explicitly P2's job (per WP-71, which already owns the controller/DTO rewrite) — this
  was previously listed under P1 in error; moving it here would touch `controllers` before P1 is allowed
  to.
- `backend/database/migrations/000023_neutral_contact_model.up.sql` / `.down.sql`:
  - add `card`, `crm`, `passthrough` TEXT columns (nullable/empty-default) — legacy columns are
    untouched, not just "kept during a transition."
  - **data migration** (Go migration or SQL + a one-shot new command, `cmd/backfill-contact-records` —
    note `cmd/migrate/main.go` already exists and is the golang-migrate *schema*-migration CLI runner;
    this new command is for the *data* transform and is deliberately named differently to avoid
    confusion with it): read each row → call the same `RecordFromContact` used by
    `BeforeSave` → serialize into `card`/`crm`/`passthrough`. Must be **idempotent** (safe to re-run;
    skip rows where `card` is already populated unless a `--force` flag is passed) and
    **round-trip-verified** (a test loads a sample of migrated rows and re-derives projections equal to
    the originals via `contactmodel.DeriveProjection`).
- `backend/models/contact_test.go` — **new file** (none exists yet) — `RecordFromContact` +
  `DeriveProjection`-via-`BeforeSave` tests: a fully-populated `Contact` round-trips into a `Record` with
  every field landing in its correct neutral home (cite `20-correspondence.md` rows), and an empty/minimal
  `Contact` doesn't panic.

**Risk/guard:** the *data migration* is the irreversible part (once real rows are backfilled, a bad
transform could ship silently-wrong `Card` JSON) — gate it behind a dry-run mode that reports diffs
before writing. The *schema* change (adding three nullable columns) is trivially reversible via the
`.down.sql`. Adding fields to a Go struct and a new `BeforeSave` call, by contrast, carries no
compilation risk to other packages by construction, since nothing else references the new fields yet.

## WP-71 · P2 — API + import/export  (effort M)

**API versioning — binding decision.** Today's `/api/v1/contacts` serves the flat, pre-neutral-model
DTO. This WP changes that shape (nested `Card` instead of flat fields) — a breaking wire change. Decided:
**take the break under the existing `/api/v1` path, now, while the only consumer is the frontend being
co-migrated in P3.** There is no third-party or mobile consumer of the current shape yet, so paying the
cost of a parallel `/api/v2` + compatibility shim buys nothing. This is the last free breaking change:
once a native mobile client (below) ships against this shape, `/api/v1` is frozen and any future
incompatible change gets a real `/api/v2`. Record this precedent in code review going forward.

**Mobile-CRUD-real, not just mobile-CRUD-possible.** The client framework (native Swift/Kotlin vs.
Flutter/Dart) is explicitly undecided and does not need to be decided here — a REST+JSON API with an
OpenAPI contract serves all of them identically (`openapi-generator`/equivalents target Swift, Kotlin,
and Dart from the same spec). What must be concrete now, regardless of framework, is the endpoint shape
itself:

- `GET /api/v1/contacts` (list) — returns `[]ContactSummary`, **not** the full `Card`. `ContactSummary`
  wraps `contactmodel.DeriveProjection`'s fields (`Firstname, Lastname, FN, PrimaryEmail, PrimaryPhone,
  Birthday, Org`) plus the record's own identity (`UID`) and existing `Photo`/`PhotoThumbnail` — i.e. a
  new controller-layer DTO built *from* `Projection`, not `Projection` reused verbatim as a wire type
  (`Projection` stays an internal persistence-projection helper, not a public API shape). This is the
  endpoint a mobile contact-list screen calls; it must not require fetching every contact's full nested
  `Card` to render a scrollable list.
- `GET /api/v1/contacts/{id}` (detail) — returns the full neutral `Record`/`Card` (nested names,
  addresses, emails, phones, organizations, personalInfo, pronouns, everything) — the endpoint a
  detail/edit screen calls.
- `POST /api/v1/contacts` / `PUT /api/v1/contacts/{id}` — accept the full neutral `Record`/`Card` shape,
  same as `Get` detail's response shape (symmetric read/write contract).
- `backend/controllers/contact_controller.go` implements all four; add content negotiation for exports
  on the existing detail path.

**OpenAPI spec — new deliverable.** `backend/openapi.yaml` (hand-authored or generated via `swaggo/swag`
annotations, implementer's choice of tooling) covering at minimum: `ContactSummary`, the full `Card`
request/response schema, and the export endpoints below. This is what actually makes the endpoints
future-proof for a not-yet-chosen mobile framework — Swift/Kotlin/Dart client models can all be
generated from one spec once it exists, rather than hand-rolled per framework later. Validate the spec
itself (e.g. `swagger-cli validate` or equivalent) as part of this WP's test gate, not just that the
Go handlers compile.

- `backend/controllers/export_controller.go` — add:
  - `GET /export/jscontact` → `jscontact.Adapter.Export` (one Card per contact, or a Card set).
  - `GET /export/vcf?version=4|3` → `vcard4`/`vcard3` adapter (user-selectable; default per config).
  - keep the combined-CSV path.
- `backend/services/import_service.go` — route VCF import through `vcard4`/`vcard3` (sniff VERSION) and
  add JSContact JSON import through `jscontact.Adapter`. Surface `Diagnostic`s in the import preview.
- `backend/routes/routes.go` — register the new list/detail/export routes.
- Validation: replace ad-hoc validators with neutral-model validation (enum membership from the
  registry enums; graceful, non-strict on unknowns).
- Tests: controller round-trips (`POST` neutral → `GET vcf?version=4` contains expected lines);
  `GET /contacts` (list) asserts the response is the slim `ContactSummary` shape, not a full `Card`, and
  omits fields a mobile list view wouldn't need; OpenAPI spec validates and its schemas match the actual
  controller request/response shapes (a spec that silently drifts from the handlers is worse than none).

## WP-72 · P3 — Frontend remodel  (effort L/XL)

- `frontend/src/api/contacts.ts` — replace the flat `Contact` interface with a TypeScript mirror of
  `contactmodel.Record` (generate from Go if practical; else hand-write). Provide typed sub-objects.
- `frontend/src/contactFields.ts` — the `ContactFieldKey`-flat-key registry is replaced by a
  **field-path registry** describing nested/collection fields (path, cardinality, editor component,
  enabled-by-default). This unwinds the "one flat string key" assumption used across ~15 modules.
- Components (extend, don't rewrite where possible):
  - `MultiValueField.tsx` / `AddressFields.tsx` — already structured; generalize to drive any
    collection (emails/phones/onlineServices/addresses/pronouns/personalInfo/…).
  - `AddContactDialog.tsx`, `ContactInformation.tsx`, `ContactHeader.tsx`, `ContactDetailPage.tsx`,
    `ContactFieldSettings.tsx` — consume the new registry + nested model.
- New editors for nested concepts with no current UI: name components, speakToAs (pronouns +
  grammatical gender), anniversaries (partial dates + place), personalInfo (kind/level), localizations
  (advanced — can defer behind a raw-JSON editor initially).
- `frontend/e2e/contacts.spec.ts` — extend for the new fields (mirror the existing gender/pronoun
  scenarios).
- **Open UX question (flag to product):** presentation of arbitrarily nested / multi-valued /
  localized data. Ship a pragmatic subset first; keep unmapped-in-UI data visible via a read-only
  "advanced/raw" panel so the frontend never silently hides model data.

## WP-73 · P4 — CardDAV 3.0/4.0  (effort M/L)

- `backend/carddav/backend.go` — the two hardcoded `Version:"3.0"` capability advertisements (~lines
  110, 133) become version-aware; advertise 4.0 (and/or 3.0) per config/negotiation.
- `backend/carddav/vcard_mapper.go` — retire the 877-line ad-hoc mapper; route
  `ContactToVCard`/`VCardToContact` through `vcard4`/`vcard3` adapters. Keep the old tests as a 3.0
  compatibility guardian until parity is proven, then delete.
- Content negotiation: emit 4.0 by default, 3.0 for clients that require it (User-Agent / Accept /
  per-address-book config). Optionally serve JSContact (`application/jscontact+json`) where supported.
- **Optional** — revive the dead `CardDAVSync` sync-token table (`backend/models/carddav.go`) for
  efficient collection sync; today only ETag+If-Match exists.

## WP-74 · Rebrand (effort S/M, do last)

- Go module path `meerkat` → new name in `backend/go.mod` + all imports (`gofmt -r` / `goimports`);
  mechanical.
- App name / branding: `frontend/src` strings, `docs/`, `README.md`, `docker-compose*.yml` service
  names, image names in `.github/workflows/docker-publish.yml`, favicon/logo assets.
- New `README` section documenting the RFC 9553/9554 support and the vCard-version export selector.
- Point `origin` at the user's fork; the existing `docker-publish.yml` (tag-triggered `ghcr.io`
  images) works unchanged once the repo owner/name change — a `v*` tag publishes the rebranded image.

## Sequencing & gates

`P0 (WP-10..60) green, app untouched` → `WP-70 (+dry-run migration verified)` →
`WP-71 & WP-72 coordinated (API contract flips with the UI)` → `WP-73` → `WP-74`.
Each WP: `go build ./... && go test ./...` (backend) and `npm run build && npx tsc --noEmit` +
Playwright e2e (frontend) green before merge. Full-stack smoke via
`docker compose -f docker-compose.test.yml up -d --build --wait` then import a v4 vCard and a
JSContact JSON, export the other, verify.
