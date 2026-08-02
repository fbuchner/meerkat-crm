# Tickets

One file per ticket, in execution order. Each is meant to be self-contained enough to implement from
without reading the whole backlog — but **read `/CLAUDE.md` first**, since it carries the repo-wide
conventions and recurring traps that every ticket assumes.

Ordering comes from `../95-backlog-and-priorities.md`'s board, by these rules in precedence order:
dependency → pre/post-alpha risk → usefulness rating → size.

**Ratings** (`R`): 5 practically necessary · 4 strong, frequent use · 3 nice to have · 2 rarely used ·
1 re-evaluate whether a CRM should do this.

## Before alpha

| # | Ticket | R | Size | Status |
|---|---|---|---|---|
| [01](01-N1-contact-merge.md) | N1 · Contact merge / dedupe | 5 | M | **DONE** |
| [02](02-N4-notes-capture-inbox.md) | N4 · Notes: dead-end journal → capture inbox | 4 | S–M | **DONE** |
| [03](03-T5-lifeevent-frontend.md) | T5 · LifeEvent frontend + timeline | 4 | M | **DONE** |
| [04](04-T5b-lifeevent-reminders.md) | T5b · LifeEvent → reminder wiring | 4 | S | **DONE** |
| [05](05-T2-circle-tag-triage.md) | T2 · Circle/Tag triage migration | 4 | M | **DONE** |
| [06](06-T3-circle-tag-backend.md) | T3 · Circle/Tag backend rewiring | 4 | S–M | **DONE** |
| [07](07-T4-circle-tag-frontend.md) | T4 · Circle/Tag frontend rewiring | 4 | L | **DONE** |
| [08](08-T25-known-gaps.md) | T25 · Known small functional gaps | 3 | S | **DONE** |
| [08b](08b-T26-delete-semantics.md) | T26 · Delete semantics — purge job + constraint fixes | 3 | M | **DONE** |
| [09](09-T1-households.md) | T1 · Household CRUD + suggestion trigger | 3 | M | **DONE** |
| [10](10-T20a-preferences.md) | T20a · Preferences migration | 3 | M | **DONE** |
| [11](11-T6-custom-fields-api.md) | T6 · Custom fields v2 — API | 3 | M |
| [12](12-T7-custom-fields-frontend.md) | T7 · Custom fields v2 — frontend + retire v1 | 3 | L |
| [13](13-T9-selective-export.md) | T9 · Selective field export + sensitivity gating | 3 | L |
| [14](14-T12a-etag-primitives.md) | T12a · Activity/LifeEvent ETag primitives | 2\* | S |
| [15](15-T24-test-coverage.md) | T24 · Non-critical test-coverage expansion | 2 | M |
| [16](16-T8-openapi.md) | T8 · OpenAPI coverage + drift test | 2\* | M |
| [17](17-T17-change-feeds.md) | T17 · Change feeds + cursor pagination | 2\* | M |
| [18](18-T23-ui-polish.md) | T23 · UI polish | 4 | M |
| [19](19-T22-legacy-audit.md) | T22 · Legacy/dead-code audit + migration squash | 3 | L |

**→ ALPHA — real data begins here**

\* T8/T12a/T17 are rated on user-visible value. If a mobile client is real they are 4s. See the open
questions in `../95-backlog-and-priorities.md`.

## After alpha

| # | Ticket | R | Size | Status |
|---|---|---|---|---|
| [20](20-T19-cadence.md) | T19 · CadencePolicy + relationship health | 5 | L |
| [21](21-T21-conversation-agenda.md) | T21 · ConversationAgenda | 4 | M |
| [22](22-N2-prep-view.md) | N2 · Prep view / person briefing | 5 | M |
| [23](23-T10-graph-traversal.md) | T10 · Graph traversal + multi-hop | 2 | M–L |
| [24](24-T11-search-fts5.md) | T11 · Search synonyms, household scope, FTS5 | 5 | L |
| [25](25-N8-2fa.md) | N8 · 2FA / TOTP | 3 | M |
| [26](26-N6-backup-restore.md) | N6 · Full backup restore | 3 | M |
| [27](27-N5-bulk-operations.md) | N5 · Bulk operations | 3 | M |
| [28](28-T20b-gift-tracking.md) | T20b · Gift tracking | 3 | M |
| [29](29-N7-attachments.md) | N7 · File / document attachments | 3 | M |
| [30](30-N9-notification-channels.md) | N9 · Notification channels beyond email | 3 | M |
| [31](31-P1-contact-sharing.md) | P1 · Contact sharing — one-time copy | 3 | M |
| [32](32-T14-external-link-substrate.md) | T14 · External-link substrate | 2 | M |
| [33](33-T15-T16-immich.md) | T15/T16 · Immich level 1 + 2 | 3 | M each |
| [34](34-T18-audit-trail.md) | T18 · Event history / audit trail | 2 | L |
| [35](35-T12b-caldav-serve.md) | T12b · Serve Interactions/LifeEvents as CalDAV | 2 | L |
| [36](36-T13-two-way-calendar.md) | T13 · Two-way calendar sync ⚠ | 2 | M–L |
| [37](37-deferred.md) | P1b, P2, P3, P4 · Deferred — need design passes | 1–2 | — |
