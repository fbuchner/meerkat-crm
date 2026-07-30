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
