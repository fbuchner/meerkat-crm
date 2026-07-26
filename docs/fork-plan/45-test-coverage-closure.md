# 45 — Test-coverage closure (TC-*)  ·  gates Part-B architecture review

## Context

A whole-backend coverage audit (2026-07-26) found the RFC-9553/9554 adapter core
(`contactmodel`/`jscontact`/`vcard4`/`vcard3`, 87-90%) and persistence/model layer (`models`, 89.8%)
solid, but `controllers` (34.9%) and `services` (48.0%) have real, confirmed gaps — most critically,
the entire import **confirm/persist** step (the code that actually writes imported contacts to the
database) has zero test coverage across all three formats (CSV/VCF/JSContact). Per §40's directional-
test philosophy, coverage must be *behavioral* (would catch a real regression), not just line
execution. This doc is the closure plan; the user's own gate is "when testing is in a good state to
enforce functionality and RFC/spec conformance" — no architecture/file-organization review (the
"Part B" questions — functional-unit separation, idiomatic Go, lint/docs) starts until this closes.

Numbering: sits between §40 (testing philosophy, still authoritative for *how* to write these tests)
and §50 (integration, whose WP-71/73/73b code this partly covers).

## Real coverage found (go test ./... -cover, 2026-07-26)

```
carddav 69.4%  contactmodel 89.5%  controllers 34.9%(low)  correspondence 84.6%
jscontact 87.8%  middleware 69.7%  models 89.8%  photostore 77.2%  services 48.0%(low)
vcard3 81.1%  vcard4 88.9%  httputil 0.0% (no test file)
```

## Work packages

### Phase 1 — Critical contact-data path (import→parse→persist→export→CardDAV), target >80%

- **TC-1.1** `services/import_session.go` (new `import_session_test.go`, **L**) — every function is
  0%, including `Confirm`/`ConfirmVCF` which contain the real `tx.Create`/`tx.Save` calls. Must prove:
  partial-success semantics (one bad row doesn't roll back the whole batch), session
  lifecycle (create/get/expire/delete), photo bridging on confirm (embedded + URL-fetched), and
  specifically that `ConfirmVCF`'s bulk `.Where("id = ?", ...).Updates(map{...})` call (line ~418)
  does not break `Contact.AfterSave`'s ETag recompute — the same GORM gotcha fixed once already this
  session (fix pattern: `db.First` the row, then `tx.Model(&loadedRow).Update(...)`).
- **TC-1.2** `controllers/import_controller.go` (new `import_controller_test.go`, **L**) — no test file
  exists at all. HTTP-boundary tests for all 6 handlers, sequenced *after* TC-1.1 lands (exercises its
  code end-to-end). Must include: oversized/malformed/wrong-extension uploads, cross-format session-ID
  confusion (VCF session hitting the CSV confirm route), user-scoping on preview/confirm.
- **TC-1.3** `controllers/export_controller.go` (extend existing test file, **M**) —
  `ExportContactsAsVCF`/`ExportContactsAsJSContact` are 0% (existing tests only cover the unrelated
  legacy CSV export). Must include version negotiation, photo-bridging (the exact
  `RecordFromContact`-vs-`RecordForContact` bug class fixed 3x this session), user-scoping.
- **TC-1.4** `services/import_service.go` gap-fill (**M**) — `ParseCSV`, `GenerateCSVPreview`,
  `GetStringField`, `NormalizeBirthday`, `NormalizeGender`, `CreateMergeNote` all 0%.
- **TC-1.5** Convert `vcard3/coverage_test.go` from its hand-maintained static `testedConcepts` map to
  the same `init()`-registration pattern `vcard4`/`jscontact` already use (**S**, mechanical) — closes
  the confirmed structural gap where a deleted vcard3 test file wouldn't fail the coverage gate.
- **TC-1.6** `ContactRecordInput` real-validator tests (**S/M**) — invalid/valid `Gender`, malformed
  JSON on Create/UpdateContact via the *real* `middleware.ValidateJSONMiddleware` (existing tests
  bypass it via a `withValidated` test helper), plus a **positive** test proving unusual-but-valid
  nested Card/CRM/Passthrough data is accepted, not rejected (guards the degradation-policy design).

### Phase 2 — This session's own remaining gaps

- **TC-2.1/2.2** `contact_sync_service.go` + `calendar_sync_service.go` dial-context / error-
  classification / rate-limit wrapper functions (**S each**, one agent for both — near-duplicate code,
  worth cross-checking against each other).

### Phase 3 — Pre-existing surface (user: "even for pre-existing code, ensure trust for changes")

**3a — security/auth-sensitive, dispatch first:**
`httputil/fetch.go` (SSRF, mirror `photostore`'s existing suite), `services/webhook_service.go`
(signature/SSRF/retry — flag `isPrivateURL`'s fail-open-on-DNS-failure vs `httputil`'s fail-closed
asymmetry to the user rather than silently "fixing" it), `services/oidc_service.go` (unit-testable
parts only; `InitOIDCProvider`/`ExchangeAndVerify` need a fake discovery server — attempt if tractable,
otherwise document the gap), `services/password_reset_service.go`.

**3b — lower-risk, dispatch after 3a (some reuse 3a's fixtures):**
`services/mailer.go` (exclude `sendViaResend` — no test seam without a production-code change;
document as a follow-up), `services/email_renderer.go`, `services/reminder_service.go` remaining gaps,
`services/birthday_service.go`'s `DaysUntilBirthday` (test the Dec 31→Jan 1 boundary explicitly),
and the pre-existing 0%-handler controllers (`graph`, `oidc`, `photo`, `user` remaining, `reminder`
remaining, `relationship.GetIncomingRelationships`, `admin_user` — check admin/last-admin-protection
invariants specifically, privilege-escalation-adjacent).

## Verification discipline (unchanged from the rest of this fork's work)

Every work package: dispatch → independently read the diff → re-run the new tests plus the package's
full `-cover` run → for Phase 1/2 specifically, hand-verify at least one negative-path assertion
actually fails when the corresponding production branch is temporarily commented out, before marking
done. Do not trust an agent's own "tests pass" report at face value.

## Exit criterion

Phase 1 packages all land, are independently verified, and push `controllers`/`services` critical-path
functions past 80% with assertions that would catch a real regression. Phase 2 closes this session's
own remaining gaps. Phase 3 is pursued for "trust changes anywhere" but is not itself a hard gate on
starting the Part-B architecture review — Phase 1 is.

## Phase 1+2 status: CLOSED (2026-07-26)

Every function on the critical path is now >80%, independently verified (diff read, tests re-run by
hand, and for the two highest-stakes cases — `ConfirmVCF`'s GORM/ETag interaction and the
mis-routed-session-confirm panic — the regression was actually reproduced by temporarily reverting the
fix and confirming the test fails before restoring it):

```
ParseCSV 88.2%  ParseVCF 89.7%  ParseJSContact 94.9%  GenerateCSVPreview 100%
GetStringField 100%  NormalizeBirthday 100%  NormalizeGender 100%  CreateMergeNote 94.7%
import_session.go lifecycle fns (New/generateSessionID/CleanupExpired/get/Delete/
  CreateCSVSession/CreateVCFSession/PreviewCSV/buildActionMap) 100%
Confirm 87.7%  ConfirmVCF 82.7%
UploadCSVForImport 91.2%  UploadVCFForImport 90.3%  UploadJSContactForImport 90.3%
PreviewImport 86.7%  ConfirmImport 85.7%  ConfirmVCFImport 85.7%
ExportContactsAsVCF 90.6%  ExportContactsAsJSContact 82.4%
```

**A real, live bug was found and fixed in the process**: `ConfirmVCF`'s photo-write used a bare
`db.Model(&models.Contact{}).Where("id = ?", id).Updates(map{...})` — the same class of GORM
bulk-update-breaks-`AfterSave` bug fixed once already this session in `contact_sync_service.go`. This
one was live, not theoretical: the bulk call's zero-value receiver broke `Contact.AfterSave`'s own
`tx.Model(c).UpdateColumn` sub-update (`ErrMissingWhereClause`), and because GORM wraps the call plus
its hooks in an implicit transaction, that hook failure silently rolled back the photo/thumbnail write
on every VCF-import photo confirm. Fixed with the established pattern: load the row first via
`db.First`, then `.Model(&loadedRow).Updates(...)`. Reproduced by reverting and confirming the test
failed, then restored.

`vcard3/coverage_test.go`'s hand-maintained static map was converted to the mechanical
`init()`-registration pattern (matches `vcard4`/`jscontact`); transcription was verified byte-for-byte
against the 29 test files and found to be a clean, false-claim-free no-op.

Two branches judged not worth chasing further (documented rather than silently skipped): `ParseVCF`'s
per-block "malformed vCard" error path is effectively unreachable — `go-vcard`'s decoder was tested
against 8 different malformed shapes once BEGIN/END framing is present (guaranteed by
`splitVCardBlocks`'s regex) and never errored; and `Confirm`/`ConfirmVCF`'s `tx.Create`/`tx.Save`
DB-failure branches require genuine SQL constraint-violation injection, lower value than the behavioral
gaps closed above.

Overall package coverage moved `controllers` 34.9%→42.4%, `services` 48.0%→64.4% — both still below 80%
in aggregate because they include large amounts of Phase-3 (pre-existing, non-critical-path) code; the
critical-path functions themselves are what the >80% gate was measured against, per the user's explicit
framing, and that gate is now met.
