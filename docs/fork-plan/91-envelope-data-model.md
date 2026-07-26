# 91 — Envelope data model (the new internal entities)

> Design-level shapes for the CRM-envelope-side entities the vision doc adds, per `90`'s decisions.
> Sketches (fields + relationships + standards-projection), not final DDL — the migration/GORM details
> land in each entity's implementation WP (see `92`). Everything here is **source-of-truth internal
> data**; it projects to vCard/JSContact only where a standard mapping exists, never the other way round.
>
> **Storage:** SQLite (decision D1). IDs are UUID strings (`text`), not autoincrement ints, so entities
> are stable across export/import and can be referenced by external systems — except where noted that an
> entity extends an existing autoincrement table (`Contact`, `Activity`), which keep their int PKs plus a
> stable UUID column.

## 91.1 Person (= the existing `Contact`, extended — not a new entity)

The vision doc's "Person" is Meerkat's `Contact`. No parallel entity. Extended, per `90`'s invariants:

- **`kind`** (CRM-envelope-side; enum `individual | pet | animal | ...`). The standards' `kind`
  (individual/group/org/location/application/device) has no pet/animal value, so this lives envelope-side
  and does not project into a standardized `Card` (a `pet` exports as `individual`, or is omitted from
  contact exports entirely per user preference — a WP decision, not a model one).
- **Thin entities are valid.** `Contact.Firstname` staying `required,min=1` is fine (a name is the
  minimum), but nothing else may be required. A pet or a minor child is a Contact with a name and little
  else, existing to be a relationship node (graph invariant, `90` D3).
- Already carries the neutral `Card`/`CRM`/`Passthrough` columns (P1). The new entities below either
  reference a Contact by UUID or hang off it.

## 91.2 Relationship (the graph edge — new first-class entity)

Replaces the legacy `models.Relationship` (D3). The edge *is* the entity.

| Field | Notes |
|---|---|
| `id` | UUID |
| `source_id` / `target_id` | UUIDs of two entities (Contacts). **Never a string** (graph invariant). |
| `type` | canonical relation token, e.g. `parent_of`, `spouse_of`, `owned_by`, `manager_of`, `mentor_of`, `sibling_of`, `roommate_of`. |
| `inverse_type` | the reciprocal token (`parent_of` ⇄ `child_of`); may be derived from a type registry rather than stored. |
| `directional` | bool. Symmetric relations (`spouse_of`, `sibling_of`, `roommate_of`) vs. directional (`parent_of`, `owned_by`). |
| `metadata` | JSON (free-form: since/until dates, notes). |
| `source` | provenance enum: `user-confirmed`, `household-inferred`, `imported` (from vCard `RELATED` / legacy table), `ai-suggested`. |
| `confidence` | 0–1. `user-confirmed` = 1.0; `household-inferred`/`ai-suggested` < 1.0 (see 91.4). |
| `status` | `confirmed` \| `suggested`. **Only `confirmed` edges are authoritative;** `suggested` edges await user approval and are never used as hard facts (matches the vision doc's propose-then-approve philosophy). |

- **Inverse handling:** store one edge, derive the inverse via the type registry (don't store both
  directions — avoids the "keep two rows in sync" bug). Traversal (`92` P6) walks either direction.
- **Inferred/multi-hop relations are NOT stored** (`grandparent_of` from two `parent_of` edges) — they're
  computed at query time (recursive CTE). Only atomic, asserted edges are persisted.
- **Type registry** (a small data table or Go map): canonical token → { inverse, symmetric?, synonyms[],
  standard-projection }. Synonyms drive search (`mom`/`mother`/`dad` → `parent_of`); the
  standard-projection field says how/whether the type maps to a vCard `RELATED` `TYPE` token.

### Edge design for real-world complexity (type = role, metadata = nature)

The edge vocabulary stays **small and searchable**; the messy variety of real relationships lives in
**flexible metadata**, not in a combinatorial explosion of edge types. This is what lets the graph model
open/poly/mono relationships, metamours, paramours, co-parenting, chosen family, adoption, and step-
relations without special-casing any of them:

- **`type` = the social role** — a small, stable set (`partner_of`, `parent_of`, `sibling_of`,
  `co_parent_of`, `friend_of`, `owned_by`, `mentor_of`, …).
- **`metadata` = the nature/qualifiers** — `{kind: biological|adoptive|step|chosen}`, legal status,
  since/until, custody arrangement, poly descriptors (nesting/anchor/etc.). So: adoption = `parent_of` +
  `{legal: adoptive}`; chosen family = `parent_of`/`sibling_of` + `{kind: chosen}`; step-parent =
  `parent_of` + `{kind: step}`. **No new edge types** for any of these.
- **Multi-edge per node pair** — two entities may hold several concurrent edges of different types. This
  is the thing a flat/vCard model *cannot* express: co-parents who aren't partners = `co_parent_of` only;
  poly partners who also co-parent = `partner_of` **and** `co_parent_of`; divorced co-parents keep
  `co_parent_of` after `partner_of` ends. The graph gets this for free.
- **Polyamory / open relationships** = simply *multiple `partner_of` edges from one node* — there is no
  "primary partner" field to break. **Metamour** (a partner's other partner) is a 2-hop *derivation* by
  default (`92` P6), but may be *explicitly asserted* with its own metadata when the user wants to
  annotate it (some metamours are close friends, some merely aware of each other). **Derived and asserted
  coexist** across the whole model — derivation is the default, an explicit edge overrides/enriches.
- **Reified edges** — a Relationship has its own UUID and is *addressable*: it can anchor a CadencePolicy
  (91.10), a timeline ("your friendship with X began in college…"), and its own external refs. It is
  simultaneously an *edge* (two endpoints) and an *entity* (addressable, metadata-bearing) — this is what
  "relationships as entities themselves" means, and 91.2's shape already provides it.

### Household is a node, not a relationship edge (why they stay separate)

A relationship edge is **dyadic** (two endpoints); a household is **n-ary** (a group of N members incl.
pets). A household therefore cannot be a relationship edge — it is a **node** members reach via
`member_of` edges (91.3), exactly as vCard models a group (`KIND:group` + `MEMBER`). It *is* in the same
graph, just as a different kind of node. The decisive reason to keep the two separate is semantic:
**relationships outlive households.** People move out and households dissolve, but a `co_parent_of` /
`spouse_of` edge persists — conflating them would destroy relationship history whenever living
arrangements change. Household = the current physical/contextual grouping (mutable); relationships = the
durable bonds. The household's role is to *suggest* dyadic relationships (91.4), never to *be* them.

**Standards projection:** a `confirmed` edge between two real Contacts whose `type` has a standard
projection emits as `Card.RelatedTo` (vCard `RELATED` / JSContact `relatedTo`) on export. This is
**deliberately lossy** (loss-on-export-never-on-import): the metadata qualifiers, the multi-edge richness,
and any type with no standard token (`co_parent_of`, `metamour`, …) do **not** survive to vCard `RELATED`
— the full truth stays in the graph, and vCard receives only the mappable subset. Edges to thin pet/child
entities, or of non-projecting types, simply stay internal.

## 91.3 Household (new first-class entity)

Co-residence grouping of people **and pets** (`90.4a`). First-class, because it carries a type and has
members — not a flat tag.

| Field | Notes |
|---|---|
| `id` | UUID |
| `name` | e.g. "Smith Family", "Apartment 4B". |
| `type` | enum: `family_unit`, `roommates`, `other`. Drives suggestion strength (91.4). |
| `address` | an *attribute*, optional (members who move keep the household). Reuses the neutral `Address` shape. |
| `members` | membership edges (below). |

**Membership edge** (`household_members` join): `household_id`, `entity_id` (Contact UUID), `role`
(optional: `head`, `child`, `pet`, `roommate`…), `since`/`until`.

**Two jobs it must serve:**
1. **Search/context boundary** — "resolve all members of household X" (e.g. invite the whole household
   to an event). A simple membership query.
2. **Relationship-suggestion source** — see 91.4.

**Standards projection:** a `family_unit`/group-like household *may* project to vCard `KIND:group` +
`MEMBER` where members are exportable contacts; pet members and roommate households generally don't
project cleanly and stay internal. Projection is best-effort, never lossy-to-internal.

## 91.4 Household → relationship suggestions (mechanism, not a new entity)

When a household's membership changes, the system *proposes* relationship edges (never writes them
silently). Each proposal is a `Relationship` row with `status: suggested`, `source: household-inferred`,
and a `confidence` set by household `type`:

| Household type | Suggested edges | Confidence | Notes |
|---|---|---|---|
| `family_unit` | adult↔adult `spouse_of`; adult→child `parent_of`; every human → household pet `owned_by` | high (e.g. 0.8) | Strong but still user-confirmable — "family unit" is an assertion the user made. |
| `roommates` | member↔member `roommate_of` only | low (e.g. 0.4) | Ambiguous by nature; never suggests parent/owner/spouse. |
| `other` | none | — | No structural inference. |

The user confirms/rejects in a review surface (same UX as AI suggestions); confirming flips `status` to
`confirmed` and `confidence` to 1.0. Rejecting deletes the suggestion (and optionally suppresses
re-suggestion). **No suggested edge is ever treated as a fact.**

## 91.5 Circle vs. Tag (two orthogonal groupings — remodel of flat `Contact.Circles`)

These are **different axes**, and the existing flat `Contact.Circles []string` conflates them. Split on
remodel:

**Circle** — *how I know them / that they likely know each other.* A social grouping with an implied
shared context of acquaintance ("Cam Girl friends", "Studio Porn friends", "College friends"). Says
something about the **connection**, not about the person as an individual.
- fields: `id` (UUID), `name`, `members` (`circle_members` join → entity UUIDs).
- No relationship inference (unlike Household). **No standards projection** — a circle is about *my*
  relationship-context, not an attribute of the contact, so it has no vCard/JSContact home; stays internal.

**Tag** — *an attribute a set of people share, even if they don't know each other.* A property of the
**person** ("SWer", "poly", "vegan", "photographer"). Tagged contacts may be strangers to each other; the
point is attribute-query: "everyone tagged `poly`", "everyone tagged `SWer`".
- fields: `id` (UUID), `name`, + taggings (contact ↔ tag).
- **Standards projection: Tag → `Card.Keywords` (vCard `CATEGORIES` / JSContact `keywords`)** — CATEGORIES
  *is* "categories/attributes of this contact", so tags have a clean standard home. (This corrects an
  earlier draft note that dismissed CATEGORIES for *circles* — it's wrong for circles, right for tags.)
  `Card.Keywords` already exists from P0; Tag promotes it into a first-class, queryable concept.

**Why the split is load-bearing (worked examples):**
- "Everyone I know from sex work" as one flat circle collapses real structure. Model it as *circles by
  connection* ("Cam Girl friends", "Studio Porn friends") **plus** a shared *tag* `SWer` identifying the
  common attribute across those circles — so both the attribute query ("who is a SWer") and the
  social-context query ("who's in my Cam Girl friends circle") are answerable.
- **`poly` is a tag, not a graph derivation.** It is an identity / preferred relationship structure — an
  attribute of the person *independent of* their current `partner_of` edges. "Friends who identify as
  poly" is an attribute query, and deriving it from the relationship graph would be *wrong*: someone
  identifies as poly whether or not they are presently partnered with multiple people.

Migration: each distinct string in existing `Contact.Circles` is triaged into a Circle **or** a Tag (a
light user-assisted step, since the flat field mixed both); tags additionally back-populate `Card.Keywords`.

## 91.6 Event / LifeEvent (new)

Permanent facts about an entity's life ("what happened in *their* life") — distinct from Interaction
("what happened between *us*", 91.7).

| Field | Notes |
|---|---|
| `id` | UUID |
| `entity_id` | subject Contact (may have secondary participants via a join, e.g. a marriage). |
| `type` | `married`, `graduated`, `job_change`, `had_child`, `adopted_pet`, `retired`, `moved`, … |
| `date` | reuse `contactmodel.PartialDate` (year/month/day optional) — life events are often known only to a year. |
| `description` | free text. |
| `source` | `user`, `imported`, `ai-suggested`. |
| `related_entity_ids` | optional (the new child, the pet adopted, the org joined). |

**Standards projection:** mostly none. `married` could inform an `anniversary` (kind=wedding) on the
Card; `had_child`/`adopted_pet` may create thin entities + `parent_of`/`owned_by` edges (as suggestions).
Life events **feed the timeline** (91.8).

## 91.7 Interaction (remodel/generalization of the existing `models.Activity`)

"What happened between us." `Activity` (title/date/location + many-to-many contacts) is its v1 — extend
it rather than build anew.

| Field | Notes |
|---|---|
| `id` | UUID (Activity keeps its int PK + gains a UUID). |
| `participants` | entity UUIDs (many-to-many, already exists as `activity_contacts`). |
| `type` | `call`, `video_call`, `visit`, `meal`, `gift`, `photo`, `message`, `shared_activity`, … |
| `date` | timestamp (exists). |
| `qualifying` | derived, not stored: whether this interaction type counts toward a cadence policy (91.10). Social-media-like/passive events are non-qualifying. |
| `source` | `user`, `calendar-import` (existing CalDAV sync), `external` (91.12), … |
| `external_ref` | optional link to an `ExternalActivity` (91.12) or `calendar_event_links` row (exists). |

## 91.8 Timeline (a query/view, not a table)

The canonical relationship history is the **union** of Interactions (91.7) + Life Events (91.6) +
External Activities (91.12) + Notes + fired Reminders, filtered by entity/relationship and ordered by
date. It is a *read model* (a query, possibly a materialized view later for performance), **not** a new
table that duplicates the above. Everything that happens "contributes to the timeline" by being one of
those underlying rows tagged with participant entities and a date. Cadence (91.10) and any future AI
context both read the timeline rather than maintaining their own history.

## 91.9 Preference (new — structured personal facts)

Important info that currently only lives in notes or the single `Contact.FoodPreference` free-text field.

| Field | Notes |
|---|---|
| `id` | UUID |
| `entity_id` | subject. |
| `category` | `food`, `drink`, `clothing_size`, `hobby`, `gift`, `dislike`, `media`, … |
| `key` / `value` | e.g. `favorite_coffee` = `Latte`; structured value (string, or JSON for sized/typed values). |
| `source` | `conversation_note`, `user`, `ai-suggested`, `external` (e.g. Jellyfin favorite). |
| `confidence` / `last_confirmed` | preferences go stale; track when last confirmed. |

**Relationship to existing:** the free-text `Contact.FoodPreference` migrates into a `food` preference;
`Card.PersonalInfo` (hobbies/interests/expertise, from P0) is the *standards-facing* subset — a Preference
of category `hobby` may project to `Card.PersonalInfo`, but most preference categories have no standard
home and stay internal.

## 91.10 CadencePolicy (new — relationship-maintenance rule)

"Stay in touch every N days." Relationship-driven, distinct from date-driven Reminders and action-driven
Tasks (the vision doc's key three-way distinction).

| Field | Notes |
|---|---|
| `id` | UUID |
| `entity_id` (or `relationship_id`) | who this cadence is about. |
| `target_interval_days` | e.g. 30. |
| `qualifying_types` | which Interaction types reset the cadence (call/visit/meal…); everything else is ignored. |

**Derived, never stored state:** relationship health (`next_due`, `overdue_by`) is *computed* from the
timeline — find the most recent *qualifying* interaction, add the interval. **Completing a generated task
does not reset cadence; recording a qualifying interaction does** (the vision doc is explicit: the
meaningful interaction is the source of truth). Overdue cadence may emit a task to an external task
manager (Vikunja) via the existing webhook/external mechanism, but the CRM owns the cadence *state*.

## 91.11 Gift + ConversationAgenda (new — small satellite entities)

- **Gift:** `id`, `recipient_id`, `status` (`idea`|`purchased`|`given`), `occasion`/`event_id`,
  `description`, `date`, optional `preference_id` link. Answers "what did I give them last year?".
- **ConversationAgenda:** `id`, `entity_id`, `item` (text), `created`, `status` (`open`|`discussed`),
  `source`. **Not a reminder** — it's contextual memory ("what to remember when I next talk to them"),
  surfaced when viewing/contacting the entity, not on a date schedule.

Both feed the timeline when acted on (a given gift, a discussed agenda item become interactions/notes).

## 91.12 ExternalIdentity + ExternalActivity (new — the generic integration substrate)

**Do not create per-integration tables.** Two generic entities serve all integrations (Immich, Matrix,
GitHub, Home Assistant, Dawarich, Jellyfin, …):

- **ExternalIdentity** — an identity link: `id`, `entity_id`, `system` (`immich`, `carddav`, `github`…),
  `external_id`, `url`, `metadata` (JSON), `sync_status`. This is what makes the CRM the "identity hub"
  (`90`): one Contact ↔ many external system identities.
- **ExternalActivity** — an event contributed by an external system: `id`, `entity_ids[]`,
  `source_system`, `external_id`, `timestamp`, `type` (photo-appearance, gps-visit, media-watched…),
  `provenance`, `sync_state`. Feeds the timeline (91.8) without the CRM owning the source data (the photo
  stays in Immich; the CRM stores a reference + a timeline entry).

Integrations progress through the Level 1–5 maturity model (`93`); Level 1 (linking) writes
ExternalIdentity, Level 2 (enrichment) reads external metadata, Level 3+ adds ExternalActivity + sync.

## 91.13 Sensitivity / contextual visibility (cross-cutting concern)

Some data must exist in the CRM but **not surface in every context** — an `affair` `partner_of` edge, a
`SWer` tag, certain life events. This is not per-entity-type; it's a **cross-cutting marker** that can sit
on relationships, tags, life events (and, if needed, preferences/interactions):

- **`sensitivity`** field (e.g. `normal | private | secret`) on the item.
- **Default-exclude rule:** items above `normal` are **omitted by default** from (a) standards **exports**
  (an affair partner does not emit as vCard `RELATED`; a secret tag does not emit as `CATEGORIES`), (b)
  **external-system sync** and any **AI** surface, and (c) **shared / multi-user views**. Including or
  revealing a sensitive item is an **explicit, per-context action**, never the default.
- Sensitivity travels with the affair-partner example cleanly under the type=role/metadata=nature rule
  (91.2): `partner_of` + `{status: affair, sensitivity: secret}` — no special edge type, and the
  sensitivity marker is what governs disclosure.

The exact context taxonomy and per-view reveal UX are a later (WP-level) detail; the binding principle
now is *sensitive-by-marker, excluded-by-default, revealed-only-explicitly*.

## 91.14 Entity/relationship summary

```
Contact (Person; existing, extended: kind, thin-entity-ok)
  ├─ 1:N ExternalIdentity        (identity hub)
  ├─ M:N Circle        (social grouping — how I know them; internal only)
  ├─ M:N Tag           (shared attribute — about the person; projects to Card.Keywords)
  ├─ M:N Household     (co-residence; role per membership)
  ├─ 1:N Preference / Gift / ConversationAgenda / LifeEvent / CadencePolicy
  └─ participates in Interaction (M:N; remodel of Activity) + ExternalActivity

Relationship (graph edge; entity↔entity; type=role + metadata=nature; multi-edge; source/confidence/status)
Timeline (read-model union of Interaction + LifeEvent + ExternalActivity + Notes + Reminders)
Sensitivity (cross-cutting marker on relationships/tags/life-events; excluded-by-default from export/external/shared)
```

Reminders, Notes, Activities, Webhooks, ETags already exist and are reused/extended, not rebuilt.
