# N7 — File / document attachments per contact

| | |
|---|---|
| **Rating** | 3 |
| **Size** | M |
| **Depends on** | — |
| **Alpha** | after |
| **Source** | New (gap found in the 2026-07-30 product review; Monica has this) |

## Why this exists

There is no file storage against a contact today beyond the profile photo. Real uses: a CV, a scan of
something they gave you, a contract, an insurance document, a kid's drawing.

## What exists today — reuse it, do not invent a second mechanism

- **`backend/photostore/`** — the existing on-disk storage package for profile photos, with its own
  tests and a 1×1 PNG fixture convention.
- **`config.Config.ProfilePhotoDir`** (`PROFILE_PHOTO_DIR`, required at boot) — the established pattern
  for a configured on-disk directory, threaded explicitly through call sites rather than read globally,
  except in `models.DefaultPhotoDir` where a GORM hook's fixed signature forced a package-level var.
- **`controllers/photo_controller.go`** — upload handling, validation, and the delete path with its
  `!= ""` guards before `filepath.Join`.
- **Image proxy hardening** — SVG is rejected outright and `Content-Disposition` + CSP are set on that
  response, because SVG served from the API origin is an XSS vector. **The same reasoning applies to
  arbitrary attachments and is the main security consideration in this ticket.**

## What to build

1. **Model + migration** — an attachment record: contact `VCardUID`, stored filename (server-generated),
   original filename (display only), content type, size, uploaded-at.
2. **Storage** — a configured directory alongside the photo dir. **Server-generated filenames only**
   (UUID), never anything derived from user input reaching `filepath.Join`.
3. **Upload/download/delete endpoints**, scoped by `user_id`, with a size limit and a content-type policy.
4. **Frontend** — an attachments section on the contact page with upload, list, download, delete.

## Traps — this ticket is mostly security

- **Never build a path from a user-supplied filename.** Store a UUID; keep the original name as a display
  string only. `contact.Photo` is already a server-generated UUID for exactly this reason.
- **Serve downloads with `Content-Disposition: attachment`** and a restrictive CSP, and reject or
  neutralise anything renderable in-origin (SVG, HTML). The image-proxy finding in Tier 1 is the
  precedent — do not reintroduce it through a different door.
- **Backups.** Attachments live outside the SQLite file, so they change the backup story — coordinate
  with [N6](26-N6-backup-restore.md), which already has to account for the photo directory.
- **Cascade delete.** `DeleteContact` and `DeleteUser` enumerate every dependent table *and* must now
  clean up files on disk. A deleted contact leaving orphaned files is a real leak. Add it to both, and to
  the real-DB cascade test.
- Disk quota: nothing stops a user filling the volume. At minimum, a per-file size cap.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- Tests cover: upload/download/delete round trip; cross-user access denied; a traversal-style filename is
  neutralised; an SVG or HTML upload is rejected or served non-renderably; deleting a contact removes its
  files from disk.
- Hand-verified: remove the content-type guard, confirm the rejection test fails, restore.
- `npx tsc --noEmit` clean, `npx vitest run` green; verified in a real browser.

### Ticket-specific
- Storage: follow `backend/photostore/` exactly — server-generated UUID filenames only, never user-supplied names reaching `filepath.Join`
- Serve downloads with `Content-Disposition: attachment` default; inline only for image/pdf allow-list
- `X-Content-Type-Options: nosniff` on the serve endpoint — browser won't sniff HTML/SVG into renderable documents
- SVG/HTML uploads: reject or serve as download-only (in-origin SVG is XSS). Follow `photo_controller.go`'s hardening precedent.
- Cascade delete: add to `DeleteContact` and `DeleteUser` in `contact_controller.go` and `admin_user_controller.go` — these enumerate every dependent table. Also clean up disk files.
- Model pattern: a new entity with UserID + ContactVCardUID key. Content is user-authored → soft delete per T26.
- Backup expansion: attachments live outside SQLite, changing the backup story. Coordinate with N6.

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
