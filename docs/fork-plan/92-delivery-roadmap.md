# 92 — Delivery roadmap (sequenced work packages)

> The vision scope (`90`/`91`) expressed as WP-numbered phases with a dependency graph, continuing the
> `50-integration-and-rebrand.md` numbering. Ordering follows `90.5` (which encodes the user's stated
> priorities, `90` D4). Each WP's acceptance gate is the same discipline used throughout: `go build ./...`
> + `go vet` + `go test ./...` green (backend), plus the frontend/e2e gates where UI is touched.
>
> **This roadmap is deliberately coarse past P8.** Near-term phases are specified tightly; later phases
> are placeholders whose detail is written when they come into view, because the model they build on will
> have taught us things by then.

## 92.0 Where the existing plan ends and this begins

Already specified in `50-integration-and-rebrand.md` (do these first — they are `90.5` step 1):
- **WP-70 · P1** — persistence swap + migration. **DONE** (committed).
- **WP-71 · P2** — API + mobile-ready summary/detail endpoints + OpenAPI. *Next up.* (Delivers priority #3.)
- **WP-72 · P3** — frontend remodel.
- **WP-73 · P4** — CardDAV 4.0 upgrade through the adapters. (Delivers the *contact* half of priority #4.)
- **WP-74** — rebrand. **Re-slotted:** originally "do last"; with the hard-fork decision (`90` D2) it can
  happen at any deliberate branding moment, but remains non-blocking and is not scheduled here. Left where
  it is; timing is a product call.

This doc covers `90.5` steps 2–8 as new phases **P5–P10** (+ deferred), starting at **WP-80**.

## 92.1 P5 · Core relationship & event model  (WP-80..84) — `90.5` step 2, priorities #1/#2

The foundation everything else depends on. Build the `91` entities. **Hard gate: nothing in P6+ starts
until P5 is green**, because search, timeline, cadence, and integrations all read these.

| WP | Scope | Depends on | Notes |
|---|---|---|---|
| **WP-80** | Relationship graph entity (`91.2`) + type registry (canonical tokens, inverses, synonyms, standard-projection) + `Card.RelatedTo` projection wiring. | P1, P0 adapters | Store one edge, derive inverse. Only `confirmed` edges project to standards. |
| **WP-81** | Migrate legacy `models.Relationship` → graph, incl. **promoting name-only endpoints to thin Contacts** (`90` D3 invariant). Dry-run + round-trip verified, same discipline as WP-70's backfill. | WP-80 | Irreversible data migration — gate behind dry-run. |
| **WP-82** | `Contact.kind` extension (individual/pet/animal) + thin-entity relaxation (nothing but name required). | P1 | Small; unblocks pets/children as entities. |
| **WP-83** | Household entity + membership + household-type→suggestion engine (`91.3`/`91.4`). Suggestions written as `status:suggested` edges only. | WP-80, WP-82 | Suggestion review surface is frontend (P-later); backend emits proposals. |
| **WP-84** | Generalize `Activity` → Interaction (`91.7`): add UUID, `type`, external-ref; keep existing rows working. Circle **and Tag** remodel (`91.5`) from flat `Contact.Circles` (Tag also back-populates `Card.Keywords`). LifeEvent entity (`91.6`). | P1 | Additive-friendly where cheap; hard-fork lets us remodel `Activity` directly if needed. |
| **WP-84b** | User-defined custom fields (`94`): `FieldDefinition` + typed `FieldValue` + validation (reuse existing validators) + standards-projection via the P0 `JCardProp`/passthrough path. Migrate the untyped v1 (`User.CustomFieldNames` + `Contact.CustomFields`). | P1 | Backend independent of the graph; **full UX depends on P3's data-driven field registry** (WP-72) — custom fields make that registry partly data-driven. Sensitivity (`91.13`) applies. |

**P5 acceptance:** the `91.13` entity graph exists, legacy relationship + circle data migrated
(dry-run-verified), all green, and a relationship edge between two real contacts round-trips to
`Card.RelatedTo` and back.

## 92.2 P6 · Relationship-aware search  (WP-85..86) — `90.5` step 3

| WP | Scope | Depends on |
|---|---|---|
| **WP-85** | Graph traversal + multi-hop chains via recursive CTEs (SQLite): "Teddy's owner", "John's sister's husband". Inferred relations (grandparent from two parent edges) computed, not stored. | WP-80 |
| **WP-86** | Search synonyms/aliases from the type registry (`mom`/`mother`→`parent_of`), household-scoped queries ("everyone in the Smith household"), FTS5 full-text over contacts/notes/interactions. | WP-85, WP-83 |

## 92.3 P7 · CalDAV export + two-way timeline  (WP-87..88) — `90.5` step 4, rest of priority #4

The existing CalDAV *import* (`services/calendar_sync_service.go` → Activities) is the read half; this
adds the write/serve half.

| WP | Scope | Depends on |
|---|---|---|
| **WP-87** | Serve the CRM's own Interactions/LifeEvents *out* as CalDAV/iCalendar (a CalDAV collection clients can subscribe to), mirroring the existing CardDAV server pattern (`carddav/backend.go`). | WP-84 |
| **WP-88** | Two-way calendar sync: extend import→Activity into reconciled two-way (respect ETags/If-Match, the existing conflict primitives). Contacts CardDAV already two-way (WP-73). | WP-87, WP-73 |

## 92.4 P8 · External-link framework + Immich  (WP-89..91) — `90.5` step 5, priority #5

Generic substrate **first**, then the first concrete integration — so no integration grows bespoke tables.

| WP | Scope | Depends on |
|---|---|---|
| **WP-89** | ExternalIdentity + ExternalActivity entities (`91.12`) + generic link/enrichment API. | WP-84 |
| **WP-90** | Immich Level 1 (linking): store Immich Person ID, deep-link to Immich, display photo count / latest appearance. Pure CRM-side, no upstream dep. | WP-89 |
| **WP-91** | Immich Level 2 (enrichment): pull latest photo / appearances into ExternalActivity → timeline. Level 3 (bidirectional) is **deferred** — needs an Immich "external links" capability upstream (`93` / vision doc flags this). | WP-90 |

## 92.5 P9 · Sync infrastructure  (WP-92..93) — `90.5` step 6

Extends existing webhooks/ETags; enables mobile offline + multi-client. Slots here (not earlier) because
the core model must exist to have something worth change-feeding, but it gates real mobile-app sync.

| WP | Scope | Depends on |
|---|---|---|
| **WP-92** | Change feeds + cursor pagination over the entity APIs (not offset — for large histories/timelines). | WP-71, WP-84 |
| **WP-93** | Event history / audit trail (immutable create/update/delete/merge/restore log) → undo, sync, debugging. Extend webhook event coverage to the new entities. | WP-92 |

## 92.6 P10 · Relationship-maintenance cluster  (WP-94..96) — `90.5` step 7

The "personal relationship OS" payoff features. All read the timeline (`91.8`) as source of truth.

| WP | Scope | Depends on |
|---|---|---|
| **WP-94** | CadencePolicy (`91.10`) + derived relationship health (next-due/overdue computed from timeline). Overdue → task to external manager (Vikunja) via webhook; **recording a qualifying interaction, not task completion, resets cadence.** | WP-84 (timeline) |
| **WP-95** | Preferences (`91.9`, migrate `FoodPreference`; project `hobby` to `Card.PersonalInfo`) + Gift tracking (`91.11`). | WP-84 |
| **WP-96** | ConversationAgenda (`91.11`) — contextual memory surfaced on contact view, not date-scheduled. | WP-84 |

## 92.7 Deferred / someday — `90.5` step 8

Explicitly not scheduled; nice-to-have, pulled in only when a concrete need arises and the above have
landed:
- **Other integrations** (Dawarich/GeoPulse, Jellyfin, Audiobookshelf, Paperless-ngx, Nextcloud) — each
  a `93`-template instance atop the WP-89 substrate; mostly Level 1–2, API-based, no upstream deps.
- **AI / Ollama layer** (summarization, entity/relationship/life-event extraction, timeline synthesis,
  memory-curator suggestions). Gated on everything structured existing first, and on the propose-then-
  approve workflow (already the pattern used by household inference in WP-83). Would revisit D1's storage
  decision *only* if vector search proves necessary, and then via an external sidecar, not a primary-store
  migration. **This is not an AI-first project** (`90` D1).

## 92.8 Dependency graph (condensed)

```
P1(done) ─┬─ P2/WP-71 ─── P3/WP-72
          │       └────────────────────────┐
          └─ P5/WP-80 ─┬─ WP-81 (migrate)   │
                        ├─ WP-82 (kind) ─ WP-83 (household)
                        └─ WP-84 (interaction/circle/lifeevent) ─┬─ P6 (search: WP-85→86)
                                                                 ├─ P7 (caldav: WP-87→88) ── needs WP-73
                                                                 ├─ P8 (extlink+immich: WP-89→90→91)
                                                                 ├─ P9 (sync: WP-92→93) ── needs WP-71
                                                                 └─ P10 (cadence/pref/gift/agenda: WP-94..96)
                                                                       └─ deferred (integrations, AI)
```

## 92.9 Cross-cutting notes

- **Hard-fork posture (D2)** means WPs from P5 on may remove/replace legacy code (e.g. retire
  `models.Relationship`) rather than only adding — with our own migration + dry-run + round-trip
  discipline as the safety net, not upstream mergeability.
- **Mobile clients** (D4 #3): every new entity in P5–P10 should get its summary/detail/OpenAPI treatment
  in the same style WP-71 establishes for contacts, so a future Swift/Kotlin/Dart client (and the
  deferred local-model pilot, `80`) targets one coherent, spec'd API — not a patchwork.
- **The local-model code-gen pilot (`80`)** re-enters when mobile client work begins (its own deferral),
  independent of this backend roadmap.
