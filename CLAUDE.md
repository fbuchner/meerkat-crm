# Working in this repo

Mycorrhizal CRM — a hard fork of [meerkat-crm](https://github.com/fbuchner/meerkat-crm) being rebuilt
into a *personal relationship OS*: a Go/Gin + SQLite backend with a React/TypeScript/MUI frontend,
CardDAV/CalDAV sync, and a neutral RFC 9553/9554/9555 contact model.

This file is the conventions and hard-won traps. **Read it before writing code here** — several of the
items below are recurring bug classes that have shipped broken more than once.

## Orientation

| Where | What |
|---|---|
| `docs/fork-plan/95-backlog-and-priorities.md` | **The ticket board — the live plan.** Read the board section; the Tier 0–6 sections below it are historical. |
| `docs/fork-plan/tickets/` | One file per ticket, self-contained enough to implement from. |
| `docs/fork-plan/91-envelope-data-model.md` | Entity specs with field tables. The detailed source. |
| `docs/fork-plan/92-delivery-roadmap.md` | WP scope. **Not** the execution order — the board is. |
| `docs/fork-plan/00`–`50` | Neutral model, adapters, correspondence, integration history. |
| `backend/` | Go. Gin + GORM + SQLite, raw-SQL migrations. |
| `frontend/` | React 18 + TypeScript + MUI + vitest + Playwright. |

**No production data exists yet.** Alpha comes after the pre-alpha tickets on the board. Until then,
breaking changes and clean removals are cheap and preferred over compatibility shims. Do not add
backwards-compatibility scaffolding "just in case."

## Commands

```bash
cd backend && go build ./... && go vet ./... && gofmt -l . && go test ./...
```

```bash
cd frontend && npx tsc --noEmit && npx vitest run
```

Migrations: `cd backend && make migrate-up` (see `make help`). Migration files are **hand-written SQL
up/down pairs** in `backend/database/migrations/` — this project does **not** use GORM `AutoMigrate` for
schema. Never add a column by editing a model struct alone.

Dev server: use the Browser/preview tooling with `.claude/launch.json`'s `frontend-dev`, never a raw
`npm start` in a shell. The backend needs `JWT_SECRET_KEY`, `PROFILE_PHOTO_DIR`, `SQLITE_DB_PATH`, and
`FRONTEND_URL` matching the frontend's actual port or CORS will fail.

## Workflow

- **One branch per concern.** `feature/<thing>`. Implement → verify → commit per concern → push → merge
  only once confirmed working → delete the branch → update the board.
- **Research → plan → approve → implement → verify.** Use plan mode for anything with real design
  decisions. This cadence is established and expected.
- **Never commit to `main` or merge without being asked.**
- **Hand-verify your tests.** Break the code, confirm the new test actually fails, restore. A test that
  has never failed has proven nothing. This has caught real bugs here repeatedly.
- Update `docs/fork-plan/95-backlog-and-priorities.md` when a ticket lands.

## Backend traps

These are real bugs that shipped, not hypotheticals.

1. **Test against the real migrated schema, not `AutoMigrate`.** GORM's column-name derivation silently
   disagrees with the hand-written migration SQL, and `AutoMigrate`-based tests cannot see it. Use
   `database.InitDB(filepath.Join(t.TempDir(), "x.db"))` for anything touching persistence.
   - `HouseholdMember.MemberVCardUID` → GORM derived `member_v_card_uid`, migration said
     `member_vcard_uid`. Caught by a real-DB test.
   - `ContactSyncLink.ETag` → GORM wrote `e_tag`; the column is `etag`. **Shipped broken**, would have
     silently killed CardDAV incremental sync. Add explicit `gorm:"column:..."` tags for anything with an
     acronym or unusual casing.

2. **Never set `Card`/`CRM` by direct field mutation before `Create`.** `BeforeSave` derives the flat
   denormalized columns from the nested model; mutating the struct field directly skips it and your data
   silently doesn't persist. Use `ApplyRecordToContact`. This bit WP-81 and WP-83 the same way.

3. **`RecordForContact`, not `RecordFromContact`.** The former reads what is actually persisted
   (including data with no flat-field home — `SpeakToAs`, `PersonalInfo`, projections); the latter
   rebuilds from flat fields and **silently drops** that data. Using the wrong one was a live bug found
   across three call sites.

4. **Check `.Error` on every `db.Updates`/`db.Save`.** Three sites silently swallowed failures until
   audited.

5. **Ownership scoping is not optional.** Every handler scopes by `user_id` (or `Contact.VCardUID` for
   WP-80+ graph entities). There are zero IDOR holes today — keep it that way.

6. **Cascade deletes are manual.** Soft delete does not fire SQL `CASCADE`. `DeleteContact` and
   `DeleteUser` enumerate every dependent table explicitly — if you add an entity, add it there. Use
   `contact_controller.go`'s `DeleteContact` as the canonical checklist.

7. **`gorm.DB.Transaction` returns the closure's error verbatim**, so you can return a typed
   `*apperrors.AppError` from inside and type-assert it after to preserve a 404/400 instead of
   flattening to 500. `relationship_edge_controller.go` does this.

### Backend conventions

- Controllers: follow `circle_controller.go` / `life_event_controller.go` (the newer idiom) over older
  ones. `currentUserID(c)`, `middleware.GetValidated[T]`, `apperrors.AbortWithError`,
  `GetPaginationParams`.
- Join rows (membership) get **real nested sub-resource endpoints** (`POST/DELETE /circles/:id/members`),
  not a bulk-replace field. A duplicate add is a checked `409 ErrAlreadyExists`, not a sniffed constraint
  error.
- UUID-PK entities (`RelationshipEdge`, `Household`, `Circle`, `Tag`, `LifeEvent`, `FieldValue`) generate
  their ID in `BeforeCreate`. Everything older uses `gorm.Model`'s uint PK.
- Validation lives in struct tags + `middleware.ValidateJSONMiddleware`; custom validators
  (`phone`, `birthday`, `safeurl`, `relation_type`) are registered in `middleware/`.
- Sensitivity (`normal|private|secret`, `91.13`): anything above `normal` is excluded from exports and
  external sync **in the query**, not in the caller.

## Frontend traps

1. **vitest here has no auto-cleanup and no `globals: true`.** Add `afterEach(cleanup)` explicitly in
   component test files or you get "multiple elements found" failures.
2. **MUI appends `" *"` to a required field's accessible label.** `getByLabelText('Name')` fails;
   `getByLabelText('Name *')` works.
3. **Do not nest a `<Chip>` (renders `<div>`) inside `<Typography variant="body2">` (renders `<p>`).**
   Invalid HTML; React warns. Put both in a sibling flex `Box`.
4. **Frontend enum/registry lists are hardcoded mirrors of backend `oneof` validators.** There is no
   dynamic type-list endpoint anywhere in this codebase, by design. If you add a token backend-side, the
   frontend copy must be updated by hand — add a comment noting it must stay in sync.
5. **All five locale files get real translations** (`en`, `de`, `es`, `fr`, `it`), not English
   placeholders. No test enforces parity, but this repo does not ship placeholders.
6. **Playwright's `e2e/global-setup.ts` hardcodes `http://localhost:7300`** for both the app origin and
   its direct API calls, separately from `playwright.config.ts`'s `baseURL`. The e2e suite cannot run
   against a different port without editing shared test infra.

### Frontend conventions

- The contact model is nested (`Card`/`CRMEnvelope`/`Passthrough` via `ContactRecordResponse`). The flat
  `Contact` type survives **only** for the list endpoint (genuinely flat on the wire) and for
  `MultiValueField`/`AddressFields`' editing contract. Do not reintroduce a flat adapter.
- API modules live in `src/api/<entity>.ts`, hooks in `src/hooks/use<Entity>.ts`, and dialogs/lists in
  `src/components/`. `relationshipEdges.ts` + `useRelationshipEdges.ts` + `RelationshipEdgeDialog/List`
  are the most recent, most complete example of the full pattern — copy that shape.

## Domain notes worth knowing

- **`RelationshipEdge.Type` describes the *source's* role relative to the target.** `type: "parent_of"`
  from A to B means "A is B's parent." Only one direction is ever stored; the inverse is derived from
  `models/relationship_type_registry.go`, never persisted. Creating from a contact's page sends
  `target_id: <viewed contact>`, so a dropdown label always describes the *other* party.
- **Only `status: confirmed` edges are fact.** `suggested` edges (household-inferred) must never be
  projected to standards, graphed, or treated as real outside a review surface.
- **Cadence resets on a *qualifying interaction*, not on completing a task** (`91.10`).
  `Activity.Qualifying()` exists for this and has had no consumer yet.
- **CardDAV/REST writes are full-overwrite by design.** `reconcileContactSync` intentionally discards
  local edits on remote change — documented, pinned by a test. Do **not** copy that policy into new
  two-way sync paths without deciding deliberately (see the T13 ticket).
- The three exporters (`vcard3`, `vcard4`, `jscontact`) all consume the same neutral `Card`, so filtering
  the `Record` *before* it reaches an exporter applies to all three at once.

## Security posture

A full security review landed (14 findings, all patched — see `95`'s Tier 1). Keep it: parameterized SQL
only, no `os/exec`, templates from an embedded FS, `user_id` scoping everywhere, explicit field
allowlists on updates (no mass assignment), CSV values neutralized against formula injection, SSRF guards
enforced in the transport dialer. Go toolchain is pinned; don't float it.

Known and accepted: no 2FA yet (ticketed as N8).
