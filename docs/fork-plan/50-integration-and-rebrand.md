# 50 — Integration phases P1–P4 + rebrand (file-level work packages)

P0 (docs 10–40) delivers the isolated, tested converter core. P1–P4 wire it into the app. These are
specified at file granularity; each WP is still one sub-agent task with a compile+test gate.

## WP-70 · P1 — Persistence swap + migration  (effort L)

**Goal:** store the neutral `Record` as JSON + projected columns; migrate existing flat rows losslessly.

- `backend/models/contact.go` — replace the flat field set with:
  - `Card datatypes.JSON` (or `string` + `serializer:json`) holding `contactmodel.Record.Card`.
  - `CRM datatypes.JSON` holding `contactmodel.CRMEnvelope`.
  - `Passthrough datatypes.JSON`.
  - Keep/keep-syncing **projected columns** (`Firstname,Lastname,FN,Email,Phone,Birthday,Org,...`) — set
    in an updated `BeforeSave` by calling `contactmodel.DeriveProjection`.
  - Keep `UserID, VCardUID, ETag, Photo, PhotoThumbnail, Archived`, relationships/notes/reminders FKs.
- `backend/models/dtos.go` — `ContactInput`/`ContactResponse` express the neutral `Record` shape
  (or accept JSContact directly — decided in P2).
- `backend/database/migrations/000023_neutral_contact_model.up.sql` / `.down.sql`:
  - add `card`, `crm`, `passthrough` TEXT columns; keep legacy columns during transition.
  - **data migration** (Go migration or SQL + a one-shot `cmd/migrate-contacts`): read each flat row →
    build a `Record` via the *existing* `carddav.VCardToContact`-equivalent field-by-field mapping →
    `contactmodel` → serialize into `card`/`crm`. Must be **idempotent** and **round-trip-verified**
    (a test loads a sample of migrated rows and re-derives projections equal to the originals).
- `backend/models/contact_test.go` — projection + BeforeSave tests.

**Risk/guard:** irreversible; gate the migration behind a dry-run mode that reports diffs before write.

## WP-71 · P2 — API + import/export  (effort M)

- `backend/controllers/contact_controller.go` — Create/Update/Get accept & emit the neutral model
  (JSON). Add content negotiation for exports.
- `backend/controllers/export_controller.go` — add:
  - `GET /export/jscontact` → `jscontact.Adapter.Export` (one Card per contact, or a Card set).
  - `GET /export/vcf?version=4|3` → `vcard4`/`vcard3` adapter (user-selectable; default per config).
  - keep the combined-CSV path.
- `backend/services/import_service.go` — route VCF import through `vcard4`/`vcard3` (sniff VERSION) and
  add JSContact JSON import through `jscontact.Adapter`. Surface `Diagnostic`s in the import preview.
- `backend/routes/routes.go` — register the new export routes.
- Validation: replace ad-hoc validators with neutral-model validation (enum membership from the
  registry enums; graceful, non-strict on unknowns).
- Tests: controller round-trips (`POST` neutral → `GET vcf?version=4` contains expected lines).

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
