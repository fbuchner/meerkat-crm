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
> itself marked fully closed. **Item 11 (correctness audit) scoped and closed** — 3 parallel research agents
> surveyed every candidate business-logic area (relationship graph, sync/import, validation/sensitivity,
> reminders/webhooks) and found most decision points already behaviorally tested from prior work; the real
> surface was 3 small WPs (11a-c, all done) plus one flagged design decision (11d, deferred). **11a**
> (`contact_sync_service.go` overwrite semantics) confirmed the full-overwrite-on-sync behavior intentional,
> but writing the pinning test caught a real, previously-unknown bug: `applyPhones` cleared `Contact.Phones`
> but left the `Phone` scalar stale when a contact's phones were removed, unlike its sibling `applyEmails`
> — fixed (`8e13ffc`). **11b** (reminder eligibility filter) and **11c** (import-merge array-field policy)
> both confirmed correct, not buggy, and are now behaviorally pinned (`b1d3d92`, `15bcf69`). **11d**
> (`graph_controller.go`'s `GetGraph` reading only the legacy `models.Relationship` table) turned out bigger
> than a rewire once fully scoped: `RelationshipEdge` has **zero CRUD API or frontend today**, so the legacy
> model is the CRM's only working relationship-management feature, not a redundant parallel one — deleting it
> without a replacement would remove real functionality. Superseded and moved to a new, fully-scoped **3d**
> section (5 WPs: new `RelationshipEdge` CRUD API, `GetGraph` rewire, new frontend UI, rewiring the other
> legacy-model dependents, then removal) — **sized `large`, explicitly scheduled before Tier 4** per the user
> (do it while it's still safe, before Phase 6 completes and production data exists). **All of Tier 3c
> (items 1-11) is now done; 3d is scoped and next up.** Four real, unrelated bugs found in passing this
> session were spawned as separate background tasks; three are **already fixed and merged**: the stale
> `dashboardsInfo` copy left over from Tier 3b (`3e88d36`), `ContactSyncLink.ETag`'s GORM column-name
> mismatch that would have broken CardDAV incremental sync writes (`ac17691`), and a handful of older
> tables (`carddav_sync`, `api_tokens`, `reminder_completions`, `calendar_subscriptions`/
> `calendar_event_links`, `relationships.related_contact_id`) that item 1's cascade-delete fix hadn't
> covered and FK enforcement couldn't help either, since their parents are soft-delete-only (`14fc7b0`);
> the fourth is root-caused and written up as **item 12** (2026-07-29), not yet fixed: `i18n.IsValidLanguage`
> normalizes through `normalizeLanguage`'s always-falls-back-to-English behavior, so it accepts any input —
> `UpdateLanguage`'s rejection branch is dead code and the raw unnormalized value gets persisted to
> `user.Language`. Tier 6 broadened from UI-only to also carry non-critical test-coverage expansion, and
> broadened again to add a **legacy-representation/dead-code audit** (candidates: `Contact.VCardExtra`,
> `RelationshipEdge.LegacyRelationshipID` + the backfill CLI tools once §3d ships, ~35 migrations worth
> squashing, more dead scaffolding like the one duplicate frontend `Relationship` type §3d already found) —
> prompted directly by §3d turning out bigger than expected, and deliberately scheduled *first* within Tier 6
> (ahead of UI polish) since its safe window closes once real production data exists, unlike the other two
> Tier 6 items which have no such deadline.
> **§3d fully done and merged** (2026-07-30, WP1+WP2+WP3 in `4b27794`, WP4+WP5 in `3b0d42d`) —
> `RelationshipEdge` is now the CRM's sole relationship representation end to end: full CRUD API, full
> frontend UI, `GetGraph`, and the CSV data export all read/write it. The legacy `models.Relationship`
> stack (model, controller, 5 routes, `AddRelationshipDialog.tsx`/`RelationshipList.tsx`/
> `useRelationships.ts`/`api/relationships.ts`, and its one-time `cmd/backfill-relationship-edges` backfill
> CLI) is fully removed, including a new migration (`000035_drop_relationships_table`) dropping the table
> itself. WP4 found `admin_user_controller.go`'s cascade-delete already covered `RelationshipEdge` (added
> by the earlier Tier 3c cascade-delete sweep, before this WP existed) — nothing to rewire there. A real
> scope correction surfaced during WP5: the backlog's Tier 6 audit had filed `cmd/backfill-relationship-
> edges`'s removal as later, deferrable cleanup, but that's not actually deferrable — it references
> `models.Relationship` directly and would not compile once the model is gone, so its removal was folded
> into WP5 out of necessity. `RelationshipEdge.LegacyRelationshipID` has no such compile dependency and
> stays, genuinely deferrable to that Tier 6 audit as originally planned. **All of Tier 3 (3a/3b/3c/3d) is
> now done. Item 12 also done** (2026-07-30, `33113c2`) — `i18n.IsValidLanguage` genuinely rejects
> unsupported/empty language codes now instead of silently accepting anything. **Tier 3 is fully closed.**
>
> **Re-groomed 2026-07-30 into a single ticket board** (see the next section) merging everything still
> open here with the remaining WPs in `92-delivery-roadmap.md` — **16 tickets to alpha, 16 after**.
> Tiers 4/5/6 are superseded as a *plan* by that board and retained only for their detail. The re-groom's
> headline finding: five backend entities are fully built but unreachable by any user (Household,
> Circle/Tag, LifeEvent, custom-fields v2), so the board's Phase A activates those first — which also
> closes P5's still-open acceptance gate. The alpha cut line was then drawn by classifying each P6–P10
> ticket on whether it changes data/contract shape: only 3 of 12 do, so 9 moved past alpha. **Next up is
> T1** (Household CRUD + suggestion trigger).

## How to read this

Tiers are ordered by "best use of time for immediate impact," not by when the idea was conceived.
Within a tier, do items top-to-bottom. Re-groom (re-run this judgment call) whenever a tier completes,
a security concern surfaces, or the user's priorities change — don't treat the ordering below as fixed
forever.

**As of the 2026-07-30 re-groom, the ticket board below is the execution order.** The Tier 0–6 sections
after it are retained as the historical record and as the detailed source material each ticket cites —
they are no longer the thing you read to decide what to do next.

# Ticket board — remaining work (re-groomed 2026-07-30)

Everything still open, from this file **and** `92-delivery-roadmap.md`, merged into one ordered set of
tickets. One ticket per feature/enhancement/change, each independently completable on its own branch
(per the branch-per-concern discipline used throughout).

**Ordering principle (user's call, 2026-07-30): activate what's already built before starting new
phases.** The survey behind this re-groom found five backend entities that are fully implemented but
**unreachable by any user** — the single largest gap in the project, and one the old tier structure hid
by filing it as "Tier 4 exception: WP-84c's deferred frontend/migration half":

| Entity | Model | Service/logic | HTTP routes | Frontend | Reachable? |
|---|---|---|---|---|---|
| `Household`/`HouseholdMember` (WP-83) | yes | yes (`GenerateHouseholdSuggestions`) | **none** | **none** | **no** |
| `Circle`/`Tag` (WP-84/84c) | yes | yes | yes | **none** (18 files still read flat `Contact.Circles`) | **no** |
| `LifeEvent` (WP-84) | yes | — | yes | **none** | **no** |
| `FieldDefinition`/`FieldValue` (WP-84b) | yes | yes (validation + projection + backfill CLI) | **none** | **none** (v1 still live) | **no** |
| `RelationshipEdge` (§3d) | yes | yes | yes | yes | **yes** — the only one |

Two knock-on facts that make this the right thing to fix first:
- **P5's own acceptance gate is still open.** `92.1` requires "legacy relationship + circle data migrated
  (dry-run-verified)". The relationship half closed with §3d; the circle half has never run.
- **§3d WP3 shipped a suggestion-review UI that can never fire.** `GenerateHouseholdSuggestions` has zero
  HTTP callers, so no `status: suggested` edge can exist in a running app. T1 below is what turns that
  already-written Accept/Reject surface on.

**Phase-to-alpha framing (user's clarification, 2026-07-30):** real production data does **not** exist
until after the alpha-readiness phase (Phase D) — alpha is the milestone at the end of Phase D, not at
the end of the board. Everything above that line is inside the safe, no-prod-data window; everything
below it is not, which is why the cut line below was drawn by asking, per ticket, *does this change the
shape of data or contracts alpha will populate?* The pre-real-data cleanup (T22) therefore sits at the
**end** of that window rather than being rushed forward. "Tier 6" was never "polish someday" — it is the
release-readiness gate.

## The board

| # | Ticket | Size | Depends on | Source |
|---|---|---|---|---|
| **Phase A — activate what's built** | | | | |
| T1 | Household CRUD API + suggestion trigger + review wiring | M | — | WP-83, §3d WP3 |
| T2 | Circle/Tag user-assisted triage migration | M | — | WP-84c-i, `91.5` |
| T3 | Circle/Tag backend call-site rewiring | S–M | T2 | WP-84c-ii |
| T4 | Circle/Tag frontend rewiring (~18 files) | L | T3 | WP-84c-iii |
| T5 | LifeEvent frontend + timeline surface | M | — | WP-84, `91.6`/`91.8` |
| T6 | Custom fields v2 — API surface | M | — | WP-84b, `94` |
| T7 | Custom fields v2 — frontend, backfill run, retire v1 | L | T6 | WP-84b, `94.6` |
| T8 | OpenAPI coverage + spec/route drift test | M | T1–T7 | `92.9` |
| **Phase B — independent, not gated by P5** | | | | |
| T9 | WP-97 selective field export + sensitivity gating | L | — | `92.6b` |
| **Phase C-pre — the only P6–P10 work that must precede alpha** | | | | |
| T12a | Activity/LifeEvent sync primitives (ETag column + backfill) | S | — | `92.3` prereq |
| T17 | WP-92 change feeds + cursor pagination | M | T8 | `92.5` |
| T20a | WP-95 Preferences — migrate `Contact.FoodPreference`, project `hobby` | M | — | `92.6`, `91.9` |
| **Phase D — alpha readiness (the pre-real-data gate)** | | | | |
| T22 | Legacy-representation / dead-code audit + migration squash | L | all above | Tier 6 |
| T23 | UI polish — typography, icons, strings | M | — | Tier 6 |
| T24 | Non-critical test-coverage expansion | M | — | Tier 6, `45` |
| T25 | Known small functional gaps sweep | S | — | Tier 0 notes |
| | **→ ALPHA — real data begins here** | | | |
| **Phase C-post — P6–P10 remainder (all additive; safe after alpha)** | | | | |
| T10 | WP-85 graph traversal + multi-hop chains | M–L | — | `92.2` |
| T11 | WP-86 search synonyms, household scope, FTS5 | L | T10, T1 | `92.2` |
| T12b | WP-87 serve Interactions/LifeEvents as CalDAV | L | T12a, T5 | `92.3` |
| T13 | WP-88 two-way calendar sync ⚠ policy call | M–L | T12b | `92.3` |
| T14 | WP-89 external-link substrate (ExternalIdentity/Activity) | M | — | `92.4`, `91.12` |
| T15 | WP-90 Immich level 1 (linking) | M | T14 | `92.4` |
| T16 | WP-91 Immich level 2 (enrichment) | M | T15 | `92.4` |
| T18 | WP-93 event history / audit trail | L | T17 | `92.5` |
| T19 | WP-94 CadencePolicy + relationship health | L | T5 | `92.6`, `91.10` |
| T20b | WP-95 Gift tracking | M | T5 | `92.6`, `91.11` |
| T21 | WP-96 ConversationAgenda | M | T5 | `92.6`, `91.11` |
| **Post-alpha — other** | | | | |
| P1 | Contact sharing — one-time filtered copy | **M** | T9 | Tier 5 |
| P1b | Contact sharing — standing/live share + permission model | XL | P1 | Tier 5 |
| P2 | Other integrations (Dawarich, Jellyfin, Paperless-ngx, …) | — | T14 | `92.7` |
| P3 | AI / Ollama layer | — | most of the above | `92.7`, `90` D1 |
| P4 | Local-model code-gen pilot | — | mobile client work | `80` |

**Alpha cut line — decided 2026-07-30.** Phase C was originally a single block of 12 tickets sitting
between the working app and alpha, which put alpha a long way out for capability depth (search, CalDAV,
Immich, sync, cadence) layered onto a CRM that would already work without any of it. Each Phase C ticket
was then classified by whether it *changes the shape of data or contracts that alpha will populate*, and
only three came back yes — so Phase C is split, and alpha now cuts after **A + B + C-pre + D**.

The classification, with the evidence behind it (checked against the code, not inferred from the WP text):

| Ticket | Verdict | Why |
|---|---|---|
| T12a ETag primitives | **pre** | `ETag` exists only on `Contact`/`ContactSubscription`. `Activity` got `UUID` in WP-84 but no ETag; `LifeEvent` has neither. CalDAV needs a per-resource ETag, so this is an additive column + backfill on `activities` — same shape as WP-84's own `activities.uuid` backfill (migration `000030`), a proven pattern here. Split out of WP-87 so the cheap schema half lands pre-alpha without dragging a whole CalDAV server with it. |
| T17 change feeds | **pre** | Not a data risk — an **API-contract** risk. Every list endpoint returns offset-shaped `{total, page, limit}`; WP-92 explicitly replaces that with cursors ("not offset — for large histories"). Free to break while we are the only consumer; a versioning scheme once anything external depends on it. Also interacts with T8 — publishing the OpenAPI spec and *then* flipping pagination means publishing the contract twice. |
| T20a Preferences | **pre** | WP-95's "migrate `FoodPreference`" moves a live, populated field that spans 13 files (model, `contactmodel` envelope, `contact_record`/`_reverse`, import, CSV export, 5 frontend files) into a new entity. Structurally the *same* migration as `Contact.Circles`→Circle/Tag, which is already pre-alpha as T2 — doing one before and one after would be incoherent. |
| T10 graph traversal | post | Pure read layer: recursive CTEs over `relationship_edges`, inferred relations "computed, not stored" per `92.2`. Zero schema. |
| T11 search + FTS5 | post | No FTS5 anywhere today, so this adds virtual tables + triggers + an index backfill — but that index is *derived*, rebuildable from source at any time. A post-alpha build is a re-runnable index job, not a destructive migration. |
| T12b CalDAV serve | post | Once T12a's primitive exists, serving is read-side and additive. |
| T13 two-way sync | post ⚠ | Additive, but carries a real policy call — see its detail below. Gated behind T12b anyway. |
| T14–T16 substrate + Immich | post | All-new tables; nothing existing changes. `Activity.ExternalRef` is already in place waiting for it (WP-84). |
| T18 audit trail | post | Pure-additive new log table. The cost of deferring is not risk but **lost history** — an audit log only knows what happened after it is switched on, so alpha-period undo/debugging is unrecoverable. A judgment call, not a hazard. |
| T19 cadence | post | New `CadencePolicy` table; health is derived from the timeline, not stored. **Better deferred** — cadence computed against a real interaction history is immediately meaningful, whereas pre-alpha it computes over nothing. `Activity.Qualifying()` already exists and has had no consumer since WP-84. |
| T20b gift tracking | post | New entity, unrelated to T20a's migration — which is why WP-95 was split. |
| T21 agenda | post | New entity, contact-view surface. Additive. |

## Ticket detail

### T1 — Household CRUD API + suggestion trigger + review wiring

**What's necessary.** `controllers/household_controller.go` following `circle_controller.go`'s idiom
exactly, including real nested member sub-resources (`POST/DELETE /households/:id/members`) rather than a
bulk-replace field, and a `409 ErrAlreadyExists` on duplicate add — that precedent is already set. A
dedicated trigger endpoint (e.g. `POST /households/:id/suggest-relationships`) that calls the existing
`services.GenerateHouseholdSuggestions` and persists the returned `status: suggested` edges idempotently.
Routes in `routes.go`. Tests including a real-DB one (`database.InitDB`, not `AutoMigrate` — this repo's
recurring GORM-column-tag bug class). Frontend: a household management surface (create, name, type, add/
remove members) plus whatever wiring makes generated suggestions appear in `RelationshipEdgeList`'s
already-built suggested section.

**Why here.** Highest payoff-per-unit-effort of anything remaining: the review UI, the suggestion engine,
and the edge model all already exist and are tested — this ticket is the missing connective tissue
between them. It is also the project's first live use of the propose-then-approve pattern that `92.7`
says the eventual AI layer must reuse.

**Watch for.** Split backend-first if it runs long, but note that a backend-only T1 does not satisfy the
ticket's own purpose (reachability), so the minimal UI is in scope, not optional.

### T2 — Circle/Tag user-assisted triage migration

**What's necessary.** A UI that walks every distinct existing `Contact.Circles` string and has the user
classify each as a `Circle` (a group you belong to) or a `Tag` (a label), then writes real `Circle`/`Tag`
+ membership rows via the CRUD API that already exists. `91.5` is explicit that this must **not** be an
automated heuristic — "a light user-assisted step" is the requirement, so a mapping-by-guess
implementation fails this ticket. Dry-run/preview before committing, matching the discipline of every
other migration in this repo.

**Why here.** This is the half of P5's acceptance gate that has never run. It must precede T3/T4 because
those switch reads over to the new tables — doing them first would read empty tables.

### T3 — Circle/Tag backend call-site rewiring

**What's necessary.** Move the ~5 backend sites still reading/writing flat `Contact.Circles` onto the real
entities: `contact_controller.go`'s `GetCircles` and its `json_each`-based JSON-array filtering, and
`import_service.go`'s circles/tags/groups/labels synonym-mapping. That last one needs real thought rather
than a find-and-replace: it currently maps **all four** of those vocabularies onto the single flat
`circles` field, and now that `Tag` exists as a distinct destination the mapping has to split by target,
not just change where it writes.

### T4 — Circle/Tag frontend rewiring

**What's necessary.** The ~18 frontend files consuming `circles` as a flat string array — chips, filters,
graph nodes, dashboard, import dialog — move onto the Circle/Tag entities and their CRUD API. Retire the
flat field once nothing reads it.

**Why here.** Largest single frontend surface left. Completing it closes P5's acceptance gate outright.

### T5 — LifeEvent frontend + timeline surface

**What's necessary.** `api/lifeEvents.ts`, a contact-detail surface for viewing/creating/editing life
events, and integration into the timeline (`91.8`) alongside notes and activities. `PartialDate` rendering
needs care — year-only and month-day-only values are both legal and already round-trip correctly on the
backend.

**Why here.** CRUD routes already exist and are tested; this is pure activation. It also unblocks T12
(CalDAV serves LifeEvents out), T19, T20, and T21, all of which read the timeline as source of truth.

### T6 — Custom fields v2, API surface

**What's necessary.** CRUD controllers + routes for `FieldDefinition` and `FieldValue`, wiring the
existing `services.ValidateFieldValue` type-dispatch into the write path, and honoring `91.13` sensitivity
on read. The model, validation, standards projection, and the `cmd/backfill-custom-fields` migration tool
all already exist and are tested — none of that needs rebuilding.

### T7 — Custom fields v2 frontend, backfill run, retire v1

**What's necessary.** Replace `CustomFieldsSettings.tsx` and the other v1 consumers
(`ContactInformation.tsx`, `AddContactDialog.tsx`, `ContactsPage.tsx`, `api/users.ts`, `api/contacts.ts`)
with the typed v2 equivalents, including per-type input rendering and the `FieldConstraints.Multi` list
case. Then run `cmd/backfill-custom-fields` for real (dry-run first — note its documented two-pass
limitation: a dry run against a fresh DB cannot show pass-2 successes), and retire the v1
`User.CustomFieldNames` + `Contact.CustomFields` columns and their CSV-export path.

**Why here.** v1 is what's actually live today; v2 is the unreachable parallel implementation. Leaving
both is exactly the "layer a new representation on top, bridge them, defer removal" pattern that T22
exists to clean up — so finish the cutover rather than adding to that pile.

### T8 — OpenAPI coverage + spec/route drift test

**What's necessary.** `openapi.yaml` currently documents **13 of ~70** route patterns — contacts, export,
import, and contact-subscriptions only. Everything else (activities, notes, reminders, circles, tags,
life-events, relationship-edges, households, graph, webhooks, calendars, api-tokens, users, admin) is
undocumented. Document them in the style WP-71 established for contacts, and add a test that fails when a
registered route has no spec entry, so this cannot silently rot again.

**Why here.** `92.9` makes this binding, not cosmetic: every new entity is supposed to get
summary/detail/OpenAPI treatment so a future mobile client targets one coherent spec rather than a
patchwork. Scheduled after T1–T7 so it documents the finished surface once instead of twice, but the
drift test is worth adding as soon as it's convenient.

### T9 — WP-97 selective field export + sensitivity gating

**What's necessary.** Full scope is in `92.6b` and should be read directly — it is unusually well
specified, including two user clarifications. Summary: a coarse-grained field-selection model over
`contactmodel.Card`'s top-level sections, applied by filtering the `Record`/`Card` **before** it reaches
an exporter so all three adapters (vCard 3, vCard 4, JSContact) inherit it with zero adapter changes; plus
a picker UI wired into the existing export flow. Sensitive items (`91.13`) are opt-**in**, ordinary
categories opt-**out**, same control, opposite default.

**The non-obvious part, already flagged in `92.6b` and worth re-flagging:** the sensitivity override is
not purely a UI concern. `projectRelationshipEdges` in `models/contact_record.go` enforces
`Sensitivity: normal` as an unconditional SQL filter with no override parameter — so "include these
sensitive edges this once" needs a real change there, not just a Card filter sitting in front of an
otherwise-unchanged projection. Also binding: an unchecked box is explicitly **not** sufficient gating;
a sensitive item must be visually distinct and behind a deliberate extra action before its control is even
interactive. This is specified as foot-gun prevention, not decoration.

**Why here.** Depends only on P0 + WP-73, both long done, so it is genuinely pickable at any time. It also
builds the field-selection model and UI that post-alpha contact sharing (P1) is supposed to reuse rather
than reinvent.

### T12a — Activity/LifeEvent sync primitives (ETag)

**What's necessary.** An `etag` column on `activities` (and `life_events` if it will be served too),
generated and refreshed the way `Contact.ETag` already is — `models/contact.go` computes
`e-{id}-{updatedAt.Unix()}` in its create/save hooks — plus a migration backfilling existing rows,
following `000030`'s `activities.uuid` backfill precedent exactly.

**Why here, split out of WP-87.** This is the only genuinely schema-shaped part of the CalDAV work, and
it is small. Splitting it out means the cheap column lands while the tables are empty, without pulling a
whole CalDAV server implementation (T12b, sized L) in front of alpha to get it.

### T10–T21 — the rest of the P6–P10 capability phases (WP-85 … WP-96)

Scope for each is in `92-delivery-roadmap.md` `§92.2`–`§92.6` and is not duplicated here; that doc remains
the source of truth. Pre/post-alpha placement and its evidence are in the classification table above.
Re-groom notes worth carrying:

- **T11 (search).** Household-scoped queries ("everyone in the Smith household") now genuinely depend on
  T1 — households are unreachable until then, so that half of WP-86 has nothing to query.
- **T12b (CalDAV serve).** Serves Interactions **and LifeEvents** out, so it depends on T5 for LifeEvents
  to be real, user-visible objects rather than an API-only entity, and on T12a for the ETag primitive.
- **T13 (two-way sync) — the one post-alpha ticket carrying a real warning.** Its risk is not schema but
  **reconciliation policy applied to real data**. `CalendarEventLink` today reconciles one-way via
  `ContentHash`, with no ETag/If-Match primitive; two-way means deciding what wins when both sides
  changed. This repo has a precedent that must **not** be inherited by default: `reconcileContactSync`
  deliberately full-overwrites local edits on remote change (intentional, and pinned by Tier 3c item
  11a's test). Shipping that same policy onto real synced calendar data would silently discard real user
  edits. Settle the merge semantics explicitly, and prefer testing against a scratch calendar over a live
  one.
- **T14–T16 (Immich).** Substrate first, deliberately, so no integration grows bespoke tables — keep that
  order even post-alpha. Immich level 3 (bidirectional) stays deferred pending an upstream Immich
  capability, which is a dependency, not a scheduling choice.
- **T17 (change feeds).** Depends on T8 in practice: cursor pagination over "the entity APIs" is only
  meaningful once those APIs are specified, and it should land in the spec at the same time — which is
  also why it is pre-alpha, so the spec is published once with its final pagination shape.
- **T19–T21.** All read the timeline (`91.8`), hence the shared T5 dependency. T19 carries the rule that
  **recording a qualifying interaction resets cadence, not completing a task** — `Activity.Qualifying()`
  already exists for exactly this and has had no consumer since WP-84.

### T22 — Legacy-representation / dead-code audit + migration squash

**What's necessary.** This ticket *is* the audit — identify candidates, then decide keep/remove/defer per
candidate, the same methodology Tier 3c item 11 used. Known starting candidates: `Contact.VCardExtra`
(its own doc comment says `Passthrough` supersedes it "in spirit" but nothing ever confirmed it dead);
`RelationshipEdge.LegacyRelationshipID` (vestigial since §3d WP5 removed the model it referenced);
`cmd/backfill-custom-fields` and `cmd/backfill-contact-records` (spent one-shot tools — `cmd/backfill-
relationship-edges` was already removed in §3d WP5 out of compile necessity); squashing ~35 incremental
migrations to a single clean baseline; and a broader dead-export sweep (a Go dead-code tool plus a
frontend unused-export check) rather than assuming the duplicate `Relationship` type §3d found was the
only instance.

**Why here.** The migration squash in particular is safe **only** while no deployment needs a stepwise
upgrade path preserved — which is true right up until alpha and false forever after. Placing it at the end
of Phase D puts it as late as possible while still inside that window, so it sweeps up debt created by
Phases A–C rather than running before them and missing it.

### T23 — UI polish

**What's necessary.** Treat this as a method, not a checklist: walk the running app flow by flow and fix
what reads as unpolished. Three calibration examples are recorded in the Tier 6 section — a typography
audit for consistency/intent rather than inheritance; adding MDI (`@mdi/js`) alongside the existing
`@mui/icons-material` and using it where it has a better semantic match (notes list, add-note, network
graph are named starting points), without ripping out every MUI icon; and a copy review, of which the
Settings page's confusing "Profile" sub-label is the one concrete known instance.

### T24 — Non-critical test-coverage expansion

**What's necessary.** The packages `45-test-coverage-closure.md` never scoped at all: `config` 24.2%,
`database` 37.8%, `routes` 0%, `errors` 0%, `i18n` (now has its first tests, from item 12), `logger` 0%,
and the remaining one-shot `cmd/*` tools. Needs a scoping pass to decide which are worth covering versus
accepted as low-value — `cmd/migrate`'s thin CLI wrapper plausibly isn't. Explicitly **do not** chase the
percentage for its own sake.

### T25 — Known small functional gaps sweep

**What's necessary.** Small, real, individually-not-worth-a-ticket issues found in passing and recorded
rather than fixed. The one confirmed instance: `AddressFields`/`toLegacyContact` round-trip only 5 address
component kinds (street/locality/region/postcode/country), so a CardDAV-imported address using other
JSContact kinds (apartment, floor, district, …) silently loses them on the next edit-and-save through the
UI. Narrow — only affects externally-imported addresses with non-standard structure — but it is real data
loss, and the fix belongs in the adapter (`api/contacts.ts`), not the components. Sweep for others while
here.

**Why here.** Data-loss bugs should be closed before real data exists, which puts this inside Phase D
rather than after it.

### P1 / P1b — contact sharing between users

**Re-sized 2026-07-30 from XL to M, and split.** The old XL was not wrong so much as *answering a
different question*: Tier 5's own text below imagines a **standing, live, permissioned share** —
"data model for shared-vs-private fields, permission model," a share that re-syncs and re-confirms
"on every field newly marked sensitive after the share was created." That is genuinely XL. But the
near-term feature is much smaller, and conflating them inflated the ticket.

**P1 — one-time filtered copy (M).** Sender picks a contact and a field selection, the system emits a
filtered copy, the recipient accepts and it lands in their account. Almost every piece already exists:

| Step | What it reuses |
|---|---|
| Field selection + filtering | T9/WP-97's picker and its `Record`/`Card` filter function — built to be reused here, not reimplemented (`92.6b` says so explicitly) |
| Serialize the filtered record | `jscontact.Adapter{}.Export(record)`, already the engine behind `ExportContactsAsJSContact` |
| Parse on accept | `services.ParseJSContact` — already exists, already feeds the shared import path |
| Duplicate detection + preview | `services.DetectDuplicate` + the existing import-session preview/confirm flow |
| Write into the recipient's account | `MergeImportedContact` / `ApplyRecordToContact`, the one shared Record→Contact writer |

So the genuinely **new** surface is small: a `ContactShare` entity (from-user, to-user, payload, status
pending/accepted/declined), create/list/accept/decline endpoints, and two frontend pieces — a share
dialog wrapping T9's picker, and an incoming-shares inbox. That is M, not XL.

**Safe post-alpha? Yes — additive, with one decision to make deliberately.** It adds a new table and
changes no existing shape, and unlike T13 the write path is user-initiated per acceptance with an
explicit preview, not a background sync. The decision worth making consciously rather than inheriting:
**what accepting a share does when the recipient already has that person.** `MergeImportedContact`'s
policy is "incoming wins if non-empty, existing survives if blank" (confirmed and pinned by Tier 3c item
11c) — which may be wrong here, since a shared copy arguably should not overwrite the recipient's own
notes and edits on someone they already track. Decide create-new vs. merge-into-match vs. ask, rather
than defaulting into the import path's existing behavior.

**One product caveat that is not a technical one.** The stated use case is two users on the same
instance, e.g. spouses. If alpha *is* that scenario, sharing may be a headline feature of alpha rather
than a follow-on — in which case pull P1 forward for product reasons, not safety ones. Technically it is
safe either side of the line.

**P1b — standing/live share + permission model (XL, deferred).** Everything Tier 5's section below
describes beyond the one-time copy: persistence for a live share, the shared-vs-private field model, the
permission model, and re-confirmation when a field is newly marked sensitive after the share exists.
Needs its own design pass before it can be broken into WPs. Do not start it as part of P1.

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
| 11 | **Correctness audit** — scoped 2026-07-29 via 3 parallel research agents surveying every candidate area (relationship graph, sync/import, validation/sensitivity, reminders/webhooks). Finding: most business-logic decision points survived items 1-10 and this session's own bug-hunting already behaviorally tested — `relationship_type_registry`/`relationship_edge` inverse resolution, `contact_record.go`'s sensitivity/status export gates, `household_service.go`'s classification engine, `custom_field_service.go`'s type-dispatch validation, `webhook_service.go`'s event-type fan-out, `ValidateJSONMiddleware`'s unusual-but-valid positive case, `import_session.go`'s partial-success semantics, and `import_controller.go`'s session-format guards all already have real assertions on the decision, not just "doesn't error." The original "5-10 areas, do this last" framing overshot; real remaining surface is 3 small WPs + 1 flagged design question. | | |
| 11a | `services/contact_sync_service.go`'s `reconcileContactSync` — on any remote CardDAV change, `ApplyRecordToContact` fully overwrites the local row; no test proves whether a local-only edit to a field untouched by the remote vCard survives a sync or is silently discarded | small | **DONE** (2026-07-29). The full-overwrite behavior itself is confirmed intentional (documented in `ApplyRecordToContact`'s own doc comment, shared with the REST PUT path; no model tracks per-field modified-since-sync state, so field-level merge isn't a small fix) — pinned down with `TestReconcileContactSyncOverwritesLocalEditsOnRemoteChange` plus a doc comment at the call site. **Writing that test surfaced a real, live, previously-unknown bug**, not the hypothesized one: `models.applyPhones` cleared `c.Phones` but left the `c.Phone` scalar untouched whenever the incoming Record had no phones at all — unlike its sibling `applyEmails`, which always resets `c.Email` (falling back to `proj.PrimaryEmail`). A contact whose last phone number was removed via REST PUT, CardDAV sync, or VCF import kept a stale `Phone` scalar even though `Phones` correctly went empty. Fixed by mirroring `applyEmails`'s pattern with the already-computed-but-unused `proj.PrimaryPhone`. Hand-verified: reverted the fix, confirmed the new test fails, restored. |
| 11b | `services/reminder_service.go`'s `SendReminders` eligibility filter (`completed=false AND email_sent=false`) — only ever tested with reminders already in the eligible state; no test seeds an already-completed/already-sent reminder alongside an eligible one to prove exclusion | small | **DONE** (2026-07-29, `b1d3d92`). Filter confirmed **correct, not buggy** — `TestSendReminders_ExcludesCompletedAndAlreadySentReminders` seeds all three states in one pass and asserts only the eligible reminder is picked up and the other two are left untouched. Hand-verified: temporarily broadened the WHERE clause to drop the `completed`/`email_sent` conditions, confirmed the test fails (all 3 reminders picked up, already-completed one gets mutated), restored. |
| 11c | `services/import_service.go`'s `MergeImportedContact` — "incoming wins" policy asserted for exactly one scalar field, one direction; untested against array fields (Emails/Phones/Addresses/Circles) and the "existing survives when incoming blank" path | small | **DONE** (2026-07-29, `15bcf69`). Policy confirmed **correct and uniform, not buggy** — every field (scalar and array) follows the same "incoming wins if non-empty, existing survives if blank" guard. `TestMergeImportedContact_ArrayFieldsOverwriteWhenIncomingNonEmpty` and `TestMergeImportedContact_ExistingSurvivesWhenIncomingBlank` cover both directions for every multi-valued field (Emails/Phones/Addresses/URLs/IMPPs/Circles), not just the one scalar field the pre-existing test covered. Hand-verified: temporarily removed the `Phones` guard (unconditional overwrite), confirmed the blank-incoming test fails, restored. |
| 11d | `controllers/graph_controller.go`'s `GetGraph` reads only the legacy `models.Relationship` table — structurally blind to `RelationshipEdge` (the WP-80+ graph model). **Superseded 2026-07-29 — moved to its own section, `3d` below**, once scoping revealed this is a feature-migration WP, not a cleanup: `RelationshipEdge` has no CRUD API or frontend at all today, so the legacy model is currently the CRM's only working way to manage relationships, not a redundant parallel one. See `3d` for the real breakdown. |
| 12 | **Fix `i18n.IsValidLanguage` accepting any input** (found in passing during item 10's Phase 3b work) | trivial–small | **DONE** (2026-07-30, `33113c2`). New `i18n.NormalizeSupportedLanguage(lang) (string, bool)` does the same lowercase/first-BCP-47-subtag normalization as `normalizeLanguage`, but never falls back to `DefaultLanguage` — an empty or unrecognized code now genuinely returns `ok=false` instead of silently normalizing to `"en"`. `IsValidLanguage` is now a thin wrapper over it. `normalizeLanguage` itself (used only by `T()`'s display-lookup path) is deliberately unchanged — its always-return-something-displayable fallback is correct there, it was only wrong to reuse for validation. `UpdateLanguage` now rejects unsupported codes for real and persists the *normalized* value (previously stored the raw input verbatim even when valid, e.g. `"DE-AT"` instead of `"de"`). **Open design question resolved**: empty input is rejected, not treated as "reset to default" — this is a validation gate for explicit user input, not `T()`'s display-fallback path. Hand-verified: reverted `NormalizeSupportedLanguage` to reuse `normalizeLanguage`'s fallback, confirmed 4 tests fail exactly as expected, restored. New `i18n/i18n_test.go` (the package's first test file) plus two new `user_controller_test.go` cases. |

### 3d. Relationship model consolidation — retire the legacy `models.Relationship` in favor of `RelationshipEdge`

**Scoped 2026-07-29, sized `large`, scheduled before Tier 4** (the user's explicit sequencing — do this while
it's still safe/cheap, i.e. before Phase 6 completes and production data starts existing). Originated from
item 11d above: closing `GetGraph`'s blindness to `RelationshipEdge` looked like a small rewire until full
scoping (2 parallel research agents, backend + frontend) found the real picture is inverted from what the
original 11d framing assumed.

**Why this is a feature migration, not a cleanup:** `RelationshipEdge` (the WP-80+ graph model — confirmed/
suggested status, directional edges, a real relation-type registry) has **zero CRUD API and zero frontend
consumers today**. It's currently written only by the one-way `cmd/backfill-relationship-edges` migration
tool and by `household_service.go`'s auto-suggestion engine. Meanwhile the legacy `models.Relationship` is
the CRM's **only working relationship-management feature**: a full 5-route REST API
(`relationship_controller.go`), a full frontend UI (`AddRelationshipDialog.tsx`, `RelationshipList.tsx`, the
"Relationships" tab in `ContactInformation.tsx`, wired up in `ContactDetailPage.tsx`), and even the `/graph`
endpoint itself still reads the legacy table, not `RelationshipEdge`. Removing the legacy stack today, without
building the replacement first, would delete the app's only way to record who's related to whom — not a
safe cleanup. WP-81's migration tool already proves the *data shape* is a lossless superset (promotes even
name-only relationships into a thin Contact + edge), so nothing here is about data loss risk — it's about
missing functionality that has to be built before the old functionality can go.

**Work packages, in order:**

1. **Backend: `RelationshipEdge` CRUD controller + routes** — **DONE** (2026-07-30, `30e61d9`, merged
   `61aa215`). Flat `/relationship-edges` resource (mirrors `life_event_controller.go`'s newer idiom, not
   `relationship_controller.go`'s nested one): create/get/list (filterable by `contact_id`, matching either
   direction/`status`/`type`)/update/delete, plus a dedicated `PATCH .../accept` for `suggested` edges (no
   consumer existed anywhere until now). Design decisions resolved: user-created edges are always
   `Source: user-confirmed, Confidence: 1.0, Status: confirmed` (server-derived, never client-settable);
   `Directional` is always derived from the registry (`!IsSymmetricRelationType`), matching
   `household_service.go`'s existing precedent, not exposed as user-facing. A real design gap surfaced during
   planning and resolved: creating a relationship to someone who isn't a Contact yet (the legacy "manual
   entry" case) accepts an inline `{name, gender, birthday}` and creates a thin `Contact` in the same DB
   transaction as the edge write, mirroring `cmd/backfill-relationship-edges`'s own thin-contact field
   mapping. 27 tests; hand-verified the transactional rollback (temporarily removed the transaction wrapping,
   confirmed the orphan-thin-contact test fails, restored) and the `relation_type` validator through the real
   middleware.
2. **Backend: rewire `GetGraph`** — **DONE** (2026-07-30, `41dfdf3`, merged `61aa215`). Reads
   `RelationshipEdge` filtered to `Status: confirmed` only — `suggested` edges must never appear as graph
   fact, per the model's own doc comment. Contact nodes now also carry `VCardUID` so edges (which reference
   contacts by `VCardUID`, not the numeric ID the graph's node scheme uses) resolve correctly; an edge is
   skipped (not left dangling) if either endpoint isn't in the current node set. `GraphEdge`/`GraphResponse`
   DTO shape unchanged, so `NetworkGraph.tsx`/`NetworkLegend.tsx` needed no frontend change (confirmed via
   exploration: they only use edge `type`/`label` for display). Real-DB integration test added
   (`relationship_edge_real_db_test.go`, against a `database.InitDB`-migrated file DB, not `AutoMigrate`) —
   full WP1+WP2 create/list/update/accept/delete/graph cycle end to end, no GORM column-tag surprises against
   the real migration SQL. Full suite green throughout, no regressions.
3. **Frontend: new CRUD UI** — **DONE** (2026-07-30, `9ad806a`, merged `4b27794`). New
   `api/relationshipEdges.ts`, `hooks/useRelationshipEdges.ts`,
   `components/RelationshipEdgeDialog.tsx`/`RelationshipEdgeList.tsx` replace `AddRelationshipDialog.tsx`/
   `RelationshipList.tsx`/`useRelationships.ts` (now fully unreferenced — left in place, removal is WP5's
   job). Small companion backend addition: a `?vcard_uid=` batch filter on `GET /contacts` (WP0,
   `3e3c7b5`), since `RelationshipEdge` carries no nested contact data to resolve names/links from.
   Direction semantics (`type` describes `SourceID`'s role relative to `TargetID`, verified against
   `relationship_type_registry.go`/the WP-81 migration tool's own doc comments) — creating from a contact's
   page always sends `target_id: viewedContactUid`, so a dropdown label like "Mother" still describes the
   *other* party, exactly matching the legacy dialog's semantics. 17 unit tests cover every
   asymmetric/symmetric token in both directions; verified for real in a running browser too (created Bob
   as Alice's Parent → saw Child from Bob's page; edited from Bob's page to Mentor → saw Mentee from
   Alice's page; created a thin "Whiskers" contact as Alice's Pet → saw Owner from Whiskers' own page;
   `GetGraph` confirmed serving the new edges correctly). Editing never resends `source_thin`/`target_thin`
   (the backend always inserts a new Contact for a non-nil `*_thin` field, even on update) — the other
   party is read-only in edit mode, with a regression test. The confirmed/suggested split unifies what used
   to be two list sections into one (`?contact_id=` already matches either direction). Suggestion
   review (Accept/Reject) is built now despite nothing able to produce a `suggested` edge yet
   (`household_service.go`'s engine has no HTTP trigger) — verified via a component test with mocked data
   instead of fabricating a fake trigger. Two real bugs caught by the new component tests before shipping:
   invalid HTML nesting (a `Chip` inside a `Typography` `<p>`) and a duplicate "Suggested" string collision
   between the section heading and the status chip — both fixed. Full Playwright e2e run wasn't possible in
   this environment (the repo's `e2e/global-setup.ts` hardcodes port 7300 for both app and API, which was
   held by a concurrent session) — every scenario the new spec (`relationshipEdges.spec.ts`) covers was
   instead verified by hand in a real running browser, except the delete step (blocked by native
   `window.confirm` auto-dismiss under browser automation, already covered by WP1's real-DB `Delete` tests).
4. **Rewire the other legacy-model dependents** — **DONE** (2026-07-30, `1f43752`, merged `3b0d42d`).
   `admin_user_controller.go`'s cascade-delete needed no change: already covered `RelationshipEdge`, added
   by the earlier Tier 3c cascade-delete sweep before this WP existed. `contact_controller.go`'s
   `includes=relationships`/`Preload("Relationships")` (`GetContacts`/`GetContact`) turned out to be dead,
   not something to rewire — zero remaining frontend callers once WP3's `RelationshipEdgeList` shipped
   (it fetches via its own `/relationship-edges` endpoint, never off this response) — removed instead,
   matching this file's own `fields=` no-op-rather-than-reject precedent. `export_controller.go`'s CSV
   "RELATIONSHIPS" section was the one genuine rewire (real, still-reachable functionality): now reads
   `RelationshipEdge`, resolving source/target names via a `VCardUID`→contact map, deliberately including
   every status/sensitivity since this is the user's own full personal backup, not a share.
5. **Removal** — **DONE** (2026-07-30, `25b142b`, merged `3b0d42d`). `models/relationship.go`,
   `relationship_controller.go` (+ test), the 5 legacy routes, the `RelationshipInput` DTO,
   `Contact.Relationships`, the legacy cascade-delete calls in `DeleteContact`/`DeleteUser`, a new
   `000035_drop_relationships_table` migration (clean drop, down recreates the cumulative schema), the
   frontend files listed in WP3 plus `api/relationships.ts`/`hooks/useRelationships.ts`, and the dead
   duplicate `Relationship`/`GetRelationshipsResponse`/`CreateRelationshipResponse`/
   `DeleteRelationshipResponse` types in `types/index.ts`/`types/api.ts`. `e2e/relationships.spec.ts` was
   already gone (deleted during WP3). **Real scope correction found while implementing, not anticipated
   by this plan**: `cmd/backfill-relationship-edges` (WP-81's one-time migration tool) references
   `models.Relationship` directly and would not compile once the model was deleted — the backlog's Tier 6
   audit had filed its removal as later, deferrable cleanup, but that's not actually optional once WP5
   runs, so its removal was folded in here. `RelationshipEdge.LegacyRelationshipID` has no such compile
   dependency and stays, genuinely deferrable to that Tier 6 audit as originally planned. New
   `TestMigrationDropsLegacyRelationshipsTable` (real-DB check against the actual `InitDB` migration
   chain) confirms `000035` drops the table and `relationship_edges` is unaffected. Verified end-to-end in
   a real running browser too: created two contacts and a `parent_of` edge via the API, confirmed the
   contact detail page's Relationships tab renders the correct inverse label ("Child"), and confirmed the
   CSV export's new RELATIONSHIPS section renders the edge correctly.

Not required for this WP: no live data migration run (WP-81's tool exists for a real carryover if ever
needed later, but there's no production data to carry over now).

## Tier 4 — P6–P10 (search, CalDAV export, external links/Immich, sync, relationship-maintenance)

> **Superseded as a plan by the ticket board (2026-07-30); retained for detail.** These WPs are now
> tickets T10–T21 (Phase C), and this tier's two "exceptions" were promoted out of it: WP-97 is T9
> (Phase B) and WP-84c's deferred half is T2/T3/T4 (Phase A), reordered ahead of the P6–P10 work.

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

> **Superseded as a plan by the ticket board (2026-07-30); retained for detail.** **Split into P1 and
> P1b, and re-sized.** What this section describes — a standing, live, permissioned share — is P1b and
> is genuinely XL. But the near-term feature is a **one-time filtered copy** (P1), which is only **M**:
> the sender's field selection reuses T9/WP-97's picker and filter, serialization reuses the existing
> JSContact export adapter, and the recipient's accept path reuses `ParseJSContact` + `DetectDuplicate` +
> the existing import preview/confirm flow. The genuinely new surface is a `ContactShare` entity, four
> endpoints, and two frontend pieces. See P1's ticket detail for the reuse map and the one merge-policy
> decision it must make deliberately.

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

> **Superseded as a plan by the ticket board (2026-07-30); retained for detail.** Now tickets T22–T25,
> Phase D. **Re-framed by the user on 2026-07-30 and worth stating plainly, because the wording below
> gets it wrong:** this tier is not "polish someday" — it is the **alpha-readiness gate**, the cleanup /
> validation / polish that makes Mycorrhizal ready for actual use, after which it goes into alpha and
> real data starts existing. That also settles the intra-tier ordering debate recorded below: the
> dead-code audit's "window closes once production data exists" concern is real but not urgent, since
> production data arrives *after* this entire tier, not during it. T22 is therefore scheduled at the
> **end** of Phase D — as late as possible while still inside the safe window — so it also sweeps up
> whatever debt Phases A–C create, rather than running first and missing it.

Explicitly last priority — polish after everything else (Tiers 1–5) is ready, per the user's own framing.
Originally UI-only (fonts/icons/strings); broadened 2026-07-29 to also carry non-critical test-coverage
expansion, and broadened again same day to carry a legacy-representation/dead-code audit (see below,
scheduled *first* within this tier — unlike the other two Tier 6 items, its window closes once real
production data exists, so it shouldn't sit behind UI polish indefinitely even though the tier itself is
still last-priority overall). *(That last parenthetical is the part the 2026-07-30 re-framing corrects —
see the note above.)*

### Legacy-representation / dead-code audit

Added 2026-07-29, prompted by §3d: that item started as a small `GetGraph` rewire and turned out to be a
whole legacy subsystem (`models.Relationship`) still doing real work years after its replacement
(`RelationshipEdge`) existed, because this fork's build pattern across WP-80→84c was consistently "layer a
new representation on top of the old one, bridge them, defer full removal" — new nested `Card`/`CRM`/
`Passthrough` alongside legacy flat fields, typed `FieldValue` alongside the old `CustomFields` map, and so
on. **Given no production data exists yet, this is the cheapest point at which to sweep for other instances
of the same pattern and remove what's safe to remove** — every session after a v0.1 pre-release cut makes
this more expensive (real data to migrate, real upgrade paths to preserve).

Not yet audited — this item is the audit itself, methodology mirrors Tier 3c item 11 (identify candidates,
then decide keep/remove/defer per candidate, not a blind deletion pass). Known starting candidates, found in
passing rather than via a dedicated sweep:

1. `Contact.VCardExtra` — its own doc comment (`models/contact_record_reverse.go`) already says it's "being
   superseded by Passthrough in spirit," but was never actually removed or confirmed dead. Check what, if
   anything, still reads it as authoritative versus `Passthrough`.
2. `RelationshipEdge.LegacyRelationshipID` and the whole `cmd/backfill-relationship-edges` CLI tool — once
   §3d removes the legacy `models.Relationship` table, both become vestigial (nothing left to migrate from).
   Natural follow-on cleanup after §3d ships, not before.
3. The other one-shot backfill tools, same category as #2: `cmd/backfill-custom-fields` (the v1→v2
   `FieldValue` migration) and `cmd/backfill-contact-records`. Built for migrating during active development;
   check whether they're still needed pre-release or are now pure dead weight.
4. Migration history: ~35 incremental migration files for a repo with zero production data. Squashing to a
   single clean baseline schema is safe and normal pre-release hygiene here specifically because there's no
   live deployment needing a stepwise upgrade path preserved — re-evaluate this once real prod data exists,
   since the tradeoff flips.
5. Dead/duplicate scaffolding left behind after a model migration — one confirmed instance already found in
   passing (frontend `types/index.ts`/`types/api.ts` had an unused, duplicate `Relationship` type parallel to
   the real one in `api/relationships.ts`, per §3d's scoping). Worth a broader sweep (a Go dead-code tool
   plus a frontend unused-export check) rather than assuming that was the only one.

None of these have been scoped for real yet (sizes, exact removal surface) — that's the point of this item,
same as item 11 was for the correctness-audit candidates before it got broken down.

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

> **Now tickets P2/P3 on the ticket board, post-alpha.** Still "pulled in only when a concrete need
> arises" — placing them on the board gives them an ID and a dependency, not a schedule.

Unchanged from `92-delivery-roadmap.md §92.7`: other integrations (Dawarich/GeoPulse, Jellyfin,
Audiobookshelf, Paperless-ngx, Nextcloud), AI/Ollama layer. Pulled in only when a concrete need arises.

## Explicitly not re-ranked here

`80-local-model-pilot.md`'s deferral is independent of this backlog (re-enters when mobile client work
begins, per `92.9`).
