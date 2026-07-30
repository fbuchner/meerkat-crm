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
