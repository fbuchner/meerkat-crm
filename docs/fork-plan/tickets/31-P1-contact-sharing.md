# P1 — Contact sharing between users (one-time filtered copy)

| | |
|---|---|
| **Rating** | 3 — **potentially 4 if alpha is the two-user/spouse scenario** |
| **Size** | M — re-sized from XL, see below |
| **Depends on** | [T9](13-T9-selective-export.md) |
| **Alpha** | after (safe — additive, and user-initiated per acceptance) |
| **Source** | Tier 5 in `95-backlog-and-priorities.md` |

## Why this is M and not XL

Tier 5's original text imagines a **standing, live, permissioned share** — a shared-vs-private field
model, a permission model, re-syncing, and re-confirmation when a field is newly marked sensitive after
the share exists. That is genuinely XL, and it is now [P1b](37-deferred.md).

The near-term feature is much smaller: **export a filtered copy, let the other user import it on accept.**
Nearly every piece already exists.

| Step | Reuses |
|---|---|
| Field selection + filtering | [T9](13-T9-selective-export.md)'s picker and `Record`/`Card` filter — `92.6b` says explicitly it is meant to be reused here |
| Serialize the filtered record | `jscontact.Adapter{}.Export(record)`, already behind `ExportContactsAsJSContact` |
| Parse on accept | `services.ParseJSContact` (`import_service.go:263`) |
| Duplicate detection + preview | `services.DetectDuplicate` + the import-session preview/confirm flow (`import_session.go`) |
| Write into the recipient's account | `MergeImportedContact` / `ApplyRecordToContact` |

## What to build — the genuinely new surface

1. **`ContactShare` entity + migration** — from-user, to-user, the serialized filtered payload, status
   (`pending` / `accepted` / `declined`), timestamps. UUID PK, following `LifeEvent`'s template.
2. **Endpoints**: create a share, list incoming, list outgoing, accept, decline.
3. **Accept** feeds the payload through the existing parse → preview → confirm path, so the recipient
   sees what they are getting before it lands.
4. **Frontend**: a share dialog wrapping T9's field picker, and an incoming-shares inbox.

## The decision to make deliberately, not inherit

**What accepting a share does when the recipient already has that person.**

`MergeImportedContact`'s policy is *"incoming wins if non-empty, existing survives if blank"* — confirmed
and pinned by Tier 3c item 11c's tests. That policy was written for **imports**, and it is plausibly wrong
here: a shared copy arguably should **not** overwrite the recipient's own carefully-kept notes and edits
on someone they already track.

Choose explicitly between create-new, merge-into-match, and ask-the-user, and write the reason down.
Defaulting into the import path's behaviour by accident is the failure mode.

## Traps

- **Sensitivity gating matters more here than for file export** — a misclick discloses to a live person on
  the instance, not just to a file. T9's foot-gun guard (sensitive items behind a deliberate extra action,
  not merely unchecked) must be in place before this ships. That is why T9 is a hard dependency.
- The payload is persisted while pending. If the `Card` shape changes, an old pending share may not import
  cleanly — version the payload or expire pending shares.
- Both users are on the same instance, so scope every query by both `from_user_id` and `to_user_id`; a
  user must never see a share they are not party to.
- Declining should not silently destroy the sender's copy of what they offered.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- Tests cover: a share round-trips sender → recipient with only the selected fields present; a
  non-selected field is genuinely absent; a sensitive field is excluded unless explicitly opted in; a
  third user can see neither side of the share.
- The merge-on-accept decision is documented in the controller's doc comment.
- `npx tsc --noEmit` clean, `npx vitest run` green; verified in a real browser with two real accounts.

### Ticket-specific
- "One-time copy" — not live sync, not permission-based access. User selects a contact, picks fields, generates a shareable export.
- Reuses the field-picker from T9 (selective export). If T9 is not done, this ticket must build a minimal picker.
- The share link/token: generates a one-time access token, returns a URL. The recipient gets a read-only view or download.
- Token model: UUID, contact_id, field_selection (JSON), expires_at, access_count. Expire after first access or time limit.
- No new auth flow — the token IS the auth for the shared view
- Frontend: "Share" button on contact detail → picker → "Copy link" / "Download file"

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
