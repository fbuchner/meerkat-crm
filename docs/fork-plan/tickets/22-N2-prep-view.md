# N2 — Prep view / person briefing screen

| | |
|---|---|
| **Rating** | **5 — practically necessary.** The difference between a database and a relationship OS |
| **Size** | M — read-side composition, almost no new persistence |
| **Depends on** | [T5](03-T5-lifeevent-frontend.md), [T19](20-T19-cadence.md), [T21](21-T21-conversation-agenda.md) |
| **Alpha** | after (by dependency, not by value) |
| **Source** | New (gap found in the 2026-07-30 product review) |

## Why this exists

Every ingredient is planned and **nothing assembles them**. There is no screen answering the question you
actually have five minutes before seeing someone: *what do I need to remember about this person right
now?*

That gap is structural — each ticket built its own slice and none owned the composition. This ticket is
the composition.

## What it shows

One screen, scannable in under a minute:

| Block | Source |
|---|---|
| Last interaction: when, what type, what was discussed | `Activity` timeline + notes |
| **How overdue this relationship is** | [T19](20-T19-cadence.md)'s derived health |
| **Open agenda items** — things to bring up | [T21](21-T21-conversation-agenda.md) |
| Household and close relationships — partner's and kids' names | `RelationshipEdge` + `Household` |
| Recent and upcoming life events | [T5](03-T5-lifeevent-frontend.md) |
| Upcoming dates — birthday, anniversary | existing birthday service |
| Food preferences, key custom fields | [T20a](10-T20a-preferences.md), custom fields v2 |

## What to build

1. **A composition endpoint or a composed frontend fetch.** Prefer a single backend endpoint
   (`GET /contacts/:id/briefing`) over six frontend round trips — this screen is latency-sensitive by
   nature, and a mobile client would want the same thing.
2. **The surface.** Either a distinct tab on the contact page or a dedicated "prep" mode. A tab is less
   work and more discoverable; a dedicated mode can be denser. Pick one.
3. **Graceful degradation.** Every block must render sensibly when its source is empty or its feature is
   not yet built. This is what lets a reduced version ship as soon as T19 lands, before T21.
4. **Optional: an "upcoming" list** — everyone you are due to see soon, each with their briefing one click
   away. That turns this from a lookup into a workflow, and is arguably where the real value is.

## Traps

- **Do not persist a briefing.** Everything here is derived; a cached briefing is a staleness bug waiting
  to happen. If it is slow, fix the queries or add a request-scoped cache, not a table.
- Relationship labels must respect direction — use `getEffectiveType`/`getDisplayLabel` from
  `api/relationshipEdges.ts`, which already handle the inverse-token logic. Do not re-derive it.
- Only `status: confirmed` edges belong here. A suggested edge is not fact.
- Respect `91.13` sensitivity: a `secret` relationship or field should not surface on a screen likely to
  be open in front of the person it concerns.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- A test proves the briefing composes correctly and that each block degrades cleanly when empty.
- `npx tsc --noEmit` clean, `npx vitest run` green.
- Verified in a real browser against a contact with a full picture — relationships, life events, agenda
  items, an overdue cadence — and against a nearly-empty contact.

## Note

Sized M despite rating 5 because it is composition over entities that already exist. If it is growing new
tables, the scope has drifted.
