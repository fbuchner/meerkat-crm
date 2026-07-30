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
