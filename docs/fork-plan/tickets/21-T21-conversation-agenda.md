# T21 — WP-96 ConversationAgenda

| | |
|---|---|
| **Rating** | 4 — underrated; high-frequency, low-effort, directly changes how conversations go |
| **Size** | M |
| **Depends on** | [T5](03-T5-lifeevent-frontend.md) |
| **Alpha** | after |
| **Source** | `92.6`, `91.11` |

## Why this exists

"Things to bring up next time I see them." Contextual memory surfaced on the contact view — explicitly
**not date-scheduled**, which is what distinguishes it from a Reminder. You do not want "ask about their
mother's surgery" firing at 9am on a Tuesday; you want it in front of you the moment you are talking to
them.

Also a dependency of [N2](22-N2-prep-view.md), the prep view, which is rated 5.

## The three-way distinction to preserve

This codebase deliberately separates:

| Concept | Driven by | Lives in |
|---|---|---|
| **Reminder** | a date | `models.Reminder` |
| **Task** | an action | external (Vikunja, via webhook) — not built here |
| **ConversationAgenda** | *the next time you talk* | this ticket |

An agenda item has no due date and no completion cron. It is surfaced by *context*, not by time. If you
find yourself adding a `remind_at`, you have built the wrong thing.

## What to build

1. **Entity + migration** per `91.11`. UUID PK, `entity_id` (a `Contact.VCardUID`), content, created-at,
   and a resolved/discussed flag with the date it was discussed. Follow `LifeEvent`'s template.
2. **CRUD + routes**, following `life_event_controller.go`'s idiom.
3. **Contact-page surface** — an always-visible list on the contact detail page. This must be *low
   friction to add to*: a single inline input, not a modal dialog, or nobody will use it mid-conversation.
4. **Mark as discussed** — one click, ideally with the option to attach it to the interaction that
   covered it (`Activity`), which then feeds the timeline.
5. **Frontend** api module + hook + component, modelled on `relationshipEdges.ts` /
   `useRelationshipEdges.ts` / `RelationshipEdgeList.tsx`.
6. **i18n** in all five locale files.

## Traps

- Resist adding scheduling. The value is precisely that it is *not* time-driven.
- Discussed items should not vanish irrecoverably — being able to see "we talked about this on the 3rd"
  is half the value. Soft-resolve rather than delete.
- Keyed by `Contact.VCardUID`, not the numeric ID.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green, with a real-DB round-trip test.
- `npx tsc --noEmit` clean, `npx vitest run` green, with a component test for add and mark-discussed.
- Verified in a real browser: add an item in one click from the contact page, mark it discussed, confirm
  it stays visible in a resolved state.

### Ticket-specific
- "Contextual memory surfaced on the contact view" — explicitly NOT date-scheduled (what distinguishes it from Reminder)
- Model: follow `Circle`/`LifeEvent` pattern (UUID PK, UserID, keyed to contact by VCardUID). New entity → decide soft vs hard delete per T26 rule.
- Items are free-text + optional link/reference — simple schema: `id, user_id, contact_vcard_uid, content, reference_url, created_at, updated_at, deleted_at`
- Frontend: a list on the contact detail page. Follow `RelationshipEdgeList`/`LifeEventList` pattern for the component structure.
- i18n: simple strings — section header, add button, placeholder text, empty state
- Delete semantics: user-authored content → soft delete per T26

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
