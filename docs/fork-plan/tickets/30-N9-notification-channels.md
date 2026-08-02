# N9 — Notification channels beyond email

| | |
|---|---|
| **Rating** | 3 |
| **Size** | M |
| **Depends on** | — |
| **Alpha** | after |
| **Source** | New (gap found in the 2026-07-30 product review) |

## Why this exists

Reminders are **email-only**, which suits self-hosters poorly — running an SMTP path (or depending on
Resend) just to be told about a birthday is heavy, and many self-hosted setups have no outbound mail at
all. ntfy, Gotify, and web push are the idiomatic answers.

## What exists today

- `services/mailer.go` + `services/email_renderer.go` — templated email with embedded FS templates and
  i18n. `sendViaResend` has no test seam (documented, deliberately excluded from coverage).
- `services/reminder_service.go` → `SendReminders` — the daily job, gated by
  `completed=false AND email_sent=false`, wrapped in the `acquireJobLock`/`releaseJobLock` pattern with a
  `minInterval`. **`email_sent` is the delivery-state field, and it is email-shaped** — that is the main
  design problem this ticket has to solve.
- `services/webhook_service.go` — full delivery infrastructure with HMAC signing, retry with backoff,
  delivery records, SSRF guards enforced in the transport dialer, and a job-locked retry processor. A
  user could already wire ntfy through a webhook crudely.
- `models.Reminder.ByMail *bool` — a per-reminder channel flag already exists, in embryonic form.

## What to build

1. **Generalise delivery state.** `reminders.email_sent` assumes one channel. Either add per-channel
   delivery records (cleaner, mirrors `WebhookDelivery`) or generalise the flag. **Decide this first** —
   everything else follows from it, and getting it wrong means a second migration later.
2. **A channel abstraction** — an interface with email as the first implementation, so adding ntfy is a
   new implementation rather than a new branch in `SendReminders`.
3. **ntfy and/or Gotify** as the first non-email channel. Both are trivial HTTP POSTs; the work is
   configuration and delivery bookkeeping, not protocol.
4. **Per-user (and ideally per-reminder) channel preference**, generalising `ByMail`.
5. **Settings UI** for configuring the channel and testing it. A "send test notification" button is worth
   more than it sounds — misconfigured notifications fail silently otherwise.

## Traps

- **Reuse the SSRF guard.** A user-supplied ntfy/Gotify URL is exactly the attack shape
  `httputil/fetch.go`'s dialer-level protection exists for, and `WEBHOOK_BLOCK_PRIVATE_URLS` already
  configures this class of decision. A self-hosted ntfy is usually on a private address, so the *default*
  here likely differs from webhooks — make that an explicit, documented choice rather than an accident.
- **Do not bypass the job lock.** `SendReminders` is job-locked so a multi-instance deploy does not
  double-send. A new channel must sit inside the same lock.
- Failure in one channel must not block another, and must not mark the reminder as sent.
- Templates are i18n'd per user language (`reminder_service.go` reads `user.Language`); a push
  notification needs its own short-form strings in all five locales, not the email body.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- Tests cover: a reminder dispatches to the configured channel; a channel failure does not mark it sent
  and does not block other channels; the job lock still prevents double-send; a private-address target is
  handled per the documented policy.
- Verified end to end against a real ntfy or Gotify instance, not a mock.

### Ticket-specific
- This extends the existing notification infrastructure — study how `SendReminders` uses `sendReminderEmail`
- The mailer (`services/mailer.go`) already supports Resend and SMTP — new channels (Pushover, ntfy, gotify, Matrix) are additive
- Each channel needs: config in `Config` struct + env vars, a sender function, registration in the notification dispatch
- Webhook as notification channel: reuse `services/webhook_service.go`'s existing delivery infrastructure
- User preference: which channels are enabled per user. New column(s) on `User` model with migration.
- For each channel: test the happy path + error handling (network failure, invalid config, rate limiting)

## Flash implementation notes

### Files to read first
- `/CLAUDE.md` at repo root (conventions, recurring traps, commands)
- Study an existing fully-implemented feature for the pattern: model → controller → routes → api → hooks → dialog → list → page wiring → i18n
- Common pattern references: `circle_controller.go` + test (newer idiom), `api/relationshipEdges.ts` + hook, `RelationshipEdgeDialog.tsx` + test, the `ContactInformation.tsx` tab + `ContactDetailPage.tsx` wiring

### Tests you must write before considering it done
- Backend: controller tests covering CRUD, ownership scoping, error states (not found, cross-user, 409 duplicate)
- Backend: real-DB test (`database.InitDB`, not `AutoMigrate`) for the core round-trip + any migration-dependent behavior
- Frontend: component test (`afterEach(cleanup)`, mock `fetch` with `vi.stubGlobal`) for dialog and list
- Hand-verify EVERY new test: break the code, confirm the test fails, restore. A test that has never failed has proven nothing.

### Self-verification checklist
1. `npx tsc --noEmit` — clean
2. `npx vitest run` — green (ALL tests, not just yours)
3. `cd backend && go build ./... && go vet ./... && gofmt -l . && go test ./...` — green
4. New migrations: run `make migrate-up` to verify they apply cleanly
5. All 5 locale files (`de/es/fr/it/en`) — real translations for any new strings

### Common traps (beyond CLAUDE.md)
- `gorm.Model` only works on uint-PK entities — UUID PK models need explicit `ID`/`CreatedAt`/`UpdatedAt` fields
- Backend tests use `setupRouter()` from `activity_controller_test.go` (sets `db`, `userID`, `cfg` in Gin context, uses AutoMigrate)
- Frontend component tests: `afterEach(cleanup)` mandatory; MUI appends `" *"` to required field `getByLabelText`
- Migration files: hand-written SQL up/down pairs — never add a column by editing the struct alone
- `gorm:"column:xxx"` tag is mandatory for acronyms/compound words — GORM silently derives wrong names
- New entities: decide soft vs hard delete per T26's rule (user-authored content → soft, edge/join rows → hard)
- Delete cascade: add new entities to `deleteContactAssociations` in `contact_controller.go` and `DeleteUser` in `admin_user_controller.go`
