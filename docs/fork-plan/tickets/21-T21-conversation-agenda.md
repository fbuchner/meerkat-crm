# T21 — WP-96 ConversationAgenda

| | |
|---|---|
| **Rating** | 4 — underrated; high-frequency, low-effort, directly changes how conversations go |
| **Size** | M |
| **Depends on** | [T5](03-T5-lifeevent-frontend.md) |
| **Alpha** | after |
| **Source** | `92.6`, `91.11` |

## Why this exists

"Things to bring up next time I see them." Contextual memory surfaced on the contact view — explicitly
**not date-scheduled**, which is what distinguishes it from a Reminder. You do not want "ask about their
mother's surgery" firing at 9am on a Tuesday; you want it in front of you the moment you are talking to
them.

Also a dependency of [N2](22-N2-prep-view.md), the prep view, which is rated 5.

## The three-way distinction to preserve

This codebase deliberately separates:

| Concept | Driven by | Lives in |
|---|---|---|
| **Reminder** | a date | `models.Reminder` |
| **Task** | an action | external (Vikunja, via webhook) — not built here |
| **ConversationAgenda** | *the next time you talk* | this ticket |

An agenda item has no due date and no completion cron. It is surfaced by *context*, not by time. If you
find yourself adding a `remind_at`, you have built the wrong thing.

## What to build

1. **Entity + migration** per `91.11`. UUID PK, `entity_id` (a `Contact.VCardUID`), content, created-at,
   and a resolved/discussed flag with the date it was discussed. Follow `LifeEvent`'s template.
2. **CRUD + routes**, following `life_event_controller.go`'s idiom.
3. **Contact-page surface** — an always-visible list on the contact detail page. This must be *low
   friction to add to*: a single inline input, not a modal dialog, or nobody will use it mid-conversation.
4. **Mark as discussed** — one click, ideally with the option to attach it to the interaction that
   covered it (`Activity`), which then feeds the timeline.
5. **Frontend** api module + hook + component, modelled on `relationshipEdges.ts` /
   `useRelationshipEdges.ts` / `RelationshipEdgeList.tsx`.
6. **i18n** in all five locale files.

## Traps

- Resist adding scheduling. The value is precisely that it is *not* time-driven.
- Discussed items should not vanish irrecoverably — being able to see "we talked about this on the 3rd"
  is half the value. Soft-resolve rather than delete.
- Keyed by `Contact.VCardUID`, not the numeric ID.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green, with a real-DB round-trip test.
- `npx tsc --noEmit` clean, `npx vitest run` green, with a component test for add and mark-discussed.
- Verified in a real browser: add an item in one click from the contact page, mark it discussed, confirm
  it stays visible in a resolved state.
