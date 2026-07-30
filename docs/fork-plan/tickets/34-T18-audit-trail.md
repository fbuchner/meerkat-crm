# T18 — WP-93 event history / audit trail

| | |
|---|---|
| **Rating** | 2 — additive and safe, but see the cost below |
| **Size** | L |
| **Depends on** | [T17](17-T17-change-feeds.md) |
| **Alpha** | after |
| **Source** | `92.5` |

## What it is

An immutable create/update/delete/merge/restore log across the entities, feeding three consumers per
`92.5`: **undo**, **sync**, and **debugging**. Also extends webhook event coverage to the new entities.

## The cost of deferring — worth understanding before agreeing to it

Technically this is pure-additive: one new append-only table, nothing existing changes shape. That is why
it sits post-alpha.

But an audit log **only knows what happened after you switch it on**. Everything from the alpha period is
unrecoverable — no undo, no "what changed this record," no debugging history for exactly the period when
you are most likely to want it, because that is when the app is newest and least trustworthy.

That is a judgment call about how much you value alpha-period history, not a migration hazard. If undo
matters to you during alpha, this belongs earlier.

## What to build

1. **An append-only event table** — entity type, entity id, operation, actor (user), timestamp, and a
   before/after diff or full snapshot. Immutable: no update or delete path, ever.
2. **Capture points.** Two options, and the choice matters:
   - **GORM hooks** (`AfterCreate`/`AfterUpdate`/`AfterDelete`) — catches everything automatically,
     including writes from services and migrations, but is invisible at the call site and fires inside
     transactions you may not control.
   - **Explicit service-layer calls** — visible and controllable, but every new write path is a chance to
     forget.
   Hooks are the safer default for an audit log specifically, because completeness is the property that
   matters. Say which you chose and why.
3. **Undo** — replay an event backwards. Note this is genuinely hard for deletes that cascaded across ~14
   tables; scope what is undoable rather than promising everything. A realistic first cut is undo for
   updates and single-entity deletes only.

   **⚠ Undo of a delete cannot reach further back than [T26](08b-T26-delete-semantics.md)'s retention
   window** — after the purge job runs, the soft-deleted row is genuinely gone and there is nothing to
   restore *to*. Either bound the undo affordance to that window and say so in the UI, or have the audit
   trail keep its own full snapshot of deleted rows (which makes it the recovery store, with all the
   volume and secret-handling consequences noted below). Decide which; do not let a user click "undo" and
   get silence.
4. **Webhook event coverage** for the newer entities, reusing `services/webhook_service.go` (signing,
   retry, delivery records, and the job-locked retry processor all exist).

## Traps

- **Volume.** Every write generating a row with a full snapshot grows fast on a single-file SQLite DB.
  Decide retention (and whether it is configurable) up front, not after the file is 2GB.
- **Do not log secrets.** Password hashes, TOTP secrets ([N8](25-N8-2fa.md)), API token hashes, and OIDC
  tokens must never reach the audit table. An audit log is a secondary copy of everything — treat it with
  the same care as the primary.
- **Sensitivity (`91.13`)** applies to audit rows too: a `secret` relationship's content should not become
  readable via the audit surface.
- Soft-deleted rows already exist (`gorm.Model`); do not confuse "soft deleted" with "audited delete."
- Transactions: an audit write that fails must not roll back the real write, and an audit write inside a
  rolled-back transaction must not persist. Pick which side you want and be deliberate.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- Tests prove: every entity's create/update/delete produces exactly one event; the table rejects mutation;
  no secret-bearing field ever appears in an event; undo restores an updated record correctly.
- Hand-verified: remove one capture point, confirm the completeness test fails, restore.
- Volume measured against a realistic write pattern, with the retention decision written down.
