# 93 — Integration spec template + maturity model

> A **repeatable format** to specify every external-system integration, so integrations stay consistent
> and all ride the generic `ExternalIdentity`/`ExternalActivity` substrate (`91.12`) instead of each
> growing bespoke tables. Fill one copy of §93.3 per integration. §93.5–93.6 instantiate it for the two
> the roadmap actually reaches first (Immich; and the CardDAV/CalDAV *protocol* distinction).

## 93.1 Two kinds of "integration" — don't confuse them

- **Application integrations** (Immich, Dawarich, Jellyfin, Audiobookshelf, Paperless, Nextcloud, Matrix,
  GitHub, Home Assistant…) — the CRM *links to* and *reads from* another app's identities/data. These use
  the `ExternalIdentity`/`ExternalActivity` substrate and the template below.
- **Protocol interchange** (CardDAV, CalDAV) — the CRM *is a server* (or a sync client) speaking a
  standard sync protocol; there is no "external app identity" to link. These are handled by their own
  protocol WPs (`WP-73` CardDAV, `WP-87/88` CalDAV), **not** the `ExternalIdentity` substrate. The
  template applies only loosely (see §93.6). Keeping this distinction explicit prevents someone forcing
  CardDAV into the wrong model.

## 93.2 Maturity model (Level 1–5)

Each integration declares its **current** and **target** level. Higher levels are strictly optional and
often gated on upstream cooperation.

| Level | Name | What it does | Substrate |
|---|---|---|---|
| **L1** | Linking | Store an external identifier; deep-link to the external system. | writes `ExternalIdentity` |
| **L2** | Enrichment | Pull useful metadata *in* (read-only): counts, latest activity, thumbnails. | reads external; caches into `ExternalActivity` / `ExternalIdentity.metadata` |
| **L3** | Synchronization | Maintain identity mappings and/or data in both directions. | `ExternalActivity` + sync state; ETags/optimistic concurrency (`92` P9) |
| **L4** | Intelligence | Cross-reference *across* systems (calendar+GPS+photos → one event; suggest relationships/life events). | reads multiple substrates; emits **suggestions** |
| **L5** | Autonomous assistance | AI proposes/performs updates. | **all changes require approval** (propose-then-approve, same pattern as household inference `91.4`) |

**Binding rule for L4–L5:** never silently modify data — surface suggestions the user approves/rejects
(`90` D-priorities put AI in "deferred/someday"; L4–L5 are not near-term).

## 93.3 The template (fill one per application integration)

```
Integration:            <name>
External system owns:    <what it is authoritative for — never duplicated into the CRM>
Current level / target:  L<n> → L<m>
Upstream dependency:     <none | requires a change/feature in the external project — describe>

Authentication:
  - <API key / OAuth / token>, per-user, user-supplied.
  - Stored encrypted (reuse services/credential_crypto.go).
  - PROHIBITED: the CRM never enters the user's credentials into the external system on their
    behalf; the user supplies an API key/token themselves (see the safety rules on credential entry).

Data imported (external → CRM):
  - Identity link → ExternalIdentity{system, external_id, url, metadata}
  - Enrichment/activity → ExternalActivity{source_system, external_id, timestamp, type, entity_ids, provenance}
  - <list concrete fields>

Data exported (CRM → external):
  - <none | list — most integrations are read-only in early levels>

Sync direction:          <one-way in | one-way out | bidirectional>
Conflict handling:       <n/a for read-only | which side wins | ETag/If-Match; see 92 P9>
User permissions:        <who may link/unlink; scope; per-user isolation>
Substrate config:        <ExternalIdentity.metadata keys used; any integration-specific config>
```

## 93.4 Substrate mapping (reinforces `91.12`)

- **No per-integration tables.** L1 writes an `ExternalIdentity` row (`system` names the integration);
  L2+ writes `ExternalActivity` rows and/or caches metadata in `ExternalIdentity.metadata` (JSON).
- Integration-specific *config* (a base URL, sync toggles) lives in `ExternalIdentity.metadata` unless it
  is genuinely per-user-global, in which case a small typed config akin to `calendar_subscriptions`
  (which already exists) is acceptable — but justify it; the default is the generic substrate.

## 93.5 Instantiated: Immich (priority #5; `92` WP-89→91)

```
Integration:            Immich
External system owns:    photos, face-recognized "Person" clusters, albums
Current level / target:  (none) → L2  (L3 deferred — needs upstream, see below)
Upstream dependency:     L1–L2 none. L3 (Immich showing a link back to the CRM contact) needs a generic
                         "External Links" capability in Immich (external system/id/url) — request or
                         contribute upstream rather than a CRM-specific Immich feature (vision doc flags this).

Authentication:          Immich API key, per-user, user-supplied, stored encrypted.
Data imported:
  - Immich Person ID          → ExternalIdentity{system:"immich", external_id, url:<person page>}
  - photo count, latest appearance, thumbnail, associated albums (L2)
                              → ExternalActivity{type:"photo-appearance", timestamp, entity_ids} + metadata cache
Data exported:           none (L1–L2). (Avatar-from-Immich selection is a possible later nicety, not L1–L2.)
Sync direction:          one-way in
Conflict handling:       n/a (read-only enrichment)
User permissions:        user links their own contacts to their own Immich persons; per-user isolation
Substrate config:        Immich base URL + api key ref in ExternalIdentity.metadata
```
WP mapping: **WP-90** = L1 (linking, pure CRM-side, no upstream dep); **WP-91** = L2 (enrichment →
timeline). "Dream state" face-match/avatar suggestions are L4+ and out of near-term scope.

## 93.6 CardDAV / CalDAV — protocol interchange, not an ExternalIdentity integration

These do **not** use §93.3. The CRM implements them as **protocols it serves** (and, for calendar, also
consumes):
- **CardDAV** — the CRM already *serves* contacts (`carddav/backend.go`, two-way). `WP-73` upgrades it to
  vCard 4.0 through the P0 adapters + version negotiation. No `ExternalIdentity` involved; the contact's
  own `VCardUID`/`ETag` are the sync primitives.
- **CalDAV / iCalendar** — the CRM already *imports* external calendar events into `Activity`
  (`calendar_sync_service.go`, `calendar_subscriptions`). `WP-87` adds *serving* the CRM's own
  timeline out as CalDAV; `WP-88` makes it two-way. Again protocol-level, not substrate-level.

The one place CardDAV *does* touch the substrate idea: a contact synced *from* an external CardDAV source
could carry that source as an `ExternalIdentity{system:"carddav", external_id:<uid>, url}` for
provenance — optional, and distinct from the CRM's own CardDAV-server role above.

## 93.7 Using this doc

When an integration comes into scope (per `92`'s deferred list), copy §93.3, fill it, drop it beside this
file (e.g. `93-immich.md` once Immich is elaborated beyond §93.5), and implement against the maturity
level declared. The substrate (`WP-89`) must land before any application integration past L1.
