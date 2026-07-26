# 90 — Vision, decisions of record, and reconciliation with the existing codebase

> **What this is.** The strategic layer above the `00`–`80` fork-plan. It reconciles an external vision
> document ("Personal CRM / Meerkat Fork Technical Roadmap" — the "relationship OS" vision) with (a) what
> the fork-plan has already built, and (b) what Meerkat already shipped before this fork. It records the
> binding strategic decisions that fork the whole roadmap, and hands off to `91` (envelope data model),
> `92` (sequenced delivery roadmap), and `93` (integration spec template) for the details.
>
> The source vision doc lives outside the repo (user-supplied). This file is the durable, in-repo
> synthesis of it — treat this file, not the source doc, as authoritative going forward.

## 90.1 The one-sentence reconciliation

The vision doc's core principle — *"the internal model is a superset of the standards; standards are
just an interchange layer; new functionality exists internally first and is mapped to export only where
possible"* — **is exactly the hub-and-spoke architecture P0 already implemented.** The `CRMEnvelope` type
(`backend/contactmodel/envelope.go`) is already the designated home for "CRM-only data no format adapter
touches." Almost every feature the vision doc adds (relationship graph, households, life events, cadence,
preferences, gifts, conversation agenda, external links/activity) lives on that envelope side. This is a
continuation of the existing design, not a pivot away from it.

## 90.2 Decisions of record (binding)

**D1 — Storage: SQLite until (and unless) AI vector search forces otherwise.** SQLite is the store for
the entire practical roadmap. Relationship-chain traversal ("John's sister's husband") uses recursive
CTEs (SQLite supports them); full-text search uses FTS5. The *only* thing that would justify revisiting
Postgres is vector search for the AI layer — which is explicitly a nice-to-have, low-priority, and not
what this project is about (this is **not** an AI-first CRM). If AI features are ever built, vector
search goes in an **optional external sidecar** rather than forcing a primary-store migration. Schema
design should avoid gratuitous SQLite-isms so a future move isn't *impossible*, but must not contort
itself for a Postgres migration nobody has committed to.

**D2 — Fork posture: hard fork, selective upstream mirroring.** Once the feature set diverges
substantially, we lose (and accept losing) the ability to `git merge` upstream `fbuchner/meerkat-crm`
directly. Instead: watch what upstream fixes and **mirror the warranted ones by hand** (dependency
bumps, security fixes, doc improvements). Strict "must stay mergeable" is **dropped as a design
constraint** — it's a nice-to-have, not a rule. We are building our own product *based on* Meerkat, not
"Meerkat plus some bolted-on extras." Practical consequence: the meticulous additive-only discipline P0
and P1 used (to preserve mergeability) is **no longer mandatory** from here forward. It's still good
engineering hygiene where cheap, but we may now remove/replace existing fields, tables, and code paths
when a remodel genuinely warrants it — with our own migration + test discipline as the safety net, not
upstream compatibility.

**D3 — Relationship reconciliation: the graph is the internal source of truth.** There are three notions
of "relationship" in the codebase today; they unify as follows (same additive-then-retire pattern P1 used
for the flat contact fields):
- **First-class relationship graph** (new; the vision doc's model — directional UUID edges, inverse,
  metadata, source, confidence) → becomes the **internal source of truth**, envelope-side, its own
  entity/table.
- **`contactmodel.Card.RelatedTo []Relation`** (built in P0; the RFC vCard `RELATED` / JSContact
  `relatedTo` interchange field) → becomes the **export projection** of the graph: a graph edge between
  two real contacts whose type maps to a standard relation token projects out to `RelatedTo` on
  vCard/JSContact export. It is not a second source of truth.
- **`models.Relationship`** (Meerkat's legacy GORM table: flat `Name`/`Type` strings, optional
  `RelatedContactID` link) → **migrated into the graph and retired**, exactly as the flat contact fields
  are being retired across P1→P2.

**Graph invariant — every relationship endpoint is an entity (never a string).** Relationships are
always entity↔entity. This is why pets and minor children (who may have no contact information of their
own) are modeled as `Person`/`Contact` entities, not as attributes on another contact: it removes the
need for messy entity↔non-entity edges entirely. Corollary: **an entity does not require contact
information to exist** — a "thin" Contact (just a name, perhaps a birthday) is valid, existing purely to
be a relationship node. Migration consequence: legacy `models.Relationship` rows that carry only a `Name`
string with no `RelatedContactID` are migrated by **promoting that name into a new thin Contact** and
pointing the graph edge at it — no edge is ever left referencing a bare string.

**D4 — Prioritization (user-stated, binding on `92`'s sequencing).** In rough priority order:
1. Core data structures (relationship graph, households, generic events/timeline) — everything depends on these.
2. Relationships specifically (the graph + traversal).
3. APIs that enable future mobile clients (this is already the in-flight P2 / WP-71).
4. CardDAV / CalDAV import-and-export of contacts **and** timeline events — explicitly ranked as *far*
   more useful than AI or the "someday" integrations. (The source vision doc under-weights this; see 90.3.)
5. Immich integration (the user's named first "fancy" integration).
6. Everything else (other media/knowledge integrations, AI) is **deferred / someday** — nice-to-have, not roadmap-critical.

## 90.3 "What already exists" audit — this reweights the whole roadmap

The source vision doc reads as if most of this is greenfield. It is not. Verified against the codebase:

| Vision-doc concept | Already in Meerkat today | Roadmap implication |
|---|---|---|
| Contacts = CRM, CardDAV interchange | **CardDAV contact server exists and is two-way** (`carddav/backend.go`: address-book home set, list/get/put/delete address objects — clients read *and* write). vCard 3.0 today; adapters for 3.0+4.0 built in P0. | "CardDAV contact import/export" is ~80% done. WP-73 (P4) upgrades it to 4.0 through the P0 adapters. **Not greenfield — a wiring/upgrade job.** |
| Timeline fed by calendar events | **CalDAV/iCalendar event *import* exists** (`services/calendar_sync_service.go` via `go-ical`; `calendar_subscriptions` + `calendar_event_links` tables; pulls external events into `Activity` records). | Import-only, subscription-based. Net-new work is *export* (serve the CRM's own events out as CalDAV) and two-way sync — but the read/parse/link half exists. |
| "Interaction" entity | **`models.Activity`** — shared events with title/date/location + many-to-many contacts. | The Timeline/Interaction model is a **generalization/remodel of `Activity`**, not a new entity from zero. |
| Relationships as entities | **`models.Relationship`** (flat, see D3). | Remodel into the graph, don't build from scratch. |
| Circles / grouping | **`Contact.Circles []string`** exists (flat tags). | **Boundaries resolved (see 90.4a, 91.5):** three distinct concepts the flat field conflated — **Circle** (*social* grouping, how I know them; internal), **Tag** (*attribute* of the person, shared across strangers; projects to `Card.Keywords`), and **Household** (*co-residence*, first-class). Split all three on remodel. |
| Infra: webhooks | **Full webhook stack exists** (`models/webhook.go`, `services/webhook_service.go`, `controllers/webhook_controller.go` + tests, migration `000019`). | The vision doc's "Phase 3: webhooks" is largely **done**. Extend event coverage, don't rebuild. |
| Infra: ETags / optimistic concurrency | **Exists** (`Contact.ETag`, used by CardDAV `If-Match`). | Present for contacts; generalize to other entities as they gain APIs. |
| Infra: change feeds / cursor pagination / event history | Not present. | Genuinely net-new (Phase 3 infra). |

**Net effect:** the user's top practical priorities (CardDAV/CalDAV, timeline) are disproportionately
"finish and generalize existing v1s," which is why they're both high-value *and* lower-risk than the
source doc's framing implies — and why they correctly outrank the from-scratch, externally-dependent, or
AI-gated "someday" items.

## 90.4 Where each new feature lives (architecture placement)

- **Standardized `Card` (projects to vCard/JSContact):** nothing new here beyond what P0 built — the
  vision-doc features are almost all non-standard.
- **CRM envelope / new internal entities (source of truth, may *project* to standards where a mapping
  exists):** relationship graph (projects to `RelatedTo`), households (project to vCard `MEMBER`/`KIND:group`
  where applicable), life events, interactions/timeline, cadence policies, preferences, gifts,
  conversation agenda, external identities, external activities.

### 90.4a Circle vs. Household (resolved)

- **Circle** = a *social* grouping (friends/contacts thought of together). No co-residence or
  relationship implication. The existing flat `Contact.Circles []string` is its v1 — remodel from tags.
- **Household** = a *co-residence* grouping of people **and pets** who live together. A **first-class
  entity** (not "a circle + a shared address"), because it does two jobs a flat tag can't: (1) a
  **search/context boundary** — "resolve every member of this household" (e.g. to invite them all to an
  event); and (2) a **relationship-suggestion source** — the household's **type** governs suggestion
  strength (family unit → confidently suggest parent/child/spouse edges, and all humans as `owned_by`
  targets for the household pet; roommate → weak suggestions only, never hard rules). Address is an
  *attribute* of the household, not its identity (members who move keep the household).
- **Pets / non-human members force one decision:** a pet is modeled as a `Person`/`Contact` with a
  **CRM-envelope-side `kind` of `pet`/`animal`** (the standards' `kind` enum has no such value). It
  participates in the relationship graph and households like any member; its non-standard kind simply
  doesn't project into a standardized `Card`. One entity type, not a parallel "Animal" model.
- **Household-suggested relationships ride existing machinery:** each inferred edge gets
  `source: "household-inferred"` and a `confidence` scaled by household type, surfaced as a
  user-confirmable suggestion (never silently written) — the same "propose, user approves" pattern the
  vision doc applies to AI, applied here to *structural* inference. Detailed shapes (membership edges,
  the household-type → suggestion-strength table, the pet `kind`) belong in `91`.
- **External systems (referenced, never duplicated):** Immich photos, Dawarich/GeoPulse GPS, Jellyfin
  media, Audiobookshelf reading, Paperless docs, Nextcloud files. The CRM stores an `ExternalIdentity`
  link + optionally cached `ExternalActivity` provenance rows — never the source data itself.

## 90.5 Proposed phase sequence (scheduling judgment — details in `92`)

Continues the existing `WP-70…` numbering. This reflects D4's priorities and 90.3's "already exists"
findings. The in-flight P2/P3/P4 (WP-71–73) stay as the immediate next work; the vision-doc scope slots
in after and around them:

1. **Finish the in-flight foundation** — WP-71 (P2 API + mobile-ready summary/detail endpoints + OpenAPI),
   WP-72 (P3 frontend remodel), WP-73 (P4 CardDAV 4.0 upgrade). This delivers priority #3 (mobile APIs)
   and most of priority #4's *contact* half. **No reason to pause any of it** — it's the substrate the
   rest needs.
2. **Core relationship & event model** (priorities #1, #2) — the relationship graph (D3), households,
   and the generalized event/timeline model (remodel of `Activity`). Everything else depends on this, so
   it comes first among the *new* scope.
3. **Relationship-aware search** — traversal, chains, synonyms/aliases, inferred relations. Depends on #2.
4. **CalDAV event export + timeline two-way** (rest of priority #4) — serve the CRM's own timeline/events
   out as CalDAV; extend the existing import into two-way. Depends on the generalized timeline from #2.
5. **External-link + external-activity framework, then Immich (priority #5)** — the generic link/activity
   model first (so integrations don't each grow bespoke tables), then Immich Level 1–2 (local linking +
   enrichment) as the first concrete integration.
6. **Sync infrastructure** (change feeds, cursor pagination, event history/audit) — extend the existing
   webhooks/ETags. Slots here because mobile offline sync and multi-client editing need it, but it's not
   blocking the core model.
7. **Relationship cadence + relationship health + conversation agenda + preferences + gifts** — the
   "relationship maintenance" feature cluster. Depends on the timeline (#2) as its source of truth.
8. **Deferred / someday** — remaining integrations (Dawarich, Jellyfin, Audiobookshelf, Paperless,
   Nextcloud) and the entire AI/Ollama layer. Explicitly nice-to-have; not scheduled until the above land
   and a concrete need pulls them in.

## 90.6 Handoffs (docs still to write)

- **`91-envelope-data-model.md`** — the new internal entities in detail: `Person` (= existing `Contact`),
  `Relationship`-graph, `Household`, `Circle`, `Event`/`LifeEvent`, `Interaction`/`Timeline`,
  `Preference`, `CadencePolicy`, `ExternalIdentity`, `ExternalActivity` — each with its relationship to
  existing code, its table shape, and whether/how it projects to the standards.
- **`92-delivery-roadmap.md`** — 90.5's sequence expanded into WP-numbered work packages with a
  dependency graph and per-WP acceptance gates, in the style of `00-overview.md` §0.7 and
  `50-integration-and-rebrand.md`.
- **`93-integration-spec-template.md`** — the repeatable per-integration format the vision doc asks for
  (auth, data owned by the external app, data imported, data exported, sync direction, conflict handling,
  user permissions) + the Level 1–5 maturity tracker, instantiated first for Immich and CardDAV/CalDAV.
