# T13 — WP-88 two-way calendar sync ⚠

| | |
|---|---|
| **Rating** | 2 |
| **Size** | M–L |
| **Depends on** | [T12b](35-T12b-caldav-serve.md) |
| **Alpha** | after — **but read the warning; this is the one deferred ticket with real risk** |
| **Source** | `92.3` |

## ⚠ The warning

**This ticket's risk is not schema — it is reconciliation policy applied to real data.**

By the time this is built, the calendar sync will be running against real events and real user edits.
Two-way means deciding **what wins when both sides changed**, and getting it wrong silently destroys
work.

There is a precedent in this codebase that must **not** be inherited by default:

> `services/contact_sync_service.go`'s `reconcileContactSync` **deliberately full-overwrites local edits
> on any remote change.** That is intentional, documented in `ApplyRecordToContact`'s doc comment, and
> pinned by `TestReconcileContactSyncOverwritesLocalEditsOnRemoteChange` (Tier 3c item 11a). The reason is
> that no model tracks per-field modified-since-sync state, so a field-level merge was not a small fix.

Applying that same policy to calendar data post-alpha means a remote calendar change silently discards
whatever the user typed into the CRM about that interaction. **Decide the merge semantics explicitly,
write them down, and test them against a scratch calendar — never a live one.**

## What exists today

- `services/calendar_sync_service.go` — one-way import (CalDAV → Activities), job-locked, with
  `CALDAV_SYNC_INTERVAL_HOURS` and `CALDAV_BLOCK_PRIVATE_URLS` config, and SSRF protection.
- `models.CalendarSubscription` + `models.CalendarEventLink` — the link table, which reconciles via
  **`ContentHash`**, not ETag. There is no `If-Match` primitive on this path today.
- `models.Contact.ETag` + the CardDAV two-way path — the closest working precedent for conflict handling
  in this codebase, and worth reading before designing this.

## What to build

1. **A conflict policy.** Options, roughly in increasing order of effort and correctness:
   local-wins / remote-wins / last-write-wins by timestamp / field-level merge / surface the conflict to
   the user. **Pick deliberately.** For a personal CRM, surfacing a conflict is often better than silently
   picking, because the volume is low and the cost of a wrong silent choice is high.
2. **ETag / If-Match support** on the outbound direction — `CalendarEventLink` currently has only
   `ContentHash`, which detects *that* something changed but not *who changed it first*.
3. **Push local changes out**: an Activity edited in the CRM propagates to the subscribed calendar.
4. **Deletion semantics**: deleted locally, deleted remotely, or deleted on both — three cases, and
   "deleted remotely" is easily confused with "not yet fetched."

## Traps

- Re-read the warning above before writing the reconcile function.
- `ContentHash`-based detection cannot distinguish "remote changed" from "local changed" on its own —
  you need per-side state, which is exactly what the contact path lacks and why it chose overwrite.
- Job-locked already; keep new work inside the lock so a multi-instance deploy does not double-write.
- SSRF guards apply to the remote calendar URL.
- A sync loop (your write triggers a remote change that triggers your write) is easy to create. Guard it.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- **The conflict policy is documented in the reconcile function's doc comment**, in the same style
  `reconcileContactSync` documents its overwrite behaviour — so the next person inherits a decision, not
  a mystery.
- Tests cover every conflict case explicitly: local-only change, remote-only change, both changed,
  deleted on each side, and no sync loop.
- Hand-verified: force the both-changed case and confirm the policy behaves as documented.
- Verified against a **scratch** calendar, never a real one.

### Post-alpha note
This ticket is post-alpha — real production data exists. Changes that modify schemas or data must be additive and non-destructive. Migration files must be hand-written SQL up/down pairs. Test against `database.InitDB`, not `AutoMigrate`. For integrations: SSRF protection via `httputil.SafeDialContext` is mandatory for any outbound requests.

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
