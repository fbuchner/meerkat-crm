# Code Audit — Meerkat CRM

**Date:** 2026-06-11
**Scope:** Full codebase (~36k LOC: Go backend ~19k, React/TS frontend ~17k)
**Method:** Three parallel deep-dive reviews (backend security, backend maintainability/architecture, frontend), findings cross-verified against the actual code before reporting. Several agent findings turned out to be wrong on verification and are listed under "Rejected findings" so they don't resurface in future audits.

---

## Verdict

The codebase is in **good shape**. No critical or exploitable vulnerabilities were found. Authorization (multi-user isolation) is consistently enforced, JWT handling is correct, SSRF protection on the image proxy is exemplary, and the frontend has no XSS vectors. The main debt is architectural: a few god-files (import controller/service, ContactDetailPage), GORM models exposed directly as API responses, and missing test coverage for webhook/OIDC/mailer services.

---


### B2. GORM models returned directly from the API (Medium)
Controllers serialize models like `Contact` straight into responses. Any new DB column with a JSON tag silently becomes API surface, and eager-loaded relations (`Contact.Relationships` ↔ `Relationship.Contact`) risk bloated or circular payloads. **Recommendation:** introduce response DTOs, starting with `Contact` (the widest type).

### B3. Missing tests for side-effectful services (High)
No tests exist for `webhook_service.go` (delivery, retry, HMAC signing), `oidc_service.go` (provisioning/linking — the most security-sensitive untested path, see S1), `mailer.go`, `password_reset_service.go`, or `config.Validate()`. **Recommendation:** prioritize OIDC provisioning and webhook delivery tests.

### B4. Missing FK constraints on early user-scoped tables (Medium)
Migration `000004_add_user_scoping` added `user_id` columns without `FOREIGN KEY ... ON DELETE CASCADE` (later migrations have them). Deleting a user orphans contacts/notes/etc. Note: SQLite cannot `ALTER TABLE ADD CONSTRAINT` — fixing this requires the rebuild-table dance (create new table, copy, drop, rename) per table, so it's a deliberate migration, not a quick patch. Alternative: enforce cascade deletion in the user-deletion code path and verify with a test.

### B5. Repeated controller boilerplate (Medium)
The `gorm.ErrRecordNotFound` → 404 / else → 500 branch appears ~43 times; the `currentUserID` + `db` + pagination preamble ~15 times. A `notFoundOrDBError(c, err, resource, id)` helper in `controllers/helpers.go` would remove ~100 LOC. Worth doing as a dedicated mechanical PR rather than piecemeal.

### B6. Inconsistent success-response shapes (Low)
Mix of `gin.H{"message": ...}`, `gin.H{"contact": ...}`, and bare objects. Errors are uniform (via `apperrors`); successes aren't. Standardize when introducing DTOs (B2).

### B7. Contact scalar/array field sync is fragile (Low)
`Contact` keeps both `Email` and `Emails[]` (etc.), synced in a `BeforeSave` hook. Search reads the scalars. A regression test asserting sync after create/update/import would protect this contract.

### B8. Scheduler lifecycle (Low)
`main.go` runs gocron via `go s.StartBlocking()`; shutdown calls `s.Stop()` but a job mid-flight can outlive the 30s window. Acceptable for current job types (idempotent, rate-limited reminders); revisit if jobs grow.

---

## Open findings — Frontend maintainability

### F1. God components & structure drift (Medium)
`ContactDetailPage.tsx` (823 lines), `ImportContactsDialog.tsx` (699), `DashboardPage.tsx` (531), `App.tsx` (485). Also, pages live in `src/` root while `CLAUDE.md` documents `src/pages/` — either move the files or fix the doc. **Recommendation:** when next touching ContactDetailPage, extract the profile-editing and timeline state into hooks; split the import dialog's 5-step wizard into per-step components.

### F2. No request cancellation in hooks (Medium)
Custom hooks and page effects (`useTimelineEditing.ts`, `ContactDetailPage` parallel fetches) don't pass `AbortSignal` and don't guard against state updates after unmount. `apiFetch` already creates an `AbortController` for timeouts — extend it to accept an external signal and add cleanup in the hooks.

### F3. Heavy components not memoized (Medium)
`NetworkGraph.tsx` (force-graph rendering) and `ContactInformation.tsx` re-render whenever their parents do. Wrap in `React.memo` and stabilize callback props with `useCallback` in the parents.

### F4. `loginUser` discards backend error detail (Low)
`auth.ts:31` throws a generic `'Login failed'`, hiding useful backend messages (e.g. rate-limit lockout with `Retry-After`). Note: using raw `fetch` here instead of `apiFetch` is **correct** — a wrong password returns 401 and `apiFetch` would redirect-loop — but the error message could be parsed from the response body. A timeout via `AbortController` would also help.

### F5. Accessibility gaps (Low)
Icon-only buttons without `aria-label` (e.g. add-circle button in `AddContactDialog.tsx`), and a click-handling div instead of a button for file selection in `ImportContactsDialog.tsx:~325`. Quick wins for a focused a11y pass.

### F7. No i18n key-parity check (Low)
All four locales are currently in sync (verified), but nothing enforces it. A small CI script comparing key sets across `i18n/locales/*.json` would catch drift.

---

## Rejected findings (verified false — do not "fix")

1. **"Set OIDC state/nonce cookies to `SameSite=Strict`"** — wrong. The OIDC callback is a cross-site top-level navigation from the IdP; `Strict` cookies would not be sent on that redirect and the flow would break. `Lax` is the correct setting (cookies are sent on top-level GET navigations) and is what the code uses.
2. **"AccountRateLimiter leaks memory (entries never cleaned)"** — wrong. `middleware/rate_limiter.go` starts a background cleanup goroutine in `init()` that sweeps all limiters every 5 minutes.
3. **"auth.ts must use apiFetch (HIGH)"** — mostly wrong. Raw `fetch` in the login path is intentional (see F4); only the error-detail and timeout aspects are worth improving.
4. **"`ALTER TABLE ... ADD CONSTRAINT` migration for FKs"** — invalid SQL in SQLite; see B4 for what a real fix entails.
5. **"Import-session cleanup goroutines accumulate unboundedly"** — overstated. `cleanupExpiredSessions` is a microsecond map sweep under a mutex; spawning it per upload is wasteful but harmless.

---

## Strengths worth preserving

- **Multi-user isolation:** every user-scoped query checked filters by `user_id`; no IDOR found across contacts, notes, activities, reminders, graph, export, photos, webhooks, API tokens, or CardDAV.
- **Image-proxy SSRF defense** (`httputil/fetch.go`): scheme validation, IP checks at dial time (DNS-rebinding safe), redirect limits, size and content-type caps — textbook.
- **Auth hardening:** HMAC-only JWT validation, bcrypt, httpOnly cookies, per-account exponential-backoff lockout, password entropy enforcement, body-size limits.
- **Error handling foundation:** `apperrors` with request IDs, used consistently for failures.
- **Migrations:** idempotent, versioned, with down migrations.
- **Frontend hygiene:** httpOnly-cookie auth (no tokens in localStorage), no `dangerouslySetInnerHTML`, central `parseErrorResponse`/`ApiError`, hooks follow the documented `{ data, loading, error, refetch }` convention, locales in sync, no dead code or stale TODOs found.

## Suggested priority order for remaining work

1. Tests for OIDC provisioning and webhook delivery (B3) — highest risk-reduction per hour.
2. Cascade-delete behavior for user deletion (B4).
3. Extract import session management to a service (B1).
4. Response DTOs starting with `Contact` (B2 + B6).
5. AbortController support in `apiFetch` + hooks (F2), memoization (F3).
6. Controller boilerplate helper (B5) as a mechanical cleanup PR.
7. Grow the Vitest unit layer opportunistically when touching pure frontend logic (F6).
8. Component splits and `src/pages/` alignment (F1), a11y pass (F5), i18n parity CI check (F7).
