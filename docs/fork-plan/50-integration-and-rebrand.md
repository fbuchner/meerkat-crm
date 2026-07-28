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

**Pre-implementation review (binding — resolved before implementation, not after).** A review of the
*actual* current controllers/services against this section found five real gaps between what this doc
said and what exists, resolved below before any code is written (same discipline as WP-70's scope
correction). **Scope for this pass: WP-71 only.** The plan's own sequencing note already says WP-71 and
WP-72 (frontend) should be *coordinated* because this WP's contract change breaks the current frontend —
that break is accepted and deliberate for this pass (the frontend currently parses a flat `Contact`
shape; it will not build/run correctly against these endpoints until WP-72 lands separately, which should
follow reasonably soon, not be indefinitely deferred).

**Gap 1 — reverse mapping is required and was missing from this doc.** WP-70 built
`RecordFromContact` (flat `Contact` → neutral `Record`) for `BeforeSave`/migration — read-direction only.
`CreateContact`/`UpdateContact` today construct/mutate the flat `Contact` struct directly from the flat
`ContactInput` DTO; there is no code anywhere going `Record`/`Card` → flat fields. Since these endpoints
must now *accept* the neutral shape, the reverse mapping is required. **Binding: write it once** —
`func ApplyRecordToContact(c *Contact, r *contactmodel.Record)` (mirroring `RecordFromContact`'s
placement in `backend/models/`) — populating every flat legacy field from the equivalent `Record`/`Card`
field, so `BeforeSave`'s existing sync logic, `carddav`, and every other reader of the flat fields keep
working unmodified during this transitional period. `CreateContact` and `UpdateContact` both call this
one function — do not let either grow its own `Record`→`Contact` mapping (the exact anti-pattern WP-70
avoided on the read side, applied here to the write side). **This function is also the natural point of
reuse for Gap 4's VCF-import merge path** (see below) — a second reason it must be one shared function.

**Gap 2 — `GET /contacts` (list) already has a feature set this doc didn't mention; preserve it.**
The current `GetContacts` has offset pagination (`page`/`limit`/`total`), multi-field free-text search
(including into JSON array columns), sort (`firstname`/`lastname`/`id`/`random`), archive filtering,
circle filtering, and an `includes=` param preloading `notes`/`activities`/`relationships`/`reminders`.
**Binding: keep every existing query parameter and mechanic exactly as-is — only the per-item JSON shape
changes**, from flat `Contact`/`ContactResponse` to `ContactSummary`. For `includes=`: since `ContactSummary`
has no room for full sub-resource arrays, when `includes` is requested the response item is an
*extended* summary — `ContactSummary` plus the requested `notes`/`activities`/`relationships`/`reminders`
arrays as additional optional fields — not a full `Card`. Do not drop search/sort/filter/pagination/includes
to simplify the list endpoint.

**Gap 3 — the `fields=` partial-projection param is deprecated, not silently broken.** Both
`GET /contacts` and `GET /contacts/:id` support `fields=` today (operating on flat column names), and the
current frontend calls it (`frontend/src/api/contacts.ts`). **Binding: remove `fields=` handling from
both endpoints in this WP** — the fixed `ContactSummary` (list) / full `Card` (detail) shapes now serve
the reason `fields=` existed (avoiding over-fetch on a list view). This is a deliberate, documented
removal, not an oversight; the frontend must stop calling it (coordinate with WP-72, per the accepted
scope above).

**Gap 4 — VCF import: swap the parser, preserve the merge/duplicate-detection UX.** VCF import currently
routes through the legacy `carddav.VCardToContact` (confirmed by reading `services/import_service.go`'s
`ParseVCF`), with real UX on top: `DetectDuplicate`, `MergeImportedContact`, `CreateMergeNote`,
`ContactToPreviewMap`/`ValidateImportedContact`. **Binding: swap the parser to the `vcard4`/`vcard3`
adapters (sniffing `VERSION`, per this doc's original ask), and adapt the merge/duplicate-detection
functions to operate on the resulting `contactmodel.Record` fields instead of flat `Contact` fields** —
do not leave a second, divergent VCF-import path on the legacy mapper. Concretely: an imported `Record`
is turned into a candidate `Contact` via `ApplyRecordToContact` (Gap 1's function) applied to a fresh/
existing `Contact`, and duplicate-detection/merge logic reads from the resulting flat fields as it does
today (or from the `Record` directly where that's cleaner — implementer's judgment, but do not fork the
mapping). **CSV import needs no equivalent change** — it already builds a flat `Contact` via
`BuildContactFromRow` and `db.Create`/`db.Save`, which already populates `Card`/`CRM`/`Passthrough` for
free via WP-70's `BeforeSave` hook.
Note this creates an intentional, temporary divergence: REST import/export (this WP) uses the new
adapters while the live CardDAV server (`carddav/backend.go`) still uses the legacy mapper until WP-73 —
expected, not a conflict, given the phasing.

**Gap 5 — none; noted for completeness.** `GET /contacts/circles` (`GetCircles`) reads the flat `Circles`
field today and is correctly out of scope until WP-84 (P5, already deferred) splits Circle/Tag — not a
WP-71 concern, just confirmed not forgotten.

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
  `Card` to render a scrollable list. **Per Gap 2/3 above: every existing query mechanic
  (`page`/`limit`, `search`, `sort`/`order`, `include_archived`/`archived`, `circle`, `includes=`) is
  preserved unchanged — only the per-item shape changes.** `includes=` extends `ContactSummary` with the
  requested `notes`/`activities`/`relationships`/`reminders` arrays rather than upgrading to a full
  `Card`. `fields=` is removed (Gap 3), not preserved.
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
  omits fields a mobile list view wouldn't need; **and** asserts pagination/search/sort/archive/circle
  filtering and `includes=` still work exactly as before against the new item shape (Gap 2), and that
  `fields=` is gone (Gap 3); `ApplyRecordToContact`/`RecordFromContact` round-trip test (write a Record,
  read it back, get the same Record) exercising Gap 1; VCF-import duplicate-detection/merge test against
  the new adapter path (Gap 4); OpenAPI spec validates and its schemas match the actual controller
  request/response shapes (a spec that silently drifts from the handlers is worse than none).

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

**Pre-implementation review (binding — resolved before implementation, not after).** Same discipline as
WP-70/71. One real gap found by reading the actual current code, and it must be fixed *before* retiring
the legacy mapper, not after:

**Prerequisite — photo bridging gap (fix first, in its own small step).** Neither `RecordFromContact` nor
`ApplyRecordToContact` (WP-70/71) touch `Contact.Photo`/`PhotoThumbnail` (disk-based) ↔ `Card.Media`
(kind=photo) at all — WP-71 already hit this and documented it as a known, accepted limitation for
one-off VCF/JSContact export. It cannot stay unfixed here: retiring `vcard_mapper.go` for the **live**
CardDAV server without this bridge means every synced device silently loses contact photos in both
directions (export: the server stops serving photos it used to; import: a photo uploaded via CardDAV
`PUT` parses into `Card.Media` but is never persisted to disk or `Contact.Photo`, so the app's own UI —
which reads `Contact.Photo`/`PhotoThumbnail` directly, not `Card.Media` — never shows it). This is a much
larger blast radius than the VCF-export edge case WP-71 accepted.
The disk-I/O helpers that already do this correctly (`readContactPhoto`, `SaveContactPhoto`,
`extractPhotoData`, `FetchPhotoFromURL`, all in `carddav/vcard_mapper.go`) cannot simply be called from
`backend/models/` — `carddav` already imports `models` (confirmed: `carddav/auth.go`, `carddav/backend.go`
both import `meerkat/models`), so `models` importing `carddav` back would be a real import cycle.
**Binding: extract these four functions into a new, dependency-free package** (e.g. `backend/photostore/`)
that `models`, `carddav`, and `services` can all import without a cycle. Then extend `RecordFromContact`
(add a `photoDir string` parameter; read the on-disk photo, encode as a `data:` URI or file reference into
one `Card.Media` entry with `Kind:"photo"`) and `ApplyRecordToContact` (same parameter; on the reverse
path, find the `photo` `Media` entry, persist it via the extracted `SaveContactPhoto`, set
`Contact.Photo`/`PhotoThumbnail`) to call the new package. This also retroactively closes WP-71's
documented VCF/JSContact photo-export gap — update `export_controller.go`'s call sites to pass
`photoDir` once this lands, and remove the "known limitation" comment there since it's no longer true.

- `backend/carddav/backend.go` — the two hardcoded `Version:"3.0"` capability advertisements (~lines
  110, 133) become version-aware; advertise 4.0 (and/or 3.0) per config/negotiation.
- `backend/carddav/vcard_mapper.go` — retire the 877-line ad-hoc mapper; route
  `ContactToVCard`/`VCardToContact` through `vcard4`/`vcard3` adapters (now photo-complete, per the
  prerequisite above). Keep the old tests as a 3.0
  compatibility guardian until parity is proven, then delete.
- Content negotiation: emit 4.0 by default, 3.0 for clients that require it (User-Agent / Accept /
  per-address-book config). Optionally serve JSContact (`application/jscontact+json`) where supported.
- **Optional** — revive the dead `CardDAVSync` sync-token table (`backend/models/carddav.go`) for
  efficient collection sync; today only ETag+If-Match exists.

## WP-73b · CardDAV *client* (sync contacts in from an external server)  (effort M — revised down from M/L; see library discovery below)

**Why this exists.** Upstream (`fbuchner/meerkat-crm`) is adding "the option to run Meerkat as a CardDAV
client" in an upcoming, not-yet-public release (per maintainer comment, PR #195:
https://github.com/fbuchner/meerkat-crm/pull/195#issuecomment-5083383141 — confirmed via `gh api`, no
branch/PR visible yet as of this writing). This is the *opposite* direction from WP-73 above: instead of
other apps connecting to Meerkat's address book, Meerkat connects *out* to a user's existing CardDAV
server (Nextcloud, Fastmail, iCloud, etc.) and pulls their contacts in.

**Do not wait for or plan to merge upstream's implementation.** The maintainer's own stated concern for
reviewing PR #195 is whether "the new classes" (from the pronouns/gramgender work) "interfere with"
their CardDAV-client code — meaning it touches the same area (`models/contact.go` and/or
`carddav/vcard_mapper.go`) this fork has already substantially diverged from (P0 replaced vCard parsing
entirely; P1 restructured the model; WP-71 is replacing the legacy mapper on the REST path). Per the
hard-fork decision (`90` D2), this is expected: build our own version using this fork's own architecture,
and only look at upstream's shipped code afterward for implementation ideas (real-world server quirks
they had to handle) — not as something to pull in directly.

**Decided: synced contacts are real `Contact` rows**, not `ExternalIdentity` references (per `91.12`'s
distinction — this is "we own this data going forward," unlike Immich-style "the other app owns it").

**Architecture — mirrors the existing CalDAV-client pattern exactly** (`services/calendar_sync_service.go`
+ `models.CalendarSubscription` + `calendar_event_links`), which already solves the same shape of problem
for calendars: subscription config, per-subscription sync-mutex, encrypted credentials
(`services/credential_crypto.go`, already exists, directly reusable), UID→local-record link table for
idempotent re-sync, content-hash change detection, sync status/error bookkeeping.

**Library discovery — this changes the implementation approach, not just an implementation detail.**
`github.com/emersion/go-webdav` (already a dependency — it's what powers our own CardDAV *server*) ships
a full CardDAV **client** in its `carddav` subpackage, confirmed by reading the library source directly
(not assumed): `carddav.DiscoverContextURL` (`.well-known/carddav`), `Client.FindCurrentUserPrincipal`,
`Client.FindAddressBookHomeSet`, `Client.FindAddressBooks`, `Client.QueryAddressBook`/
`MultiGetAddressBook`, `Client.GetAddressObject`/`PutAddressObject`, and — critically —
`Client.SyncCollection`, a **complete RFC 6578 sync-collection implementation already built in**
(`SyncQuery{SyncToken}` in, `SyncResponse{SyncToken, Updated []AddressObject, Deleted []string}` out —
`Deleted` already gives the reconciliation list directly). **Binding: use this client directly; do not
hand-roll WebDAV/XML/PROPFIND/REPORT construction.** This changes WP-73b from "build a CardDAV protocol
client" to "orchestrate an existing one" — materially lower effort/risk than originally scoped.
`AddressObject.Card` comes back as an already-parsed `vcard.Card` (the same low-level type our own
adapters use internally) — bridge to our adapters by re-encoding it to bytes (`vcard.NewEncoder`), then
sniffing `VERSION` and calling `vcard4`/`vcard3.Adapter{}.Import()` exactly as WP-71's VCF import does;
do not use `go-webdav`'s own parsed `vcard.Card` fields directly, so there is one vCard-interpretation
path (the P0 adapters + correspondence table), not two.

- `backend/models/contact_subscription.go` (new) — `ContactSubscription{UserID, Name, URL, Username,
  PasswordEncrypted, SyncEnabled, LastSyncedAt, LastSyncStatus, LastSyncError}`, same shape as
  `CalendarSubscription`.
- `backend/services/contact_sync_service.go` (new) — the sync loop, built on `carddav.NewClient` +
  `webdav.HTTPClientWithBasicAuth` (both already available): construct the client against the
  subscription's stored URL/credentials; if a stored sync-token exists, call `SyncCollection` directly
  (v1 may skip full discovery and use a user-supplied direct address-book URL, per below — discovery via
  `FindCurrentUserPrincipal`/`FindAddressBookHomeSet`/`FindAddressBooks` is a fast-follow); if the server
  doesn't support `sync-collection` (the library will surface an error/unsupported response — handle this
  as a fallback trigger), fall back to `FindAddressBooks` + `QueryAddressBook`/`MultiGetAddressBook` for a
  full refetch. For each `Updated` `AddressObject`: re-encode `.Card` to bytes, sniff `VERSION`, run
  through the `vcard4`/`vcard3` adapter to get a `contactmodel.Record`, then **reuse WP-71's
  `ApplyRecordToContact`** to turn it into a real `Contact` row — do not write a third, divergent
  `Record`→`Contact` mapping. Store each `AddressObject.ETag` in the link table. Persist the returned
  `SyncResponse.SyncToken` back onto the subscription for the next run.
- `contact_sync_links` table (new, mirrors `calendar_event_links`): `subscription_id, user_id,
  href/uid, contact_id, etag, content_hash`. For each path in `SyncResponse.Deleted`: look up the link,
  archive (not hard-delete, consistent with how `Contact.Archived` is already used elsewhere) the linked
  `Contact`, remove the link row.
- One-way (server→Meerkat) is the v1 target, matching CalDAV's current one-way-in shape. Two-way (push
  local edits back via the already-available `PutAddressObject`) is a later increment, same phasing logic
  as WP-88 for calendars.
- Multiple address-book collections per subscription: v1 may sync a single, user-specified collection
  URL (simplest, skips discovery entirely); full discovery-and-choose-which-collections (using
  `FindAddressBooks`) is a fast-follow, not required for v1.
- **Depends on WP-73's photo-bridging prerequisite too**: an incoming synced contact's photo needs the
  same `Card.Media` ↔ `Contact.Photo`/`PhotoThumbnail` bridge in `ApplyRecordToContact` that WP-73 is
  adding — do not duplicate that fix here.

**Sequencing:** depends on WP-71 (`ApplyRecordToContact` must exist — done) **and now also on WP-73's
photo-bridging prerequisite** (the `backend/photostore/` extraction + `ApplyRecordToContact` photo
support), since WP-73b needs that same bridge for incoming synced photos — do not duplicate it. Given the
user's own stated order (server first, since "having a sound server will better inform our client work"),
run WP-73 (including its prerequisite) to completion first, then WP-73b. No dependency on P3/P5+.

## WP-74 · Rebrand (effort S/M, do last)

**Name direction: mycelium.** The chosen brand direction is mycelium-themed, reflecting the project's
heavy emphasis on a *network* / graph of connections (the relationship graph, `90`–`92`): a mycelial
network is a decentralized web linking many nodes, which is exactly the "personal relationship graph /
identity hub" the fork is becoming. **Leading placeholder: `Mycelial`** — the adjective form reads well
as a product name and has lower collision risk than `Mycelium` (cf. Mycelium Wallet) or `Mycelia`, which
remain fallback candidates. Lock the final exact name (and the Go module path token, e.g. `mycelial`) at
rebrand time; the mechanics below are name-agnostic. (Note the hard-fork decision `90` D2 means the rebrand can happen at any deliberate
branding moment, not strictly last — but it remains non-blocking; sequencing leaves it here by default.)

- Go module path `meerkat` → new name (e.g. `mycelial`) in `backend/go.mod` + all imports (`gofmt -r` /
  `goimports`); mechanical.
- App name / branding: `frontend/src` strings, `docs/`, `README.md`, `docker-compose*.yml` service
  names, image names in `.github/workflows/docker-publish.yml`, favicon/logo assets.
- New `README` section documenting the RFC 9553/9554 support and the vCard-version export selector.
- Point `origin` at the user's fork; the existing `docker-publish.yml` (tag-triggered `ghcr.io`
  images) works unchanged once the repo owner/name change — a `v*` tag publishes the rebranded image.

## Sequencing & gates

`P0 (WP-10..60) green, app untouched` → `WP-70 (+dry-run migration verified)` →
`WP-71 & WP-72 coordinated (API contract flips with the UI)` → `WP-73` (+ `WP-73b`, parallel-eligible
once WP-71's `ApplyRecordToContact` lands) → `WP-74`.
Each WP: `go build ./... && go test ./...` (backend) and `npm run build && npx tsc --noEmit` +
Playwright e2e (frontend) green before merge. Full-stack smoke via
`docker compose -f docker-compose.test.yml up -d --build --wait` then import a v4 vCard and a
JSContact JSON, export the other, verify.
