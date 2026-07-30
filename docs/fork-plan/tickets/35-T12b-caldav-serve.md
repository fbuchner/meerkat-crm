# T12b — WP-87 serve Interactions/LifeEvents as CalDAV

| | |
|---|---|
| **Rating** | 2 — reading your calendar *in* is where the value is; pushing *out* is thinner |
| **Size** | L |
| **Depends on** | [T12a](14-T12a-etag-primitives.md), [T5](03-T5-lifeevent-frontend.md) |
| **Alpha** | after |
| **Source** | `92.3` |

## What it is

Serve the CRM's own Interactions and LifeEvents **out** as a CalDAV/iCalendar collection that clients can
subscribe to. The existing CalDAV *import* (`services/calendar_sync_service.go` → Activities) is the read
half; this is the write/serve half.

Rated 2 deliberately: the import direction already delivers the high-value behaviour (your timeline
populates itself from your real calendar). Publishing CRM events back out is a nice-to-have.

## What exists to copy

- **`backend/carddav/`** — a full working DAV server for contacts: `backend.go`, auth (password **and**
  `carddav`-scoped API tokens via `middleware.LookupAPIToken`), sync-token handling, and `models.CardDAVSync`.
  **This is your template.** The CalDAV equivalent should mirror its structure rather than invent one.
- **`Contact.ETag`** — the per-resource ETag pattern; [T12a](14-T12a-etag-primitives.md) brings the same
  to `Activity` (and `LifeEvent`), which is why it is a hard dependency.
- `services/calendar_sync_service.go` — the existing iCalendar *parsing* side; reuse its library and
  vocabulary for generation rather than adding a second iCal dependency.
- `models.CalendarSubscription` / `CalendarEventLink` — the import-side link model.

## What to build

1. **A CalDAV collection endpoint** mirroring `carddav/backend.go`'s shape: PROPFIND, REPORT, GET.
2. **iCalendar generation** — `Activity` → `VEVENT` (or `VJOURNAL` for something that happened with no
   duration — decide and document, since it affects how clients render it), `LifeEvent` → recurring
   `VEVENT` for annual events.
3. **ETags and sync tokens** so clients can sync incrementally rather than re-fetching everything.
4. **Auth** reusing the existing shared path — password or a `carddav`-scoped token. Consider whether the
   scope should be renamed or a `caldav` scope added; `ApiToken.Scope` already exists as a column with
   `full` ⊇ `carddav`.

## Traps

- **`PartialDate` life events.** A year-only event has no month/day and cannot be a calendar event at all.
  Skip them explicitly rather than emitting something malformed.
- **Read-only to start.** Two-way is [T13](36-T13-two-way-calendar.md) and carries real risk; do not let
  scope creep merge them.
- Timezones: `Activity.Date` is a `time.Time`; life events are date-only. Emit `DATE` vs `DATE-TIME`
  values correctly or clients will shift them by a day.
- Recurring life events need a sane `UID` that is stable across regenerations, or clients will duplicate
  them on every sync.
- Do not serve `secret`-sensitivity items (`91.13`) — a calendar subscription leaves the instance.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- Tests cover generation, ETag stability across an unrelated write, and incremental sync via sync-token.
- **Verified against at least two real clients** (e.g. Thunderbird and a phone). DAV interop is not
  something a unit test establishes — the CardDAV side was verified this way too.
- A year-only life event proven to be skipped cleanly, not emitted broken.
