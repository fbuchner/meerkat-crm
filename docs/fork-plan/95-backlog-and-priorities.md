# 95 — Backlog and priorities

> **Unlike `00`–`94`, this file is a living document, not a historical planning record.** Re-groom it
> whenever priorities shift. It is the durable source of truth for "what's next" — the in-session task
> tool is used only for tracking the one or two items actively being worked on right now, since it has
> repeatedly lost its full contents mid-session and should not be trusted as the backlog's home.
>
> Last groomed: 2026-07-29 (Tier 1 closed; Tier 2/P5 — WP-80/81/82/83/84/84b/84c(backend) all done; the
> triage-UI/migration/frontend half of WP-84c is now filed under Tier 4, and P5's own acceptance gate is
> technically still waiting on that piece — see Tier 2's note. **Tier 3a fully done** — CardDAV-scoped API
> tokens, configurable OIDC scopes, and RP-Initiated Logout (retained `id_token` cookie, `end_session_endpoint`
> resolved from the already-cached discovery document, real end-to-end verified with a genuine
> RS256-signed fake IdP). **Tier 3c items 1-8 all done** — the confirmed live cascade-delete bug (14
> tables, wider than originally scoped), the 3 unchecked `db.Updates`/`db.Save` sites, the webhook-retry
> job lock, dependabot ecosystem coverage, real WAL mode (not just a doc claim), the rate-limiter doc note,
> and SQLite FK enforcement (decided to enable — no production data to carry pre-existing violations, and a
> full 26-FK audit found no code path that could newly fail, since the only 4 FKs whose parent is ever
> genuinely hard-deleted are all declared `CASCADE`; enabling it also fixed two silent orphan-row leaks in
> `DeleteCircle`/`DeleteTag` for free). Item 3's guard turned out already satisfied. **Tier 3c items 9-10
> (re-dispatching `45-test-coverage-closure.md` Phase 3a+3b, 9 work packages, done together) also closed**
> — every named 0%-coverage function in `httputil/fetch.go`, `webhook_service.go`, `oidc_service.go`
> (a permanent fake-IdP-with-signed-JWT fixture, promoted from a Tier 3a item 2 scratch script),
> `password_reset_service.go`, `mailer.go`/`email_renderer.go`, `reminder_service.go`/`birthday_service.go`
> (`DaysUntilBirthday`'s Dec31→Jan1 boundary confirmed correct, not buggy), and the `graph`/`oidc`/`photo`/
> `user`/`reminder`/`relationship`/`admin_user` controllers (including the doc's specifically-requested
> last-admin/self-promotion audit — both properly guarded) now covered. `45-test-coverage-closure.md`
> itself marked fully closed. Only item 11 (correctness audit) remains in Tier 3c, needing its own scoping
> pass. Four real, unrelated bugs found in passing this session were spawned as separate background tasks;
> three are **already fixed and merged**: the stale `dashboardsInfo` copy left over from Tier 3b (`3e88d36`),
> `ContactSyncLink.ETag`'s GORM column-name mismatch that would have broken CardDAV incremental sync writes
> (`ac17691`), and a handful of older tables (`carddav_sync`, `api_tokens`, `reminder_completions`,
> `calendar_subscriptions`/`calendar_event_links`, `relationships.related_contact_id`) that item 1's
> cascade-delete fix hadn't covered and FK enforcement couldn't help either, since their parents are
> soft-delete-only (`14fc7b0`); one more is still pending: `i18n.IsValidLanguage` accepting any input
> (makes `UpdateLanguage`'s rejection branch unreachable), found during item 10. Tier 6 broadened from
> UI-only to also carry non-critical test-coverage expansion, both still explicitly last-priority, after
> everything else is ready. **Next up: Tier 3c item 11, or Tier 4.**).

## How to read this

Tiers are ordered by "best use of time for immediate impact," not by when the idea was conceived.
Within a tier, do items top-to-bottom. Re-groom (re-run this judgment call) whenever a tier completes,
a security concern surfaces, or the user's priorities change — don't treat the ordering below as fixed
forever.

## Tier 0 — DONE: WP-72 frontend nested-model remodel

**All 7 items done as of `eb7549d` (2026-07-28).** No contact-editing component depends on the flat
`Contact` adapter shape anymore; every one reads/writes `Card`/`CRMEnvelope` (or the raw
`ContactRecordResponse`) directly. The shim functions (`getContact`/`createContact`/`updateContact`/
`toLegacyContact`) are deleted. `Contact`/`ContactValue`/`ContactAddress`/`summaryToLegacyContact`/
`toContactRecordInput` survive deliberately, not as leftover debt — see the item 7 note below for why
each one is a legitimate permanent use rather than shim residue.

| Order | Item | Size | Status | Why here |
|---|---|---|---|---|
| 1 | `contactFields.ts` field-key registry → nested keys | 93 lines | **done** (`8b1cd71`) | Prerequisite for everything below; cheap |
| 2 | `MultiValueField.tsx` / `AddressFields.tsx` → real Card arrays | ~250 lines | **done — no code change needed** | Turned out already correct on investigation; see note below |
| 3 | `ContactInformation.tsx` migration | 421 lines | **done** (`18e68dc`) | Not just a payoff feature — turned out to be architectural, see note below |
| 4 | `AddContactDialog.tsx` migration | 494 lines | **done** (`eb7549d`) | Same shape of work, on the creation path |
| 5 | `ContactHeader.tsx` migration | 422 lines | **done** (`eb7549d`) | Coupled to item 6, same as item 3 was to item 6 — see note below |
| 6 | `ContactDetailPage.tsx` orchestration — remainder | 813 lines | **done** (`eb7549d`) | Landed alongside item 5; every handler and prop now reads `record` directly, the derived `contact` view is gone |
| 7 | Migrate remaining peripheral consumers + retire the adapter shim | — | **done** (`eb7549d`) | `getContact`/`createContact`/`updateContact`/`toLegacyContact` deleted — see note below for what stayed and why |
| ongoing | Unit test coverage for the migrated contact-editing surface | — | 68 tests in `api/contacts.test.ts` | Revisit if a future change touches this surface again; not tracked as a standalone item anymore now that Tier 0 is closed |

Branch: `feature/frontend-nested-model` — ready to merge to `main` and open a PR whenever the user wants.

**Note on item 2 (2026-07-27):** this item's original sizing assumed `MultiValueField.tsx`/
`AddressFields.tsx` didn't yet support multiple entries. Re-investigation (reading the adapter in
`api/contacts.ts` alongside the backend's `contactmodel`/`vcard4` packages) found they already do —
`toLegacyContact`/`toContactRecordInput` already map the *full* `card.emails`/`phones`/`addresses`/
`links`/`imppAddresses` arrays, not just a first entry, and both components already support add/remove
rows. Two smaller vocabulary mismatches were found and traced end-to-end, and turned out to be harmless:
- The frontend stores context tokens as `'home'` where the backend's internal vocabulary is `'private'`,
  and stores phone `'cell'`/`'fax'` selections into `Contexts` rather than `Features`. Both look like bugs
  on paper, but `backend/vcard4/adapter.go`'s `contextsToTypeTokens` falls back to passing unrecognized
  tokens through verbatim — so vCard4/CardDAV export produces the correct `TYPE=home`/`TYPE=cell` either
  way. No functional or data-loss bug; left as-is rather than adding a translation layer to fix something
  that isn't broken.
- `AddressFields`/`toLegacyContact` only round-trip 5 address component kinds (street/locality/region/
  postcode/country). A CardDAV-imported address using other JSContact component kinds (apartment, floor,
  district, ...) would have those silently dropped on the next edit-and-save through this app's UI. Real
  but narrow (only affects externally-imported addresses with non-standard structure) and the fix belongs
  in the adapter (`api/contacts.ts`), not these components — noted here rather than fixed now; revisit if
  CardDAV-imported addresses turn out to actually use these in practice.

**Note on items 3/6 (2026-07-27):** verified empirically (not just from reading code) that multi-value
editing already worked end-to-end *before* any of this migration work — created a contact with 2 emails
and 2 phones via the API, confirmed both displayed correctly through the old shimmed UI, then added a
third via the UI and confirmed all three persisted. So item 3's real value isn't "unlocking a hidden
feature" (nothing was hidden) — it's genuinely retiring this component's dependency on the flat `Contact`
shim, which is architecture work, not a user-facing feature.

That surfaced a real coupling the original item ordering didn't account for: `ContactInformation.tsx`
can't consume `Card`/`CRMEnvelope` directly unless its parent (`ContactDetailPage.tsx`, item 6) *also*
holds nested state, since that's where the fetched record lives and gets threaded down as props. Item 3
ended up including the minimal slice of item 6 needed to unblock it — `ContactDetailPage`'s `contact`
state became `record` (the raw `ContactRecordResponse`, now the single source of truth for every
mutation: circles, profile name/nickname/gender, archive), with `contact` demoted to a value derived from
it via `toLegacyContact` purely for the consumers that haven't migrated yet (`ContactHeader`, delete
confirmation, note/activity/reminder dialogs). This is a strangler-fig approach — old and new shapes
coexist, with `record` as the one source of truth — rather than a big-bang rewrite of everything at once.
Item 6 is now "partially done": the state-shape work landed, what's left is migrating the *other*
consumers (`ContactHeader` in item 5) off the derived `contact` view.

**Note on items 4/5/6/7 (2026-07-28):** items 5 and 6 turned out coupled exactly like 3 and 6 were —
`ContactHeader.tsx` couldn't take `Card`/`CRMEnvelope` without `ContactDetailPage.tsx`'s last remaining
derived `contact` value (kept after item 3 specifically for `ContactHeader`, delete/archive confirmation
text, and dialog `contactId` props) going away too. Both landed together; item 6 has no remainder left.

Item 7 ("retire the adapter shim") turned out to mean something narrower than the phrase suggests, once
it was time to actually do it: the shim isn't one thing to delete, it's two different uses that happened
to share a type.
- **Genuinely retired**: `getContact`/`createContact`/`updateContact`/`toLegacyContact` — the
  record-shaped round-trip through the flat `Contact` type. Zero production callers once
  `DashboardPage.tsx`'s one remaining `getContact()` call (backfilling a reminder's contact name when
  that contact isn't in the "random 5" dashboard widget) was rewritten to read straight off
  `getContactRecord` via `nameComponentValue`. Deleted outright, not deprecated-and-kept.
- **Deliberately kept, not shim debt**: `Contact`/`ContactValue`/`ContactAddress` remain the permanent
  shape for (a) the `GET /contacts` *list* endpoint, which is genuinely flat on the wire
  (`ContactSummaryDTO` via `summaryToLegacyContact`) — there's no nested shape to migrate to there, and
  (b) `MultiValueField`/`AddressFields`' editing-UI contract (item 2's finding: these were already
  correct, no reason to touch them). `toContactRecordInput` stays too — `e2e/fixtures.ts` and
  `e2e/global-setup.ts` still use it to build nested payloads from simple flat test data, which remains
  the most convenient way to do that regardless of what the app itself does.

Verified end-to-end via `docker-compose.test.yml`: full Playwright suite (28/28) plus two manual checks
the suite doesn't cover — a contact created through every `AddContactDialog` field group including the
birthday-reminder-on-create flow, and a seeded scenario (8 contacts, 6 reminders) specifically
engineered to force the `DashboardPage` backfill path to fire, confirming it still resolves names
correctly post-migration.

## Rebranding — DONE (2026-07-28)

Not originally a tier here — a user-directed pivot after Tier 0 landed, done and merged to `main` before
returning to this list. All three legs are merged:
- Typography (`feature/rebrand-typography`) — self-hosted EB Garamond for the wordmark + Source Sans 3
  for UI. See `assets/fonts/README.md`.
- OKLCH color system (`feature/rebrand-colors` + `feature/rebrand-status-colors`) — core palette,
  interaction states, and mushroom-themed error/warning/info/success colors, all wired into `theme.ts`.
  See `assets/colors/README.md`.
- Logo/icons (`feature/rebrand-logo`, merged `644d762`) — mycelium mark replaces the meerkat mark across
  favicon, PWA icons, login page, and Settings About section; light/dark variants via `BrandLogo.tsx`.
  See `assets/logo/README.md`.

Nothing further planned here; revisit only if new brand assets are produced.

## Tier 1 — DONE: Security review (2026-07-28)

All three items done. Fourteen findings, all patched and merged to `main`.

**1. Core backend audit** (`becd907`, `34fbc2c`, `7e988c5`, `fda2c18`, `ae0ef6c`) — eight findings:

| Finding | Severity | Fix |
|---|---|---|
| Go 1.26.0: 19 reachable stdlib CVEs, incl. 2 `html/template` XSS reachable from the email renderer | high | Pinned toolchain to 1.26.5 (19 → 0); pinned floating `golang:alpine`/`alpine:latest` images |
| Webhook SSRF guard bypassable via 302 to an internal address or DNS rebinding | high | Enforcement moved into the transport dialer, which every connection incl. redirects passes through |
| Password change/reset did not invalidate issued JWTs (up to 96h) | medium | `users.token_version` claim, checked per request |
| API tokens never expired | medium | `api_tokens.expires_at`, 90d default / 365d max |
| SVG served by the image proxy → XSS on the API origin | medium | SVG rejected; `Content-Disposition` + CSP on that response |
| No `Content-Security-Policy` on API responses | medium | `default-src 'none'; frame-ancestors 'none'` |
| `FRONTEND_URL` defaulted to `*` (wildcard CORS + credentials) | medium | Refuses to boot in release mode; compose default is now a concrete origin |
| Password accepted via login URL query param | low | Removed |

**2. OIDC/OAuth evaluation** (`04b74cf`) — sound foundation: state and nonce both validated with
constant-time compares, ID token signature/issuer/audience/expiry verified, `email_verified` required
before linking an OIDC identity to an existing account, no open redirect in the callback. Three fixes:
- **UserInfo fallback** — upstream [#189](https://github.com/fbuchner/meerkat-crm/issues/189) applies
  here unchanged. Claims came only from the ID token, but providers (Authelia among them) may return
  just `sub` there. Result was an empty email → linking silently skipped, and with auto-provisioning on,
  users written with `email=''`; since `users.email` is UNIQUE NOT NULL, only the *first* such login
  ever worked. Now falls back to UserInfo, **verifies the UserInfo `sub` matches the ID token's**
  (OIDC Core 5.3.2 — go-oidc does not do this for you), and refuses to provision without an email.
- **PKCE** — flow sent no `code_challenge`; added S256.
- **`COOKIE_DOMAIN`** — both `.env.example` files shipped `'localhost'`, inherited from upstream
  ([#196](https://github.com/fbuchner/meerkat-crm/pull/196)). Now empty (host-only cookie).

A follow-up spec-conformance pass (`3556e3b`) found two more, both in ID token claim decoding:
- **`email_verified` type** — providers disagree on boolean vs quoted string (AWS Cognito sends
  `"true"`); the spec doesn't settle it. The struct declared a plain `bool`, and since `idToken.Claims`
  decodes in one pass, the string form failed *every* claim and killed the login with a generic
  `oidc_error`. go-oidc has this workaround for UserInfo but doesn't expose it for ID tokens.
- **`azp` unchecked** — OIDC Core 3.1.3.7 steps 4-5. go-oidc verifies our `client_id` is in `aud` but
  ignores `azp`, so a token minted for another client that lists us in a multi-valued `aud` was accepted.

**Verified as already correct, so deliberately unchanged:** the `(oidc_subject, oidc_provider)` pairing
key is safe because `oidc.NewProvider` refuses to start unless the configured URL exactly equals the
discovery document's `iss`, so the stored provider value cannot drift from the real issuer. `at_hash`
is unvalidated, which is spec-conformant — it is `MAY` for the code flow (OIDC Core 3.1.3.8), and the
UserInfo `sub` check already covers the substitution risk that validating it would address.

**Known gap, not implemented:** no RP-initiated logout (OIDC RP-Initiated Logout 1.0), plus hardcoded
scopes. These are features rather than patches, and unlike `email_verified`/`azp` above they degrade an
otherwise working login rather than blocking it — with username/password available as the path used for
testing and releasing, none of them gate anything. Scheduled as **Tier 3a** below, together with the
CardDAV app-password work that also unblocks SSO users.

**3. Injection audit** (`04b74cf`) — one real finding: **CSV formula injection** in export. `encoding/csv`
quotes delimiters but leaves a leading `=`/`+`/`-`/`@`/tab/CR intact, so a contact field carrying a
formula executes when the export is opened in a spreadsheet — and contact data is not all self-authored
(CardDAV sync, VCF/CSV import). At-risk values, including user-defined custom field names in the header
row, are now prefixed as text.

Everything else came back clean, verified rather than assumed: SQL uniformly parameterized (the
`ORDER BY` at `contact_controller.go` *looks* injectable but is allowlisted); no `os/exec` anywhere;
templates parsed from an embedded FS, never built from input; export filenames server-generated;
Go's `encoding/xml` resolves no external entities, so the CardDAV surface has no XXE; every `:id`
handler across all resource controllers scopes by `user_id` (zero IDOR); update handlers use explicit
field allowlists (no mass assignment).

**Known, accepted, not patched** — all three now scheduled in Tier 3 rather than left as loose notes:
CardDAV authenticates with the account password rather than app-specific credentials, so a synced-device
leak is full account compromise (**Tier 3a item 1**, which also unblocks SSO users). `contact.Photo`
reaches `filepath.Join` unguarded on the delete path, but is only ever set to a server-generated UUID,
so it is not reachable (**Tier 3c**). `golang.org/x/crypto`'s `openpgp` subpackage carries a standing
advisory with no fixed version; govulncheck confirms it is not called (**Tier 3c**, CVE sweep).

## Tier 2 — IN PROGRESS: P5 core relationship & event model (WP-80..84c backend slice done)

Full detail already lives in `92-delivery-roadmap.md §92.1` — not duplicated here. Hard gate: nothing in
P6+ starts until this is green, since search, timeline, cadence, and integrations all read these
entities. **Not yet fully green**: `92.1`'s own P5 acceptance line requires "legacy relationship + circle
data migrated (dry-run-verified)" — the relationship half is done (WP-81), but the circle half needs the
user-assisted triage UI that was split out of WP-84c and re-filed under Tier 4 (see that section) — so P5's
hard gate technically waits on a piece of Tier 4 work, not just Tier 2 work. Worth keeping in mind when
re-grooming: P6+ isn't actually unblockable purely by finishing what's left in this tier's own list.

**WP-80 — DONE (`cc67a07`, merged to `main` `c8fdc1f`, 2026-07-28).** Relationship graph entity
(`RelationshipEdge`, `models/relationship_edge.go` — the first UUID-string-primary-key entity in this
codebase), type registry (`models/relationship_type_registry.go`, a Go map seeded with every relation
from `91.2`'s worked examples), and `Card.RelatedTo` projection wiring in `RecordForContact`
(`models/contact_record.go`) — verified end-to-end against the real vCard4 exporter, not just Go struct
assertions. Backend-only by design: no new HTTP routes, the existing `/contacts/:id/relationships` API
still serves the legacy `models.Relationship` table unchanged. `RecordForContact` and
`NewContactRecordResponse` both gained a `*gorm.DB` parameter to run the projection query.

**WP-81 — DONE (`e09440f`, merged to `main` `a493255`, 2026-07-29).** `cmd/backfill-relationship-edges` —
the data migration: every legacy `models.Relationship` row becomes a `RelationshipEdge`, with name-only
rows (no `RelatedContactID`) promoted into new thin Contacts per `90` D3. Same dry-run/idempotent/fail-fast
discipline as `cmd/backfill-contact-records` (WP-70), but — unlike that precedent, which has no CLI-level
test coverage — this one does, since it creates new user-visible `Contact` rows rather than just a backend
JSON column. That coverage caught two real bugs before they shipped: `-force` initially tried to `INSERT`
a duplicate edge (fixed by updating the existing row in place) and then, once fixed, still lost the
edge's own `CreatedAt` (a full-column `Save()` from a fresh struct zeroes it) and separately created a
second, orphaned thin Contact on every forced re-run instead of reusing the first one — both caught by
running the real CLI against a seeded scratch database, not just the unit tests. Backend-only, matching
WP-80: the legacy API and every other consumer (`graph_controller.go`, `birthday_service.go`,
`export_controller.go`'s CSV section) still read `models.Relationship` unchanged.

Free-text `Relationship.Type` values are matched to registry tokens via the type registry's own
`Synonyms` field (extended, and now with its first real consumer — previously earmarked for WP-86 search
and unused); anything unmatched (confirmed against this repo's own test fixtures: `"Work"`, `"Family"`)
falls back to a new `related_to` token rather than being dropped, preserving the original text in
`Metadata["legacy_type"]` for later manual reclassification.

**Known, accepted limitation:** a migrated name-only relationship's birthday can now produce a duplicate
reminder — one from the new thin Contact's normal birthday path, one from the old, untouched
`Relationship`-based path in `birthday_service.go` — until a later WP retires that legacy read path.
Scheduled below as a Tier 3b follow-up rather than fixed by expanding WP-81 into a consumer file, which
would repeat the same scope question already settled for WP-80.

**WP-82 — DONE (`6f6ae5d`, merged to `main` `8e512d4`, 2026-07-29).** `CRMEnvelope.Kind`
(`contactmodel/envelope.go` — `individual`/`pet`/`animal`), deliberately separate from the pre-existing
standards-side `Card.Kind` (which has no pet/animal value) and deliberately unvalidated, matching this
codebase's own documented policy of accepting-and-preserving unrecognized nested `Card`/`CRM` values
rather than a hardcoded `oneof`. No migration needed — `CRM` is a JSON blob copied wholesale everywhere
it's touched, so the new field required zero changes outside its own struct definition. No export-time
synthesis for pets: `Card.Kind` stays pure passthrough, exactly as today.

The other half of this WP's scope, "thin entities: nothing but name required," turned out to already work
end-to-end — both backend and frontend — confirmed during planning rather than assumed, and locked in with
a new test (`TestCreateContact_RealValidation_ThinEntityAccepted`) rather than left as a one-time
observation. Also fixed in passing: the shared `controllers` package test fixture had been missing
`RelationshipEdge` from its `AutoMigrate` call since WP-80 merged, silently logging a swallowed "no such
table" warning on every contact-creating test in the package.

**WP-83 — DONE (`64d7cbc`, merged to `main` `872200f`, 2026-07-29).** `Household`/`HouseholdMember`
(`models/household.go` — the second UUID-string-PK entity after `RelationshipEdge`) and
`services.GenerateHouseholdSuggestions` (§91.4's mechanism): re-scans a household's current membership and
idempotently ensures a suggested `RelationshipEdge` exists for every applicable pair, never treated as
fact until a user confirms it in a review surface this WP doesn't build (P-later, per the roadmap).
Member classification is `HouseholdMember.Role` + `Contact.CRM.Kind` only (confirmed during planning) — no
birthday/age inference, since birthdays are frequently unknown, especially for WP-81's thin entities.
Backend-only: no API, no controller, no frontend, no standards projection (`Household` → vCard
`KIND:group`+`MEMBER`, which §91.3 mentions but this WP's own roadmap line doesn't) — all future work.

Two real bugs found and fixed before merge, both caught by tests rather than inspection: (1) GORM's
default column-naming derived `HouseholdMember.MemberVCardUID` to `member_v_card_uid`, silently
mismatching the raw-SQL migration's real `member_vcard_uid` column — fixed with an explicit `gorm:"column:
..."` tag; (2) the suggestion-engine tests initially set `Contact.CRM.Kind` by direct field mutation
before `Create`, which doesn't survive `BeforeSave` (the same bug WP-81's passthrough test hit for the
same reason) — every "pet" test contact was silently classified as an adult until fixed via
`ApplyRecordToContact`. That investigation also caught a real arithmetic error in this WP's own plan: the
worked example (2 adults + 1 child + 1 pet) was planned as producing 2 `owned_by` edges, but §91.4 says
"every human → household pet `owned_by`," not "every adult" — the child gets one too, for 3. The shipped
test asserts the correct count, not the plan's.

**WP-84 — DONE (2026-07-29).** Bundled all three sub-projects, scoped **backend-only/additive** (confirmed
during planning): `Contact.Circles`, the existing `Activity` API, and every current controller/frontend
consumer keep working exactly as before, untouched — no new routes, no frontend changes, matching
WP-80–83's own precedent.
- **Interaction** (§91.7): extended `Activity` in place (`models/activity.go`) rather than replacing it —
  kept the existing int PK, added `UUID`/`Type`/`ExternalRef` as new columns, following
  `Contact.VCardUID`'s own precedent for adding a stable UUID identity to a table with existing production
  rows. Existing rows backfilled via `migrations/000030`'s `UPDATE` statement, the same
  `000008`/`000009`-style split used historically for `contacts.vcard_uid`. Added `Activity.Qualifying()`
  (§91.7's "derived, not stored" field) with no consumer yet — cadence (WP-94) is future work.
- **LifeEvent** (§91.6): new UUID-PK entity (`models/life_event.go`, `migrations/000032`), following
  `Household`'s exact template. `Date` reuses `contactmodel.PartialDate` per the spec's own instruction.
  `RelatedEntityIDs` is a single JSON-array field (not a join table) covering both "secondary participants"
  (e.g. both spouses in a `married` event) and "related entity" (the new child/pet/org) — a deliberate
  simplification since nothing needs to query from the related-entity side yet, the same proportionality
  call `Household.Address` made.
- **Circle + Tag** (§91.5): two new small entity pairs (`models/circle.go`, `models/tag.go`,
  `migrations/000031`), following `Household`/`HouseholdMember`'s exact template. Tag projects onto
  `Card.Keywords` via a new `projectTags` in `models/contact_record.go`, structurally identical to WP-80's
  `projectRelationshipEdges` — wired into `RecordForContact` alongside the existing `RelatedTo` projection.
  No data migration of existing `Contact.Circles` strings into these new tables — that's WP-84c below.

No new bugs this time (the `MemberVCardUID`-column-naming and `ApplyRecordToContact`-vs-direct-mutation
traps WP-83 hit were both known going in and avoided from the start). Verified against a real migrated
SQLite DB (`database.InitDB`, the actual production migration path, not just `AutoMigrate`): a fresh
`Activity` gets a `BeforeCreate` UUID, the migration's own backfill statement recovers a cleared one, two
`Tag`s merge into `RecordForContact`'s `Card.Keywords` alongside an existing passthrough keyword without
duplication, a `LifeEvent` with a year-only `PartialDate` and a `RelatedEntityIDs` entry round-trips
exactly, and `CircleMember`'s unique constraint is enforced by the real DB.

**WP-84c (backend CRUD slice) — DONE (2026-07-29).** Confirmed with you: split WP-84c rather than build it
as one bundle — the backend CRUD API happens now, the triage UI, `Contact.Circles` data migration, and all
frontend rewiring move to Tier 4 (see below), since "no one is using this yet" and breaking changes there
are fine to defer without urgency. This is also the **first** WP in this series to add real HTTP surface —
WP-80 through WP-84b were deliberately backend-model-only.

Full CRUD (`controllers/circle_controller.go`, `tag_controller.go`, `life_event_controller.go`) for
`Circle`/`Tag`/`LifeEvent`, following `activity_controller.go`'s existing conventions exactly (
`currentUserID`/`.Where("user_id = ?", ...)` ownership, `middleware.GetValidated[T]`, `apperrors.
AbortWithError`, `GetPaginationParams`). No existing precedent covered join-row (`CircleMember`/
`ContactTag`) endpoints, so this WP had to decide one: real nested sub-resource endpoints (`POST/DELETE
/circles/:id/members`, `POST/DELETE /tags/:id/contacts`) rather than folding membership into a bulk-replace
DTO field the way `Activity.Contacts` does — membership add/remove is its own action with its own
lifecycle, not a field of "editing the circle's name." A duplicate add is a clear `409 ErrAlreadyExists`
(checked by querying first, not by sniffing a unique-constraint error string).

Also closed a real, separate gap found during research: `Activity`/"Interaction" already had full CRUD, but
its `Type`/`ExternalRef` fields (added in WP-84) were never wired into `ActivityInput` — they round-tripped
on read but could never be set via the API. Fixed alongside this WP since it directly completes "CRUD
routes for Interaction" from WP-84c's own original description.

No bugs found beyond the confirmed Activity DTO gap above. Verified two ways: the full Go test suite
(27 new/modified tests across `circle_controller_test.go`/`tag_controller_test.go`/
`life_event_controller_test.go`/`activity_controller_test.go`), and — since this is the first WP with real
routes — an actual running server against a scratch SQLite DB, driven with real `curl` requests through
cookie-based auth: registered a user, created a contact, created a Circle and added/removed a member
(confirming the 409 on a duplicate add), tagged/untagged a contact, created a `LifeEvent` with a partial
date and listed it back filtered by `entity_id`, and confirmed an Activity's `Type`/`ExternalRef` persist
through a real `POST`+`GET` round trip.

**Deferred to Tier 4 as its own item** (see below): the triage UI, the `Contact.Circles` → `Circle`/`Tag`
data migration (which genuinely needs that UI, not a heuristic — §91.5 is explicit this is "a light
user-assisted step"), and rewiring the ~5 backend call sites (`contact_controller.go`'s `GetCircles` and
JSON-array filtering, `import_service.go`'s circles/tags/groups/labels synonym-mapping) and ~17 frontend
files currently consuming `circles` as a flat string array (chips, filters, graph nodes, dashboard, import
dialog) — the highest-blast-radius piece of the original WP-84c scope.

**WP-84b — DONE (2026-07-29).** `FieldDefinition`/`FieldValue` (`models/field_definition.go` — the
schema/data two-part model from §94.3), generalizing the untyped v1 (`User.CustomFieldNames` +
`Contact.CustomFields`, both left fully intact and untouched — CSV export and the two frontend pages that
read them keep working exactly as before). Scoped **backend-only**, matching the roadmap's own "full UX
depends on P3" note and WP-80–84's no-routes precedent.
- **Validation** (§94.4): a new `services.ValidateFieldValue`, dispatching per `Type` to a mix of reused
  validators (`middleware.ValidateEmail`, and a new `middleware.ValidateVar` primitive that exposes the
  existing `phone`/`birthday`/`safeurl` custom validators plus the validator library's built-ins to a
  single runtime value rather than only a tagged struct field) and small native Go checks (string
  length/pattern, RFC3339 datetime) where the validator library has no dynamic-tag equivalent.
  `FieldConstraints.Multi` (not a 10th `Type` token) makes any scalar type a validated list, per §94.4's own
  wording.
- **Standards projection** (§94.5): only `internal-only` and `vcard:X-<NAME>` are implemented — the doc's
  third option, a raw `jscontact:<pointer>` projection, is deliberately **not built**: JSContact's
  `Card.VCardProps` already *is* `Passthrough.VCard` copied through verbatim, so a `vcard:`-projected field
  already reaches vCard3, vCard4, *and* JSContact through the one existing mechanism. New
  `projectCustomFields` in `models/contact_record.go`, structurally identical to WP-84's `projectTags`,
  filtering `sensitivity='normal'` in the query itself (§91.13 discipline, verified: a `secret`-sensitivity
  field with a `vcard:` mapping does **not** project).
- **v1 migration** (§94.6, explicitly in this WP's own roadmap line, unlike WP-84's Circle/Tag split):
  `cmd/backfill-custom-fields`, following `cmd/backfill-relationship-edges`'s dry-run/idempotent/fail-fast
  template exactly. Two passes (definitions, then values, since a value references its definition) — a
  documented, non-bug consequence of this split: a **dry-run** report cannot show pass-2 successes for
  names pass 1 would create, since pass 1 makes no real writes to look up in dry-run mode (confirmed during
  manual verification — pass 2 reports "no field definition found" for every value on a dry run against a
  fresh DB, then succeeds normally once `-write` actually runs). `-force` exists only for the values pass
  (a value can drift from v1 after a first migration); definitions have no `-force`, since a migrated
  `Key`/`Label`/`Type` never changes out from under it.

No bugs found this time — verified against a real migrated SQLite DB (`database.InitDB`): an invalid
`phone` value is rejected, a valid one accepted; a `vcard:X-PRONOUNS`-projected, non-sensitive `enum`+
`Multi` value appears correctly in `RecordForContact(...).Passthrough.VCard`; a `secret`-sensitivity
`vcard:`-mapped field does not; the backfill tool's dry-run → `-write` → re-run sequence against a seeded
DB with real v1 data produced exactly 2 definitions + 3 values, then zero further writes on re-run.

Next: **WP-84c** (the deferred Circle/Tag data migration + CRUD routes + frontend rewiring, scheduled
above) is now the only unscheduled item directly descended from Tier 2/P5's work. Time to re-groom Tier 3
and beyond for what's next.

## Tier 3 — Remaining backend hardening/audits + auth follow-ups

Lower urgency than Tier 1's security-relevant items — risk-reduction and correctness, not exposure.

### 3a. Auth follow-ups carried out of Tier 1

Deferred behind Tier 2 deliberately: username/password login works and is the path used for testing and
releasing, so none of these block anything. They are correctness gaps in a *secondary* auth path. Note
the distinction from what Tier 1 actually fixed — `email_verified` and `azp` were outright blockers for
affected providers, whereas everything below degrades an otherwise working login.

**Re-scoped 2026-07-29** against the actual code rather than the original grooming-time guesses — sizes
below are corrected, not the original estimates.

| # | Item | Size | Notes |
|---|---|---|---|
| 1 | **App-specific passwords for CardDAV** | **medium** (actual; was: small–medium) | **DONE** (2026-07-29). Open design question resolved: added a `carddav`-scoped token type rather than accepting any general-purpose token (`full` ⊇ `carddav` — a `full` token still works for CardDAV, but a `carddav`-scoped token is rejected by the general REST bearer-auth path, so a leaked synced-device credential is confined to contact sync). `ApiToken.Scope` column (migration 000034), `middleware.LookupAPIToken`/`TouchAPIToken` extracted and shared between `AuthMiddleware` and `carddav/auth.go`'s new token fallback, `ApiTokensPage.tsx` gained a scope selector + column. Actual size landed at `medium` (schema + backend + frontend + 5 locale files), not the original `small–medium` estimate, since the scoped-token decision (rather than "accept any token") pulled in the frontend create-dialog and column work too. Real-DB verified: password auth unchanged, both token scopes work for CardDAV, a `carddav`-scoped token is rejected 403 against the general REST API, SSO/OIDC users (empty password) authenticate CardDAV via a token, wrong-username/expired/revoked tokens all rejected. |
| 2 | **RP-initiated logout (OIDC RP-Initiated Logout 1.0)** | **medium–large** (confirmed) | **DONE** (2026-07-29). `ExchangeAndVerify` now also returns the raw ID token JWT (previously discarded once verified), retained via a new `id_token` cookie (same flags/lifetime as `auth_token`) — its presence is also the signal for whether *this* session came via SSO at all, so a local-password login even with OIDC enabled never attempts an IdP round trip. New `OIDCProvider.EndSessionEndpoint()` reads the already-cached discovery claims (zero extra network I/O). `LogoutUser` always clears both cookies first (local logout never depends on IdP reachability), then — only when an `id_token` cookie was present — builds the full `end_session_endpoint` URL (`id_token_hint` + `post_logout_redirect_uri`, via `net/url`, not string concatenation) and returns it as `redirect_url`; any failure (unsupported by the IdP, parse error) logs a warning and falls back to today's plain response. New `OIDCConfig.PostLogoutRedirectURL` derived from `FrontendURL` (`.../login`), mirroring `RedirectURL`'s existing precedent — reuses the existing unauthenticated catch-all route, so no new frontend route was needed. Frontend `logoutAndRedirect()` now does a real top-level navigation to the IdP when a `redirect_url` comes back (a `fetch` can't clear the IdP's own first-party cookie), else keeps today's client-side `/login` redirect. Real end-to-end flow verified against a real migrated DB with a fake IdP doing genuine RS256-signed JWT issuance: login sets both cookies, logout's `redirect_url` carries the exact same `id_token_hint` and a correctly URL-encoded `post_logout_redirect_uri`, both cookies cleared. Closes Tier 3a. |
| 3 | **Configurable OIDC scopes** | small (confirmed) | **DONE** (2026-07-29). `OIDC_SCOPES` env var (comma-separated, defaults to `openid,email,profile`), `config.getScopesEnv` following the existing `getProxies` list-parsing pattern, wired into `InitOIDCProvider`'s `oauth2.Config.Scopes`. `.env.example` and `docs/getting-started.md` updated; unit-tested in `config_test.go`. |

**On item 1 — why the two CardDAV problems are one piece of work.** Tier 1 recorded that CardDAV
authenticates with the account password, so a synced-device credential leak is full account compromise.
Separately, OIDC-provisioned users get `Password: ""`, which bcrypt can never match — so **SSO users
cannot use CardDAV at all**. Both are the same missing capability: a per-device credential that is not
the account password. Building app-specific passwords fixes the blast radius *and* unblocks SSO users in
one change, so they should not be scheduled separately.

Sequencing note: item 1 touches the CardDAV auth path, which P5 (Tier 2) does not, so there is no
ordering constraint between them beyond priority.

**Tier 3a — all three items done** (2026-07-29): 1 (App-specific passwords), 3 (Configurable OIDC scopes),
2 (RP-Initiated Logout).

### 3b. WP-81 follow-up

- **Retire `birthday_service.go`'s legacy `Relationship`-based birthday reminder path — DONE (2026-07-29).**
  Deleted the `related_contact_id IS NULL` relationship-birthday query block. Actual scope turned out wider
  than the original "~55-line block + one frontend branch" estimate: `GetUpcomingBirthdays` is the single
  source feeding the dashboard widget, the emailed daily reminder, and the `birthday.occurred` webhook
  trigger, and its `Type: "relationship"` DTO branch cascaded through 7 files total —
  `birthday_service.go` (the query), `models/dtos.go` (`Birthday.RelationshipType`/`AssociatedContactName`
  removed, `Type` narrowed to always `"contact"`), `reminder_service.go` + `email_renderer.go`'s
  `BirthdayItem` (dead fields removed), `templates/reminder.html` (dead `{{if .IsRelationship}}` block
  removed), and `frontend/src/api/contacts.ts` + `DashboardPage.tsx` (dead TS fields/JSX branch removed).
  Verified with a real-DB check: seeded a contact birthday plus a legacy name-only relationship with the
  same birthday (the exact shape the deleted query matched on — `related_contact_id IS NULL` + a set
  `birthday`), confirmed `GetUpcomingBirthdays` now returns exactly one entry (the contact), not two. Full
  backend test suite green, frontend `tsc --noEmit` clean.

### 3c. Standing audits — broken down into pickup-ready items

**Re-scoped and broken down 2026-07-29**: these were unsized placeholder bullets at grooming time. Actually
researching each turned two into confirmed, currently-live bugs (not hypothetical audit findings) and fully
converted the rest from "go audit this" into concrete, ready-to-implement fixes — so almost none of these
should be framed as open-ended audits anymore, and several split into independently-pickable pieces. Table
is in **recommended pickup order** (small/high-value/low-risk first, the still-unsized item last), not the
original listing order.

| # | Item | Size | Notes |
|---|---|---|---|
| 1 | **Extend `DeleteUser`/`DeleteContact` cascade deletes** (data-lifecycle audit, part a) | small–medium | **DONE** (2026-07-29, `c6c90b9`). Actual scope was 14 tables, not ~13 — the "pre-existing" `webhooks`/`webhook_deliveries`/`contact_subscriptions`/`contact_sync_links` turned out **not** to be handled either, contrary to earlier assumption. `DeleteContact` scopes by `Contact.VCardUID` for every WP-80+ entity (the graph invariant), except `ContactSyncLink` (genuine `ContactID` FK). Containers (`Household`/`Circle`/`Tag`/`FieldDefinition`) are only deleted by `DeleteUser`, never `DeleteContact`. Known, accepted limitation: `LifeEvent.RelatedEntityIDs` can still hold a deleted contact's `VCardUID` as a secondary participant (a JSON array value, not an orphaned row). Real-DB verified end-to-end. A real, unrelated bug was found in passing during verification, spawned as a separate task, and **fixed** (2026-07-29, `ac17691`): `ContactSyncLink.ETag` had no explicit `gorm:"column:etag"` tag, so GORM wrote to a nonexistent `e_tag` column against the real migrated schema — would have broken CardDAV incremental sync updates in production. |
| 2 | **Fix the 3 unchecked `db.Updates()`/`db.Save()` call sites** (silent-failure audit) | small | **DONE** (2026-07-29, `ad06ab2`). `reminder_controller.go:138`, `note_controller.go:237`, `relationship_controller.go:195` — all three now check `.Error` and abort with `apperrors.ErrDatabase`, matching each file's own existing convention. |
| 3 | **Guard `contact.Photo`/`PhotoThumbnail` before `filepath.Join` on delete** (defense-in-depth) | trivial | **DONE (already satisfied, no code change)** — 2026-07-29 re-check found `deleteContactPhotos` already has the `!= ""` guards this item described as missing; the original grooming's file read was stale. |
| 4 | **Add the missing job lock to `ProcessWebhookRetries`** (single-instance audit, part a) | small | **DONE** (2026-07-29, `f3bf7cd`). Added `JobNameWebhookRetries` and wrapped the function with `acquireJobLock`/`releaseJobLock` using a 4-minute `minInterval` (shorter than the 5-minute cron cadence, so the lock doesn't suppress every other tick). New regression tests cover both the locked-skip and acquire/release paths. |
| 5 | **Restore `npm_and_yarn`/`bundler` to `.github/dependabot.yml`** (CVE sweep, part b) | small | **DONE** (2026-07-29, `8287a8b`). Git history check found the "used to exist" framing didn't hold (this file has exactly one commit, never had these entries in this repo) — added fresh, matching the file's existing minimal style. `frontend/` (npm_and_yarn) and `docs/` (bundler, the Jekyll site's Gemfile) now watched. |
| 6 | **Correct or implement the WAL-mode claim in `docs/deployment.md:88`** (backup audit spinoff) | trivial | **DONE** (2026-07-29, `43604bb`). Chose to make the claim true rather than soften it: added `?_pragma=journal_mode(WAL)` to both real DB-open call sites (`database.InitDB`, `cmd/migrate`'s standalone CLI). WAL is persisted in the file itself once set. Manually verified via `PRAGMA journal_mode` against a real connection. No doc edit needed. |
| 7 | **Add a doc note on in-memory rate limiters** (single-instance audit, part b) | trivial | **DONE** (2026-07-29, `bf16bd8`). One sentence added to `docs/deployment.md`'s "How the Docker Setup Works" section. |
| 8 | **Decide + implement SQLite FK-enforcement policy** (data-lifecycle audit, part b) | medium | **DONE** (2026-07-29). Decided to enable, since no production data exists yet to carry pre-existing constraint violations. Before flipping it on, audited all 26 declared FKs across every migration (not just the newer ones): 17 target `contacts`/`users`, both soft-delete-only (`gorm.Model`) with zero hard-delete call sites anywhere in the app, so enforcement is a no-op for those regardless. Only 4 FKs (`household_members`/`circle_members`/`contact_tags`/`field_values` → their respective parents) have a parent that's ever truly hard-deleted, and all 4 are declared `CASCADE` — no `RESTRICT`/`SET NULL` exists anywhere in the schema, so **no code path could newly start failing**. Two of those four (`DeleteCircle`, `DeleteTag`) had no explicit member-cleanup code and were silently leaking orphaned rows — enabling enforcement fixed both for free via the DB-level `CASCADE`, no code change needed. Implemented via `database.openDSN`'s `?_pragma=foreign_keys(1)` (a per-connection setting, unlike `journal_mode`, so it's supplied via the DSN — applied by the driver on every new physical connection — rather than a one-time `PRAGMA` statement). Real-DB verified: `PRAGMA foreign_keys` reports 1, `DeleteCircle`/`DeleteTag` now auto-cascade with zero orphaned rows, `DeleteUser`/`DeleteContact` (item 1's explicit ordering) unaffected. A handful of additional orphan-row gaps unrelated to this decision (`carddav_sync`, `api_tokens`, `reminder_completions`, `calendar_subscriptions`/`calendar_event_links`, `relationships.related_contact_id` — all soft-delete-parent, so FK enforcement can't help them) were found in passing, spawned as a separate task, and **fixed** (2026-07-29, `14fc7b0`) — `DeleteUser`/`DeleteContact` now cover all of them, real-DB verified. |
| 9 | **Re-dispatch `45-test-coverage-closure.md` Phase 3a** (test-coverage closure, security-sensitive half) | medium | **DONE** (2026-07-29, 4 commits: `0705810`, `554129d`, `a8def33`, `1b96ed2`). `httputil/fetch.go`'s SSRF guards now 100%/17.1% (remainder is the live-network success path); `webhook_service.go`'s signing/retry/persistence/delivery path 100% across all 5 previously-0% functions; `oidc_service.go`'s token-exchange path 75-100% via a permanent fake-IdP-with-signed-JWT fixture (promoted from a scratch script built during Tier 3a item 2); `password_reset_service.go` 71-100%. Every WP hand-verified with a real negative-path break/confirm-fail/restore cycle. Checked, not fixed: the `isPrivateURL` fail-open/`httputil` fail-closed asymmetry the source doc flagged no longer exists (fixed by an earlier commit, independently confirmed). `services` package coverage 69.8%→81.1%. |
| 10 | **Re-dispatch `45-test-coverage-closure.md` Phase 3b** (test-coverage closure, lower-risk half) | medium | **DONE** (2026-07-29, 5 commits: `98847df`, `425736c`, `846049c`, `0874ed6`, `dc1c810`). Mailer/email-renderer (`sendViaResend` deliberately excluded, no test seam, per the source doc's own note); `DaysUntilBirthday`'s Dec 31→Jan 1 boundary verified **correct**, not buggy (forward-looking-only semantics, proven with explicit wrap-around tests); `reminder_service.go` remaining gaps; `graph`/`oidc`/`photo` controllers (OIDC callback tests reuse item 9's fake-IdP fixture, proving forged-signature/nonce-mismatch/azp rejection through the real handler); `user`/`reminder`/`relationship` controllers including explicit cross-user ownership-boundary tests (no gaps found); `admin_user_controller.go`'s remaining handlers with the doc's specifically-requested last-admin/self-promotion audit (both properly guarded; self-promotion is gated correctly but only at the route-middleware layer, not defense-in-depth — documented, not changed). Two real, out-of-scope findings spawned as separate tasks rather than fixed inline: `i18n.IsValidLanguage` accepts any input, making `UpdateLanguage`'s rejection branch unreachable. `services`/`controllers` package coverage 81.1%/47.1%→81.1%/64.0%. (Coverage for packages `45` never covered at all — `config`/`database`/`routes`/`errors`/`i18n`/`logger`/`cmd/*` — stays in Tier 6, not here.) |
| 11 | **Correctness audit — needs its own scoping pass before it can be broken down further** | **unsized** | Re-scoped per the user's clarification: the methodology is **writing tests at business-logic decision points**, not code review — matching this session's own experience that nearly every real bug (GORM column-naming mismatches, `Contact.CRM` mutations that silently didn't persist, the WP-81 `-force` edge-duplication chain) was caught by writing a test exercising real behavior, not by inspection. Distinct from items 9-10: those chase specific named zero-coverage functions (mechanical); this one is exploratory — writing tests at the riskiest/least-verified business-logic points even where some coverage exists (e.g. suggestion-engine edge cases, projection/sensitivity filtering, validation dispatch). **Do this last, and treat "identify the 5-10 highest-risk business-logic areas" as its own first step** before it can be split into pickable pieces the way items 1-10 already are.

## Tier 4 — P6–P10 (search, CalDAV export, external links/Immich, sync, relationship-maintenance)

Unchanged from `92-delivery-roadmap.md §92.2–92.6` — gated behind Tier 2 (P5) by that doc's own hard
gate. See that file for the full WP breakdown and dependency graph (`§92.8`).

**Exception: WP-97, selective field export (vCard 3/4, JSContact)** (`92-delivery-roadmap.md §92.6b`,
added 2026-07-29 at the user's request — Google Contacts' "choose fields to export" is the reference).
Depends only on P0 + WP-73 (both done), **not** on P5's graph work, so it is not actually gated behind the
rest of this tier and could be picked up independently whenever convenient. Its field-selection UI and
filter function are meant to be **reused, not rebuilt**, by Tier 5 below.

**Exception: WP-84c's deferred frontend/migration half** (split out 2026-07-29 — the backend CRUD slice of
WP-84c is done, see Tier 2 above). Three pieces, likely worth splitting further when picked up rather than
doing as one PR, given how large WP-84's own frontend blast-radius estimate was:
1. A user-assisted triage UI for classifying each existing `Contact.Circles` string as a `Circle` or a
   `Tag` (§91.5 is explicit this must not be an automated heuristic) — this is also the piece Tier 2/P5's
   own acceptance criteria is technically still waiting on (see the note in that section).
2. Rewiring the ~5 backend call sites still reading/writing the flat field directly: `contact_controller.
   go`'s `GetCircles` and its `json_each`-based JSON-array filtering, and `import_service.go`'s
   circles/tags/groups/labels synonym-mapping (which currently maps ALL of those onto the one flat
   `circles` field — once Tag exists as a real destination, this mapping needs to split by target, not just
   change where it writes).
3. Rewiring the ~17 frontend files currently consuming `circles` as a flat string array (chips, filters,
   graph nodes, dashboard, import dialog) to use the new Circle/Tag entities and their CRUD API instead.

## Tier 5 — Contact sharing between users (standalone, big)

Two users on the same instance (e.g. spouses) should be able to share contacts directly — opting which
fields to include — rather than round-tripping through lossy VCF export/import. Explicitly a "revisit on
the roadmap at some point" item per the user, not scheduled. Comparable in scope to a P8-or-later phase;
needs its own design pass (data model for shared-vs-private fields, permission model) before it can be
broken into WPs.

**The "opting which fields to include" half is already scheduled**: Tier 4's WP-97 (above) builds a
reusable field-selection UI and filter function for export, at the user's explicit request that the same
system be reused here rather than built twice. When this tier gets its design pass, start from WP-97's
selection model/UI and add the sharing-specific parts (persistence for a live share vs. a one-time export,
and the permission model) rather than re-deriving field selection from scratch.

Also carries WP-97's sensitivity-default rule (see `92-delivery-roadmap.md §92.6b`): sensitivity-marked
relationships/tags/life-events (`91.13`) default excluded from a share, with an explicit per-share opt-in
override — arguably higher-stakes here than for a file export, since sharing hands the data to a specific
other person on the instance, not just out to a file. Worth confirming this default still feels right once
this tier's permission model exists (e.g. whether a standing/live share should re-apply the default on
every sync, or only at share-creation time).

Carries WP-97's foot-gun-prevention requirement too, and it matters *more* here than for export: a
sensitive item must be gated behind a deliberate extra action before it's even selectable for a share, not
just unchecked by default — a misclick here doesn't just produce an unwanted file, it discloses the data to
another live person on the instance, likely without either party noticing until later. If this tier ends
up supporting a *standing* share (auto-syncing, not a one-time send), the gating should arguably re-confirm
on every field newly marked sensitive after the share was created, not just at creation time — flagged here
as a design question, not decided.

## Tier 6 — Polish (UI + non-critical test coverage)

Explicitly last priority — polish after everything else (Tiers 1–5) is ready, per the user's own framing.
Originally UI-only (fonts/icons/strings, items 1–4 below); broadened 2026-07-29 to also carry non-critical
test-coverage expansion (item 5), since both share the same "do this once everything that actually matters
is done" priority, even though one is frontend and one is backend.

### UI polish (fonts, icons, strings)

Not a fixed checklist: the task is to actually walk through the app's flows page by page and find places
where the typography, iconography, or copy could be clearer or more consistent, using the examples below as
a starting calibration for what "better" looks like, not an exhaustive list to check off.

1. **Typography audit** — review which font is used where across the app (headings, body, labels, monospace
   if any) and confirm it's consistent and intentional, not just whatever a component happened to inherit.
2. **Icon library**: the frontend currently uses `@mui/icons-material` only (confirmed via `package.json` —
   no `@mdi/js`/`@mdi/react` dependency exists yet). Add MDI (Material Design Icons, Pictogrammers) as a
   dependency and use it where it has a more specific/better semantic match than MUI's own set — doesn't
   require ripping out every existing MUI icon at once. Named examples to start from: notes list icon →
   `mdi-note-multiple-outline`, "add note" action → `mdi-note-plus-outline`, network/graph page →
   `mdi-graph-outline`.
3. **String/copy review** — walk each flow (contacts, network, notes, activities, settings, import/export,
   ...) and fix labels that don't clearly describe what they do. Named example: the Settings page's
   "Profile" sub-label doesn't make sense — it's just Settings, not a distinct "Profile" section within it;
   needs either a clearer label or removing the redundant sub-naming. (Exact current location wasn't
   pre-located — finding it by walking the real UI is part of this task, not done ahead of time here.)
4. General instruction for whoever picks this up: treat 1–3 as a method, not a checklist — go through the
   actual running app, flow by flow, and note/fix anything that reads as unpolished (inconsistent icon
   style, a label that's technically accurate but not clear, a font that doesn't match its surroundings),
   similar in spirit to the three examples above.

### Non-critical test coverage expansion

5. **Expand test coverage outside the already-scoped security and critical-path tests** (added 2026-07-29,
   split out of Tier 3c's "broader test-coverage closure" during that item's re-scoping — see the note
   there). `docs/fork-plan/45-test-coverage-closure.md`'s own Phase 3 scope (Tier 3, higher priority)
   doesn't cover these packages at all; real `go test ./... -cover` numbers as of 2026-07-29: `config`
   24.2%, `database` 37.8%, `routes` 0.0%, `errors` 0.0%, `i18n` 0.0%, `logger` 0.0%, and the one-shot
   `cmd/backfill-custom-fields` 33.7% / `cmd/backfill-relationship-edges` 55.6% / `cmd/backfill-contact-records`
   0.0% / `cmd/migrate` 0.0%. None of these are security-sensitive (config loading, route registration,
   logging, i18n string loading, already-run one-off migration scripts) — that's exactly why this is Tier 6
   and not Tier 3. Needs a fresh scoping pass to decide which of these are actually worth covering versus
   accepted as low-value (e.g. `cmd/migrate`'s `main.go` may just be a thin CLI wrapper not worth testing in
   isolation) — don't chase the percentage for its own sake.

## Deferred / someday

Unchanged from `92-delivery-roadmap.md §92.7`: other integrations (Dawarich/GeoPulse, Jellyfin,
Audiobookshelf, Paperless-ngx, Nextcloud), AI/Ollama layer. Pulled in only when a concrete need arises.

## Explicitly not re-ranked here

`80-local-model-pilot.md`'s deferral is independent of this backlog (re-enters when mobile client work
begins, per `92.9`).
