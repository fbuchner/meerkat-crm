# T5b — LifeEvent → reminder wiring

| | |
|---|---|
| **Rating** | 4 — strong, frequent use |
| **Size** | S |
| **Depends on** | [T5](03-T5-lifeevent-frontend.md) |
| **Alpha** | before |
| **Source** | New (gap found in the 2026-07-30 product review) |

## Why this exists

Birthdays auto-generate reminders. Life-event dates generate nothing — so "work anniversary", "death
anniversary", "sobriety date", "wedding anniversary" are inert data once entered. This completes T5;
without it, alpha cannot evaluate whether life events are useful, because they do nothing.

## What exists today

- `backend/services/birthday_service.go` — `GetUpcomingBirthdays(db, userID, now)` is the **single**
  source feeding three consumers: the dashboard widget, the daily reminder email, and the
  `birthday.occurred` webhook trigger. Its `Birthday` DTO (`models/dtos.go`) had its `Type` narrowed to
  always `"contact"` when the legacy relationship path was retired in Tier 3b — so the multi-source
  shape it once had is gone and would need reintroducing carefully.
- `services.DaysUntilBirthday` — verified correct including the Dec 31 → Jan 1 wrap (forward-looking
  only, pinned by tests). Reuse it rather than writing new date math.
- `models.Reminder` — has `Recurrence`, `ContactID`, `ByMail`, `Completed`, `LastSent`.
- `services/reminder_service.go` — `SendReminders` eligibility filter is
  `completed=false AND email_sent=false`, pinned by `TestSendReminders_ExcludesCompletedAndAlreadySentReminders`.

## What to build

Pick one of two shapes and say which in the commit message:

- **(a) Derived, like birthdays** — extend the upcoming-dates query to also scan `life_events` with a
  recurring flag, and widen the DTO to carry a source discriminator. Nothing new is persisted. Cleaner
  model; touches the shared `GetUpcomingBirthdays` path and its three consumers.
- **(b) Materialised** — when a `LifeEvent` opts in, create a real recurring `Reminder` row linked to it.
  Simpler to reason about and reuses the whole existing reminder pipeline for free, but needs
  create/update/delete to keep the reminder in sync with its event.

**(b) is the smaller, lower-risk change** and is recommended unless you have a reason to prefer (a).

Either way:
1. Add an opt-in flag on `LifeEvent` (migration + model + DTO + the T5 dialog).
2. Annual recurrence from the event's month/day.
3. Surface it wherever birthdays already surface, so there is one "upcoming" concept, not two.

## Traps

- **`PartialDate` year-only events have no month/day to remind on.** The opt-in must be *unavailable*
  (disabled with an explanation), not silently broken, for those.
- Migrations here are hand-written SQL up/down pairs — never add the column by editing the struct alone.
- If you touch `GetUpcomingBirthdays`, remember all three consumers (dashboard, email, webhook) and the
  frontend `DashboardPage.tsx` shape.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- A **real-DB test** (`database.InitDB`) seeds a life event with an opted-in recurring date and asserts a
  reminder fires on the right day, including across a year boundary.
- A year-only event is proven to be rejected/disabled rather than producing a broken reminder.
- Hand-verified: break the recurrence logic, confirm the test fails, restore.
