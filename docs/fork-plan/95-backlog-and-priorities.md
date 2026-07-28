# 95 — Backlog and priorities

> **Unlike `00`–`94`, this file is a living document, not a historical planning record.** Re-groom it
> whenever priorities shift. It is the durable source of truth for "what's next" — the in-session task
> tool is used only for tracking the one or two items actively being worked on right now, since it has
> repeatedly lost its full contents mid-session and should not be trusted as the backlog's home.
>
> Last groomed: 2026-07-27.

## How to read this

Tiers are ordered by "best use of time for immediate impact," not by when the idea was conceived.
Within a tier, do items top-to-bottom. Re-groom (re-run this judgment call) whenever a tier completes,
a security concern surfaces, or the user's priorities change — don't treat the ordering below as fixed
forever.

## Tier 0 — Active: WP-72 frontend nested-model remodel

The backend already speaks the full RFC 9553/9554 neutral Card/CRM model (WP-70–71, done). The frontend
adapter shim (`frontend/src/api/contacts.ts`) makes the app *work* against it, verified end-to-end via
Playwright against `docker-compose.test.yml` (28/28 passing). But the shim collapses every multi-value
field down to a single scalar, so **the actual payoff of the RFC 9553 backend work — multiple emails,
phones, addresses, structured name/org data — is currently invisible to users.** Finishing this tier is
the highest-leverage work available: it's close to done, and it's what makes the last several months of
backend investment demoable.

| Order | Item | Size | Status | Why here |
|---|---|---|---|---|
| 1 | `contactFields.ts` field-key registry → nested keys | 93 lines | pending | Prerequisite for everything below; cheap |
| 2 | `MultiValueField.tsx` / `AddressFields.tsx` → real Card arrays | ~250 lines | pending | Prerequisite. The one generic component reused by every multi-value field — highest leverage per line |
| 3 | `ContactInformation.tsx` migration | 421 lines | pending | The actual payoff: multi-email/phone/address with structured types. Ships the visible feature. |
| 4 | `AddContactDialog.tsx` migration | 494 lines | pending | Same payoff, on the creation path — first impression |
| 5 | `ContactHeader.tsx` migration | 422 lines | pending | Lower urgency — name/nickname/photo already round-trip fine as scalars through the shim today, no functional gap |
| 6 | `ContactDetailPage.tsx` orchestration | 813 lines | pending | Glue layer; depends on items 4–5, biggest file, leave until the shape is settled |
| 7 | Migrate remaining peripheral consumers + retire the adapter shim | — | pending | Cleanup once 3–6 land |
| ongoing | Unit test coverage for the migrated contact-editing surface | — | pending | Write incrementally per-component as each one migrates, not saved to the end — cheaper against code just touched |

Branch: `feature/frontend-nested-model`.

## Tier 1 — Security review, before the data model grows further

Cheap relative to new feature work (a review pass, not new implementation), and better done *before*
Tier 2 (relationship graph, households) adds more PII surface than *after*. Also motivated by known
upstream issues and the general "don't assume inherited Meerkat code is correct" audit the user asked
for.

1. **Security audit of the core backend** (not just RFC 9553 additions) — Meerkat is still a dependency;
   evaluate it as a risk surface now that more sensitive personal data is going into this database.
2. **OIDC/OAuth implementation evaluation** — known issue areas upstream in Meerkat. Explicitly should
   not block client work, hence not Tier 0, but shouldn't wait for the full P5–P10 backend expansion either.
3. **Injection audit** — the other "actual exposure" item from the original backend-hardening brainstorm.

## Tier 2 — P5: Core relationship & event model (WP-80..84b)

Full detail already lives in `92-delivery-roadmap.md §92.1` — not duplicated here. Hard gate: nothing in
P6+ starts until this is green, since search, timeline, cadence, and integrations all read these
entities. Depends on P1/P0 (done), not on Tier 0/1 above — but sequenced after them here because it's a
large net-new backend surface, and finishing the client (Tier 0) plus a security pass (Tier 1) on the
*current* surface is worth more right now than starting the *next* one.

## Tier 3 — Remaining backend hardening/audits

Lower urgency than Tier 1's security-relevant items — risk-reduction and correctness, not exposure:
- Correctness audit
- Data-lifecycle audit
- Backup audit
- Silent-failure audit
- Single-instance-assumption audit
- CVE sweep of dependencies
- Broader test-coverage closure (Phase 3, remaining tiers beyond what `45-test-coverage-closure.md` already closed)

## Tier 4 — P6–P10 (search, CalDAV export, external links/Immich, sync, relationship-maintenance)

Unchanged from `92-delivery-roadmap.md §92.2–92.6` — gated behind Tier 2 (P5) by that doc's own hard
gate. See that file for the full WP breakdown and dependency graph (`§92.8`).

## Tier 5 — Contact sharing between users (standalone, big)

Two users on the same instance (e.g. spouses) should be able to share contacts directly — opting which
fields to include — rather than round-tripping through lossy VCF export/import. Explicitly a "revisit on
the roadmap at some point" item per the user, not scheduled. Comparable in scope to a P8-or-later phase;
needs its own design pass (data model for shared-vs-private fields, permission model) before it can be
broken into WPs.

## Deferred / someday

Unchanged from `92-delivery-roadmap.md §92.7`: other integrations (Dawarich/GeoPulse, Jellyfin,
Audiobookshelf, Paperless-ngx, Nextcloud), AI/Ollama layer. Pulled in only when a concrete need arises.

## Explicitly not re-ranked here

`80-local-model-pilot.md`'s deferral is independent of this backlog (re-enters when mobile client work
begins, per `92.9`).
