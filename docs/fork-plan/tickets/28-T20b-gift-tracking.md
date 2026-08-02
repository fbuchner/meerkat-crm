# T20b — WP-95 Gift tracking

| | |
|---|---|
| **Rating** | 3 — seasonal but genuinely used |
| **Size** | M |
| **Depends on** | [T5](03-T5-lifeevent-frontend.md) |
| **Alpha** | after — a brand-new entity, nothing to migrate |
| **Source** | WP-95 (split), `92.6`, `91.11` |

## Why this is split from T20a

WP-95 bundled Preferences and Gifts. They have **opposite alpha risk**: Preferences migrates a live,
populated field (hence [T20a](10-T20a-preferences.md), pre-alpha), while gift tracking is a brand-new
entity with nothing to migrate — purely additive and safe to defer.

## What to build

Per `91.11`. A gift record against a contact, covering the three states that make the feature useful:

| State | Meaning |
|---|---|
| **Idea** | "she mentioned she liked X" — captured whenever you notice, which is the whole point |
| **Offered/given** | what you actually gave, and when |
| **Received** | what they gave you — useful for reciprocity and for remembering to say thanks |

Plus: occasion (birthday, holiday, life event), optional value/price, and a link to the `LifeEvent` or
`Activity` it relates to where one exists.

Follow `life_event_controller.go`'s idiom for CRUD and `LifeEvent`'s template for the model (UUID PK,
`UserID`, keyed to the contact by `VCardUID`).

## Where it surfaces

The capture side matters more than the reporting side. A gift *idea* is recorded opportunistically — mid
conversation, months before it is needed — so adding one must be near-zero friction, the same argument
as [T21](21-T21-conversation-agenda.md)'s inline input. A modal buried three clicks deep will not get
used, and an empty gift-ideas list is worse than none.

Consider surfacing open ideas on the contact's [prep view](22-N2-prep-view.md) and near upcoming
birthdays, where they are actually actionable.

## Traps

- `Activity.Type` already includes an `InteractionTypeGift` constant. Decide the relationship between a
  logged gift *interaction* and a gift *record* — they are not the same thing, and having both without a
  clear story will confuse. Simplest: giving a gift optionally creates the interaction, and the gift
  record is the durable object.
- Keyed by `Contact.VCardUID`, not the numeric ID.
- If you add money values, be explicit about currency rather than assuming one.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green, with a real-DB round-trip test.
- `npx tsc --noEmit` clean, `npx vitest run` green.
- Verified in a real browser: capture an idea in one interaction, mark it given, see it against the
  contact and against the relevant occasion.

### Post-alpha note
This ticket is post-alpha — real production data exists. Changes that modify schemas or data must be additive and non-destructive. Migration files must be hand-written SQL up/down pairs. Test against `database.InitDB`, not `AutoMigrate`.

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
