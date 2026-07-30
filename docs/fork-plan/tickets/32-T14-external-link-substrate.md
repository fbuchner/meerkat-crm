# T14 — WP-89 external-link substrate (ExternalIdentity / ExternalActivity)

| | |
|---|---|
| **Rating** | 2 alone — invisible infrastructure that enables 3s |
| **Size** | M |
| **Depends on** | — |
| **Alpha** | after (all-new tables; nothing existing changes) |
| **Source** | `92.4`, `91.12` |

## Why this exists, and why it comes first

The generic substrate is built **before** the first concrete integration, deliberately, **so that no
integration grows bespoke tables.** That ordering is the entire point of this ticket — keep it even
post-alpha. If Immich is built first, Immich gets its own columns and the next integration gets its own
too, and you have the sprawl this design exists to prevent.

Rated 2 on its own because a user sees nothing from it; it is the enabler for
[T15/T16](33-T15-T16-immich.md) and later `92.7` integrations.

## What to build

Per `91.12`:

- **`ExternalIdentity`** — this contact *is* this thing in that external system. Contact `VCardUID`,
  system identifier (e.g. `immich`), external ID, optional deep-link URL, last-synced timestamp.
- **`ExternalActivity`** — something that happened in an external system, linkable into the timeline.
  System, external ID, type, occurred-at, payload/summary, and the contact(s) it concerns.
- **A generic link/enrichment API** — CRUD over both, plus the ability to attach an `ExternalActivity`
  into a contact's timeline.

## What already anticipates this

- **`Activity.ExternalRef`** (`backend/models/activity.go`) — added in WP-84 specifically to link an
  Interaction to an `ExternalActivity`, and it has had no consumer since. Use it rather than inventing a
  parallel link.
- `93-integration-spec-template.md` — the per-integration template each future integration instantiates.
  Read it; this substrate should satisfy what that template assumes.

## Traps

- **Resist system-specific fields.** The moment `ExternalIdentity` grows an `immich_person_id` column,
  the abstraction has failed. System-specific data goes in a JSON payload field — follow
  `RelationshipEdge.Metadata`'s `map[string]interface{}` + `serializer:json` pattern.
- Keyed by `Contact.VCardUID`, not the numeric ID — the graph invariant.
- **Cascade delete**: add both tables to `DeleteContact` and `DeleteUser`, and to the real-DB cascade
  test. Every WP-80+ entity is scoped by `VCardUID` there.
- Unique constraint on (system, external ID, user) so a re-sync does not duplicate.
- External URLs are user-influenced — the SSRF guard (`httputil/fetch.go`, dialer-level) applies to
  anything this fetches.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- A **real-DB test** covers the round trip, the unique constraint, and cascade cleanup on contact delete.
- The API is demonstrably generic — sketch how a second, unrelated integration (say Paperless-ngx) would
  use it without schema changes. If it cannot, the abstraction is wrong and Immich will prove it later.
