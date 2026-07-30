# P1b / P2 / P3 / P4 — deferred, need design passes

| | |
|---|---|
| **Rating** | 1–2 |
| **Alpha** | after |
| **Source** | Tier 5, `92.7`, `90` D1, `80` |

These are **not implementation-ready and are not meant to be.** Each needs its own design pass before it
can be broken into work packages. This file records what they are and what a design pass would have to
settle, so nobody starts one by accident.

## P1b — Contact sharing: standing/live share + permission model

**XL. Depends on [P1](31-P1-contact-sharing.md).**

Everything Tier 5's section describes beyond the one-time copy: persistence for a *live* share that
re-syncs, a shared-vs-private field model, a real permission model, and **re-confirmation when a field is
newly marked sensitive after the share was created**.

A design pass must settle:
- Does a standing share re-apply the field-selection default on every sync, or only at creation time?
  (Tier 5 flags this as an open question, not a decision.)
- What is the permission model — read-only, read-write, revocable, time-bounded?
- What happens to already-shared data when a share is revoked?
- How does the recipient's own editing interact with incoming updates? This is the same reconciliation
  problem as [T13](36-T13-two-way-calendar.md), with the same trap.

**Do not start this as part of P1.** P1 is deliberately a one-time copy; conflating them is what produced
the original XL estimate.

## P2 — Other integrations

**Depends on [T14](32-T14-external-link-substrate.md).**

Dawarich/GeoPulse, Jellyfin, Audiobookshelf, Paperless-ngx, Nextcloud. Each is a
`93-integration-spec-template.md` instance on top of the T14 substrate — mostly level 1–2, API-based, no
upstream dependencies.

Explicitly **pulled in only when a concrete need arises**, one at a time. If building one requires
changing T14's substrate, that is a signal the substrate is wrong — fix it there rather than
special-casing.

## P3 — AI / Ollama layer

**Rating 1.** Summarization, entity/relationship/life-event extraction, timeline synthesis,
memory-curator suggestions.

Gated on two things: everything structured existing first, and the **propose-then-approve** workflow —
which is already the pattern used by household inference ([T1](09-T1-households.md)) and its
suggested-edge review surface. Any AI output must land as a *suggestion* a human confirms, never as fact.

**`90` D1 is explicit: this is not an AI-first project.** Would revisit D1's storage decision *only* if
vector search proves necessary, and then via an external sidecar — never a primary-store migration.

## P4 — Local-model code-gen pilot

**Rating 1.** See `80-local-model-pilot.md`. Deferred on its own terms; re-enters when **mobile client
work begins**, independent of this backend roadmap.

Note this connects to the open question flagged in `95-backlog-and-priorities.md`: whether a mobile client
is real at all. That answer re-rates [T8](16-T8-openapi.md), [T12a](14-T12a-etag-primitives.md), and
[T17](17-T17-change-feeds.md) from 2 to 4 — so it is worth settling before those tickets rather than
before this one.
