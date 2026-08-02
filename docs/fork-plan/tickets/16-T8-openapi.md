# T8 — OpenAPI coverage + spec/route drift test

| | |
|---|---|
| **Rating** | 2 (**4 if a mobile client is real**) |
| **Size** | M |
| **Depends on** | T1–T7 — so it documents the finished surface once, not twice |
| **Alpha** | before |
| **Source** | `92.9` |

## Why this exists

`backend/openapi.yaml` documents **13 of roughly 70** route patterns. `92.9` makes this binding rather
than cosmetic: every new entity is supposed to get summary/detail/OpenAPI treatment in the style WP-71
established for contacts, so a future Swift/Kotlin/Dart client (and the deferred local-model pilot, `80`)
targets one coherent spec rather than a patchwork.

## What is documented today

`/contacts`, `/contacts/{id}`, `/export/vcf`, `/export/jscontact`, the five import endpoints, and
`/contact-subscriptions` (+ `/{id}`, `/{id}/sync`).

## What is missing

Everything else. Get the live list with:

```bash
cd backend && grep -oE '(protected|v1)\.(GET|POST|PUT|PATCH|DELETE)\("[^"]+"' routes/routes.go \
  | sed 's/.*("//' | sort -u
```

Which covers, roughly: activities, notes, reminders (+ completions, upcoming, complete), circles (+
members), tags (+ contacts), life-events, relationship-edges (+ accept), households (new in
[T1](09-T1-households.md)), field definitions/values (new in [T6](11-T6-custom-fields-api.md)), graph,
webhooks (+ deliveries, test), calendars (+ sync), api-tokens, users (me, language, date-format,
custom-fields, enabled-contact-fields, change-password), admin users, auth (register, login, logout,
password reset, OIDC), photos, proxy, health.

## What to build

1. **Document every route** in the existing file's style — request/response schemas matching the actual
   controller behaviour, not aspirational shapes. Where a response envelope is inconsistent, **document
   the truth and note it**: `CreateRelationshipEdge` wraps as `{relationship_edge: …}` while
   `UpdateRelationshipEdge` and `AcceptRelationshipEdge` return the raw object. That asymmetry is real and
   already burned the frontend once.
2. **A drift test** that fails when a registered route has no spec entry (and ideally vice versa).
   `backend/openapi_test.go` already exists — extend it. Enumerate routes from the Gin router
   (`router.Routes()` after `RegisterRoutes`) rather than by parsing source, so it cannot go stale.
3. Note the deliberate omissions from the spec that are **not** bugs: `fields=` is gone and silently
   ignored; `includes=relationships` was removed in §3d WP4 and is likewise a no-op rather than an error.

## Traps

- Pagination shape is about to change in [T17](17-T17-change-feeds.md) (offset → cursor). **Coordinate**:
  either do T17 first, or document the current shape knowing you will revise it. Publishing the contract
  twice is exactly what T17's pre-alpha placement was meant to avoid — read T17 before starting.
- Auth: most routes sit behind `AuthMiddleware` with a cookie; CardDAV-scoped API tokens are rejected by
  the general REST path (a `full` token works, a `carddav` one does not). Document that.
- Do not document routes that only exist when OIDC is enabled as though they are unconditional.

## Done when

- `go build ./... && go vet ./... && go test ./...` green, including the new drift test.
- The drift test hand-verified: add a throwaway route without a spec entry, confirm it fails, remove it.
- The spec validates against an OpenAPI linter.
- Spot-checked: pick three documented endpoints and confirm a real request/response matches the schema.

### Ticket-specific
- Get the live route list: `cd backend && grep -oE '(protected|v1)\.(GET|POST|PUT|PATCH|DELETE)\("[^"]+"' routes/routes.go | sed 's/.*("//' | sort -u`
- Drift test: enumerate routes from `router.Routes()` after `RegisterRoutes()` — do NOT parse source. Extend `backend/openapi_test.go`.
- Document the actual controller behavior, not aspirational shapes. Where response envelopes are inconsistent (e.g. Create wraps, Update doesn't), document the truth and note it.
- Coordinate with T17: T17 changes pagination from offset→cursor. Either do T17 first (change then document once) or document current shape then revise.
- Deliberate omissions that are NOT bugs: `fields=` is gone (ignored), `includes=relationships` is gone (no-op)
- Spot-check: pick 3 documented endpoints, confirm a real request/response matches the schema

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
