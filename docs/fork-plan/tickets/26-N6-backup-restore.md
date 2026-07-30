# N6 — Full backup restore

| | |
|---|---|
| **Rating** | 3 |
| **Size** | M |
| **Depends on** | — |
| **Alpha** | after |
| **Source** | New (gap found in the 2026-07-30 product review) |

## The gap

**Export is complete. Import is contacts-only.** So an exported instance cannot be restored.

- `ExportData` (`controllers/export_controller.go`) writes a combined CSV with `=== CONTACTS ===`,
  `=== RELATIONSHIPS ===`, `=== ACTIVITIES ===`, `=== NOTES ===`, `=== REMINDERS ===` sections, plus
  custom-field columns.
- `ExportContactsAsVCF` / `ExportContactsAsJSContact` cover contacts richly.
- **Import** (`import_service.go`, `import_controller.go`) handles CSV/VCF/JSContact **contacts only** —
  there is no path back in for notes, activities, reminders, or relationships.

## Decide the scope first — this ticket has two honest answers

**(a) Document SQLite file-level backup/restore and declare app-level restore out of scope.**
For a single-file SQLite self-hosted app this is the *correct* answer for disaster recovery: stop the
service (or use `VACUUM INTO` / the backup API for a consistent online copy — note WAL mode is on, so
naively copying the `.db` file alone is not safe), copy the file, done. Cheap, complete, and honest.
Deliverable is a documented, tested procedure in `docs/deployment.md` plus possibly a `make backup`
target.

**(b) Build app-level import for the remaining entity types.**
Only worth it if the real requirement is *partial* restore or *instance migration* — moving to a new
host, merging two instances, cherry-picking one contact's history back. That is a different need from
disaster recovery, and it is the only thing (a) does not cover.

**Recommendation: do (a) first and see whether (b) is ever actually wanted.** (a) is a fraction of the
work and covers the case that actually loses data.

## If you build (b)

- Reuse the import session machinery (`import_session.go`) — preview, partial success, and confirm are
  already there and already handle partial-failure semantics.
- Entity order matters: contacts before anything referencing them; relationship edges need both endpoints
  present.
- Identity: the CSV carries numeric IDs, which will not survive a restore into a fresh DB. Match on
  `VCardUID` instead, and note the export must therefore include it — **check whether it currently
  does**, because the contacts section exports `ID`, not `uid`.
- Idempotency: re-running a restore must not duplicate.
- **Soft-deleted rows.** A file-level backup contains them; an app-level export probably should not.
  Either way, a restore must not resurrect rows that were sitting in
  [T26](08b-T26-delete-semantics.md)'s purge queue — decide whether restore excludes
  `deleted_at IS NOT NULL` rows, and document it. Restoring a contact the user deleted is a surprising
  and arguably privacy-relevant failure.

## Done when

**For (a):** the procedure is documented and **actually tested** — back up a populated instance, destroy
it, restore, and verify contacts, notes, activities, reminders, relationships, and photos all survive.
Include the WAL caveat and the profile-photo directory, which lives outside the database file.

**For (b):** round-trip tests prove export → import into a fresh DB reproduces every entity type, with a
real-DB test rather than mocks, plus an idempotency test for a repeated restore.
