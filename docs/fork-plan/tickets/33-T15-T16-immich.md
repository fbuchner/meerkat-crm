# T15 / T16 — WP-90 & WP-91 Immich integration (levels 1 and 2)

| | |
|---|---|
| **Rating** | 3 — genuinely delightful; the user's own stated priority #5 |
| **Size** | M each |
| **Depends on** | [T14](32-T14-external-link-substrate.md) |
| **Alpha** | after |
| **Source** | `92.4`, `93-integration-spec-template.md` |

Two tickets, kept in one file because level 2 is a direct continuation of level 1.

## T15 — Immich level 1 (linking)

**Scope:** store the Immich Person ID against a contact, deep-link out to Immich, and display photo count
and latest appearance. **Pure CRM-side — no upstream Immich dependency**, which is why it is level 1 and
why it can ship independently.

**What to build:**
1. Immich connection config — base URL and API key, per user. Store the key encrypted, not plaintext.
2. Person search/browse against the Immich API to pick the right person for a contact.
3. Persist the link as an **`ExternalIdentity`** (`system: "immich"`) — not a new column on `Contact`.
   That is the whole reason [T14](32-T14-external-link-substrate.md) comes first.
4. Contact-page surface: photo count, latest appearance date, and a deep link into Immich.

## T16 — Immich level 2 (enrichment)

**Scope:** pull the latest photo and appearances into **`ExternalActivity`** records so they land on the
contact's timeline.

**What to build:**
1. A sync that fetches recent appearances for each linked person and writes `ExternalActivity` rows.
2. Timeline integration — appearances render alongside notes, activities, and life events.
3. Scheduling: reuse the existing job-lock pattern (`acquireJobLock`/`releaseJobLock` with a
   `minInterval`, as `calendar_sync_service.go` and `ProcessWebhookRetries` do) rather than a bare cron.
4. Thumbnails: prefer deep-linking or proxying over copying image data into this app. `proxy/image`
   already exists and is hardened.

**Level 3 (bidirectional) stays deferred** — it needs an "external links" capability upstream in Immich
that does not exist. That is a dependency, not a scheduling choice; do not attempt a workaround.

## Traps

- **SSRF.** The Immich base URL is user-supplied and typically a *private* address, which is precisely the
  case `WEBHOOK_BLOCK_PRIVATE_URLS` exists to make configurable. Decide the policy explicitly and
  document it — silently allowing private addresses here while blocking them for webhooks is confusing;
  silently blocking them makes the integration useless for most self-hosters.
- **The image proxy rejects SVG** and sets `Content-Disposition` + CSP, because an image served from the
  API origin is an XSS vector. Any thumbnail path must go through the same hardening.
- **Do not add Immich-specific columns.** If you need somewhere to put an Immich field, it goes in
  `ExternalIdentity`'s JSON payload. A column named `immich_*` means T14's abstraction failed.
- Immich API version drift — pin what you rely on and fail gracefully, since this is an external service
  you do not control.
- A person unlinked in Immich, or an Immich instance that is down, must degrade to "no photos" rather
  than erroring the contact page.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- Tests use a **fake Immich HTTP server** (the `oidc_service` fake-IdP fixture from Tier 3c item 9 is the
  precedent for a permanent, real-protocol test double here) rather than mocking at the client boundary.
- Failure paths tested: unreachable instance, expired API key, unlinked person.
- Verified against a **real Immich instance** — this is an integration; a green test suite is not
  sufficient evidence it works.
