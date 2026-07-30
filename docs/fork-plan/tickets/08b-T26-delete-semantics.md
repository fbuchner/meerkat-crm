# T26 — Delete semantics: purge job + constraint fixes

| | |
|---|---|
| **Rating** | 3 — hygiene, but it fixes two live bugs |
| **Size** | M |
| **Depends on** | — |
| **Alpha** | **before** — bounds data accumulation and fixes two constraint bugs before real data exists |
| **Source** | New (design discussion, 2026-07-30) |

## Why this exists

Soft delete currently **has no consumer and no bound**. Verified:

- **No purge job exists.** Nothing ever reclaims a soft-deleted row.
- **No restore surface exists.** `Unscoped` appears exactly once in production code
  (`services/calendar_sync_service.go:652`), and not for undo.
- So a deleted contact's name, notes, and reminders stay in the database **forever**, invisible. For a
  CRM holding data about people who never consented to being in it, that is the wrong default.

Soft delete's two legitimate purposes are **undo** ([T18](34-T18-audit-trail.md), post-alpha) and **sync
tombstones** ([T17](17-T17-change-feeds.md)). Until those exist it is pure accumulation — and even after,
it needs a retention bound.

## The decision: delete semantics are a property of the model, not the call site

**One rule per entity, decided once when the model is created, never varying by which function is
deleting it.**

| Shape | Examples | Delete |
|---|---|---|
| User-authored content | `Contact`, `Note`, `Activity`, `Reminder`, `LifeEvent` | **Soft** |
| Edge- and join-shaped rows | `RelationshipEdge`, `CircleMember`, `ContactTag`, `HouseholdMember`, `ContactSyncLink`, `CalendarEventLink`, `FieldValue` | **Hard** |

**Why not make it depend on the operation** (e.g. "cascade deletes hard, single deletes soft"), which is
intuitively appealing: it makes `tx.Delete(x)` mean different things in different functions, so every
future cascade site becomes a chance to forget an `Unscoped()` — and the failure is *silent*, rows just
quietly linger. This codebase has already demonstrated it forgets cascade sites: Tier 3c item 1 found
**14 tables** that `DeleteUser`/`DeleteContact` had missed. Do not add per-call-site variance to a
pattern with that track record.

**The existing split is already right, and the unique constraints prove it.** Every table with a
natural-key composite unique index hard-deletes, so a dead row never blocks re-creating the same pair:

| Table | Unique constraint | Deletes | Collision? |
|---|---|---|---|
| `circle_members`, `contact_tags`, `household_members`, `field_values`, `field_definitions`, `contact_sync_links`, `calendar_event_links` | natural-key composites | hard | ✅ none |
| `contacts` | `(user_id, vcard_uid)` | **soft** | ⚠️ **bug — see below** |
| `users` | `username`, `email`, `(oidc_subject, oidc_provider)` | **soft** | ⚠️ **bug — see below** |

That is not a coincidence: a join row *is* identified by its endpoints, which is exactly what makes a
lingering dead one break re-creation.

## What to build

### 1. The purge job — the missing piece

- A scheduled job that hard-deletes (`Unscoped()`) soft-deleted rows older than a retention window.
- **Reuse the existing job-lock pattern**: `acquireJobLock`/`releaseJobLock` with a `minInterval` shorter
  than the cron cadence, exactly as `ProcessWebhookRetries` and the calendar sync do. Add a
  `JobNamePurgeDeleted` constant. A multi-instance deploy must not double-purge.
- **Retention window configurable** (e.g. `DELETED_RETENTION_DAYS`, default 30) via `config.getEnv`,
  documented in `.env.example` and `docs/deployment.md`.
- **An admin "purge now"** action, for the account-deletion case where waiting is not acceptable.
- **Respect FK ordering.** SQLite FK enforcement is on (`database.openDSN`'s `foreign_keys(1)`). Some
  children declare `ON DELETE CASCADE` (e.g. `Note`'s constraint tag), so hard-deleting a parent may
  cascade automatically — verify rather than assume, and purge children before parents where it does not.

### 2. Fix: `DeleteUser` hard-deletes

`users.email` and `users.username` are `UNIQUE`, and the row is currently soft-deleted — so **you cannot
re-register with the same email after deleting an account**, ever, since nothing purges. That is a
concrete technical reason, not a philosophical one.

Make `admin_user_controller.go`'s `DeleteUser` use `Unscoped()` for the user row and its soft-deleting
children. **This is the one deliberate call-site exception in the codebase** — document it in the
function's doc comment with the reason, so it reads as a decision rather than an inconsistency.

Rationale beyond the constraint: no sync client survives a deleted account, so there is nothing to
tombstone for, and "delete my account" should mean it.

### 3. Fix: partial unique index on `contacts`

`idx_contacts_vcard_uid_user` is `UNIQUE (user_id, vcard_uid) WHERE vcard_uid IS NOT NULL`. With soft
delete, a deleted contact keeps its `vcard_uid` reserved — so **re-importing or re-syncing that same
contact collides**. That is a live CardDAV scenario.

Migration: recreate the index as
`WHERE vcard_uid IS NOT NULL AND deleted_at IS NULL`. SQLite supports partial indexes and this one
already uses a `WHERE` clause, so it is a small change and the standard answer.

## Implications for other tickets — check these when you touch them

| Ticket | Implication |
|---|---|
| [T5](03-T5-lifeevent-frontend.md) | `LifeEvent` gets soft delete (its step 0) — consistent with the rule, and `life_events` has **no** unique constraint, so no collision. T26 settles the `DeleteUser` question T5 raises; if T26 has not landed, leave the plain `tx.Delete` and T26 converts it. |
| [T17](17-T17-change-feeds.md) | **The retention window becomes the sync horizon.** A client offline longer than it will have missed tombstones that were purged, and *must* full-resync. The feed needs a "your cursor is older than the retention window" response. |
| [T18](34-T18-audit-trail.md) | **Undo cannot reach further back than the retention window** for deletes, since the row is gone. Bound the undo affordance accordingly, or have the audit trail keep its own snapshot. |
| [N1](01-N1-contact-merge.md) | The merge loser soft-deletes, so its `vcard_uid` stays reserved until purge — fix 3 is what makes re-importing that contact work afterwards. |
| [T22](19-T22-legacy-audit.md) | The migration squash must carry the partial-index change; fold it into the clean baseline rather than replaying it. |
| [N6](26-N6-backup-restore.md) | A backup contains soft-deleted rows. A restore should not resurrect rows that were pending purge — decide and document. |
| New entities in [T19](20-T19-cadence.md), [T20b](28-T20b-gift-tracking.md), [T21](21-T21-conversation-agenda.md), [N7](29-N7-attachments.md), [T14](32-T14-external-link-substrate.md) | Each must pick soft or hard **per the rule above** at model-creation time. `CadencePolicy`, gifts, agenda items and attachments are content → soft. `ExternalIdentity`/`ExternalActivity` are link-shaped → hard. |

## Traps

- **`gorm.Model` only works on uint-PK entities.** The UUID-string-PK models have their own
  `ID`/`CreatedAt`/`UpdatedAt`; embedding it adds a conflicting `ID uint`. Add
  `DeletedAt gorm.DeletedAt \`gorm:"index" json:"-"\`` instead.
- **`admin_user_controller_test.go`'s `assertGone` counts via `db.Model(...).Count()`, which excludes
  soft-deleted rows** — so it passes whether a row is truly gone or merely marked. Any test that needs
  the distinction must use `Unscoped()`. This is why the `DeleteUser` change needs its own test.
- Purging must be idempotent and safe to interrupt — it will run on a cron forever.
- Do not purge rows whose `deleted_at` is null. Obvious, but a mis-built `WHERE` here destroys live data;
  test that case explicitly.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- A **real-DB test** (`database.InitDB`) proves: a row soft-deleted beyond the window is hard-deleted by
  the job; a row inside the window survives; a **live row is never touched**; the job is
  lock-protected and idempotent across two runs.
- A test proves re-registration with a deleted account's email now succeeds.
- A test proves a contact can be re-imported with the same `vcard_uid` after deletion — asserted against
  the **real migrated schema**, since it is an index-definition change.
- `DeleteUser`'s `Unscoped()` exception is documented in its doc comment, and pinned by a test that
  asserts with `Unscoped()` (the default helper cannot see the difference).
- Retention default documented in `.env.example` and `docs/deployment.md`.
