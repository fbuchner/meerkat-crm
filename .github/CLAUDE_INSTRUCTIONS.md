# Instructions for Claude running in GitHub Actions

The local `CLAUDE.md` at the repository root is gitignored, so it does **not** exist
in a CI checkout. Any rule Claude must follow when it runs from a GitHub workflow has
to live here, in a committed file. This file is the authority for CI; it is not a copy
of anyone's personal `CLAUDE.md` and does not depend on one existing.

Depth is in [README-developer.md](../README-developer.md) (architecture, per-file
detail, Docker, E2E) and [.env.example](../.env.example) (every environment
variable). Read those instead of guessing; do not copy them into this file.

## Stack

- **Backend:** Go 1.26 + Gin + GORM + SQLite (migrations embedded, auto-run on startup)
- **Frontend:** React 19 + TypeScript + MUI + i18next (CRA, dev server on 7300)

## Commands

```bash
# Backend (from backend/)
go vet ./...
go test ./...

# Frontend (from frontend/)
yarn tsc --noEmit
CI=true yarn test
```

The workflow has already run `yarn install --frozen-lockfile` and set up Go and
Node 22, so these run without further setup.

## Definition of done

A change is not finished until all of these hold. Do not report work as complete
with any of them outstanding — say explicitly which one is failing instead.

1. **Backend touched** → `go vet ./...` and `go test ./...` pass.
2. **Frontend touched** → `yarn tsc --noEmit` and `CI=true yarn test` pass.
3. **Any new user-facing string** → present in *all six* locales:
   `en.json`, `de.json`, `fr.json`, `es.json`, `it.json`, `zh.json`. English-only is incomplete.
4. **Any date shown to a user** → formatted through `DateFormatProvider`, which
   honors the user's date-format preference. Never `toLocaleDateString` or a
   hand-rolled format.
5. **New contact field** → every layer of the checklist below, or it silently
   loses data.

The local `.claude/hooks/` verification hooks do not run in CI. In a workflow you
must run the suites yourself.

## Traps specific to this codebase

**Two independent column allowlists.** A new `contacts` column must be added to *both*:

- `contactSelectableFields` in `backend/controllers/contact_controller.go`
- `CONTACT_FIELDS` in `frontend/src/ContactDetailPage.tsx`

Miss the frontend one and the partial-column `GET` never reads the field, returns a
zero value, overwrites real data in local state, and the next save persists the wipe.
This is a silent data-loss bug with passing tests — it shipped once already (`348a46f`).

**Contact field checklist.** Adding one touches, in order:
migration → `models/contact.go` → `models/dtos.go` → `middleware/validation.go`
(if it needs a validator) → `contact_controller.go` (allowlist + create + update) →
`services/monica_import_service.go` (if Monica maps it) →
`frontend/src/contactFields.ts` → `ContactDetailPage.tsx` allowlist →
`components/ContactInformation.tsx` → six locale files → tests both sides.

**Migrations** are numbered sequentially (`000026_*.up.sql` / `.down.sql`) and
embedded in the binary. Always write the matching `.down.sql`.

## Conventions

### Backend
- Controllers get the DB from `c.Get("db")`, the validated payload from
  `c.Get("validated")`, the caller from `c.Get("userID")`, and the trace id from
  `c.Get("request_id")`.
- Never decode the request body again in a controller — the validation middleware
  already did.
- Return `*apperrors.AppError`; the error middleware maps it to the JSON envelope.
- Every user-scoped query filters by `UserID`. Multi-entity writes go in a transaction.

### Frontend
- All network calls go through `api/client.ts` (auth headers, timeouts, 401 auto-logout).
- Custom hooks return `{ data, loading, error, refetch }`.
- MUI for all UI; `useTranslation()` for all copy.
- Page components live at `src/*.tsx` — there is no `src/pages/` directory.

## Git boundaries in CI

These apply to Claude running inside a GitHub workflow. They are the defaults these
workflows are built around — a branch and a pull request for the maintainer to review,
and nothing that lands or rewrites work on its own:

- Commit and push **only** to the branch the action created for this run
  (`claude/**`). Open a pull request against `main`.
- Never commit or push to `main`, and never push to a branch you did not create.
- Never force-push.
- Never merge a pull request, and never enable auto-merge.
- Never rewrite history (`rebase`, `commit --amend`, `reset --hard` on a pushed branch).
- Never delete branches, tags or releases.
- Never change files under `.github/workflows/` unless the issue you are working on
  is explicitly about CI.

The maintainer reviews and merges every pull request by hand.
