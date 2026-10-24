# Open PR triage — 20 unsolicited PRs

Reviewed 2026-10-24. All 20 are CI-green and individually mergeable into `main`;
every one is blocked only on review. So nothing here is about whether the code
*works* — it is about whether it belongs in this codebase.

Repo policy (README): AI assistance is fine, but the submitter owns the quality and
must be able to explain it in review. Worth knowing: **PRs 247 and 249 have
`Claude <noreply@anthropic.com>` as the git author**, not the submitter, and 247/248/249/251/259
all use `jip/<slug>/<random>` branch names from an agent harness. That is a signal to
ask the submitter to vouch for them, not proof of anything by itself.

## Verdicts

| PR | Title | Verdict |
|---|---|---|
| 259 | sync contacts with unquoted ETags | **Accept** |
| 222 | deceased field | **Accept** |
| 227 | toggle birthday reminders | **Accept** |
| 233 | respect date format in date fields | **Accept** |
| 253, 244, 241, 252, 257, 258, 242, 243, 245 | dependabot | **Accept** |
| 247 | golangci-lint in CI | **Trim** — keep the 3 real fixes, drop gosec |
| 249 | ESLint/tsc/Prettier gates | **Trim** — keep ESLint+tsc, drop Prettier |
| 251 | vCard 2.1 import | **Trim** — gate the normalizer on `VERSION:2.1` |
| 224 | per-contact vCard export | **Trim** — drop the 3.0/4.0 picker |
| 248 | Makefile + Docker-free E2E | **Reject the script**, Makefile optional |
| 219 | "translate extended relationship types" | **Reject** |

---

## Accept

**259 — unquoted ETags.** 3 backend files, dedicated test, conflicts with nothing,
newest PR in the set. A real CardDAV interop bug. Take it.

**222 — deceased field.** The one large PR that earns its size, because *it does not add
scope*: the Monica importer already writes a `Deceased` custom field
(`monica_import_service.go:71-74`), and this replaces that half-baked representation with
a real column. The backfill (`000025`) is careful work — it matches only the documented
value domain so `"no"` and `""` can't mark someone deceased, removes the now-redundant
custom field, cleans the stale key out of `users.custom_field_names`, and the `.down.sql`
documents what it cannot reconstruct and gets its ordering right. It also clears both
allowlist traps (`contactSelectableFields` *and* `CONTACT_FIELDS`) and all five locales.
This is the only non-trivial unsolicited PR here I would take unchanged.

**227 — birthday reminder toggle.** Not new scope either: `AddContactDialog.tsx:188-214`
on main already creates birthday reminders client-side. This extracts that into
`utils/birthdayReminder.ts` (−32/+8 in the dialog) and adds the toggle for *existing*
contacts. Fits the architecture it found.

**233 — date format in date fields.** Fixes an actual violation of CLAUDE.md rule 4 —
routes the activity/note/reminder date fields through `DateFormatProvider` instead of
ad-hoc formatting. Straight accept.

**Dependabot ×9.** Merge. 253 first — it is the only explicit CVE (CVE-2026-93749,
source-map-js DoS). 244 carries a security fix too. Merge 241 before 251 (one-line
`go.mod` conflict).

## Trim

**247 — golangci-lint.** The three fixes buried in here are genuinely good and worth
keeping on their own:
- `m.db.Query("VACUUM")` → `Exec` — a real `*sql.Rows` leak.
- Validator registration now panics instead of dropping the error.
- `strconv.Atoi` → `ParseUint` in the photo/relationship controllers — fixes gosec G115
  *and* turns a negative ID into a 400 instead of a database round-trip.

**Drop gosec from the linter set.** Every one of the eight `//nolint:gosec` comments this
PR sprinkles through the code exists to silence a non-issue — photo-directory permissions,
a table name that comes from config, a user id from an already-signed token. That is
permanent comment noise in your source to appease a linter you did not ask for, plus an
85-line exclusion file to maintain. `errcheck`, `staticcheck`, `bodyclose` and
`sqlclosecheck` earn their keep; `gosec` and `revive` do not, at this size.

Also note a deliberate behavior change to call out in release notes: `MinVersion:
tls.VersionTLS12` on implicit-TLS SMTP (465) breaks mail servers that only offer TLS
1.0/1.1. And `schedule()` only *logs* a failed job registration, so a malformed
`REMINDER_TIME` silently means no reminders — consider making that fatal.

**249 — lint gates.** ESLint and tsc both pass clean on `main` today (verified: exit 0,
zero findings), so those two jobs plus the `pull_request` trigger fix are free. Keep them.

**Drop Prettier entirely** — not "defer", drop. It checks whole files, and **83 of 103
ts/tsx files (81%) are not Prettier-clean**, so the "legacy files don't block unrelated
work" claim doesn't hold. Measured against the queue it would force 21 full-file
reformats that bury the real diffs: 233 → 7 of 7 files, 222 → 5 of 7, 224 → 3 of 4,
227 → 3 of 4, 219 → 3 of 3. The alternative (a repo-wide reformat commit) conflicts with
all five frontend PRs at once. Not worth it for a solo-maintained repo.

While editing: **declare `eslint` and `eslint-config-react-app` explicitly.** `npx eslint`
currently resolves to eslint **8.57.1** arriving transitively from `react-scripts@5.0.1`,
driven by the legacy `eslintConfig` block in package.json. eslint 8 is end-of-life, and if
react-scripts ever goes — plausible, `@vitejs/plugin-react` and `vitest` are already in
use — `npx` silently fetches eslint 9+, which ignores eslintrc entirely.

**251 — vCard 2.1 import.** The problem is real and in scope; contact import is a core
feature and old phones do export 2.1. But as written, `NormalizeLegacyVCF` is **not gated
on version** — it rewrites *every* import through 155 lines of hand-rolled text munging,
so any bug in it regresses the 3.0/4.0 imports that work today. It also switched to
`io.ReadAll` on the whole file instead of streaming into the decoder. Ask for a
`VERSION:2.1` gate before this merges; the risk/benefit is upside-down otherwise.

**224 — per-contact vCard export.** The export itself is useful and the route addition is
clean (`GET /contacts/:id/vcf`). **Drop the 3.0/4.0 version picker.** It adds two new UI
surfaces — a dropdown in `DataSettingsPage` and a submenu in `ContactHeader` — exposing a
format detail that almost no user of a personal CRM can evaluate. Pick one version, emit
it, keep the UI a single button.

## Reject

**248 — the `scripts/e2e-stack.mjs` half.** 135 lines of hand-rolled HTTP proxy, static
file server and process supervisor that **duplicate `docker-compose.test.yml` +
`docker/nginx.conf`** as a second, independent definition of the E2E stack — which CI does
not use (CI still runs E2E through Docker) and which will drift from the Docker path
silently. Its stated motivation is "environments without Docker (e.g. cloud agents)",
i.e. the submitter's tooling, not this project's need; your documented path is
`docker compose up -d`. It also already ships a bug: no `server.on('error')` handler on
`APP_PORT` 7300 — the frontend dev-server port — so with `yarn start` running it dies on
an unhandled `EADDRINUSE` *after* spawning the backend, orphaning a `meerkat` process on
8081 and leaking `/tmp/meerkat-e2e-*`. Reproduced. Don't take on a parallel stack
definition to fix a problem you don't have.

The root `Makefile` in the same PR is a separate question and genuinely optional. If you
like one entry point, keep a minimal version (`install`, `lint`, `typecheck`, `test`,
`precommit`) and drop `format`/`format-check` (they only exist to serve the Prettier gate)
and `test-e2e` (it drives the rejected script). Be aware it now duplicates the command
list in CLAUDE.md, so it is one more thing to keep in sync.

**219 — "feat(i18n): translate extended relationship types".** The title describes an i18n
change. What it actually does is **add 43 relationship types that do not exist on main** —
`Grandparent`, `Godchild`, `Mother-in-law`, `Mentee`, `Ward`, `Caregiver`, `Kin`, `Muse`,
`Crush`, `Date` — taking the vocabulary from ~10 to ~53, then translating all of them into
five locales and adding 169 lines to `vcard_mapper.go` to map them onto vCard `RELATED`
types. The relationship vocabulary of your CRM is a product decision that belongs to you,
and this arrives as a 482-line fait accompli under a maintenance-shaped title. Reject as
submitted. If you *do* want a longer list, choose it yourself — the i18n and mapper work
is then straightforward, and you avoid inheriting someone else's taste in "Muse".

---

## Resulting merge order

The rejections collapse most of the conflict graph: dropping 219 removes the 219↔224
conflict, and dropping Prettier removes the `package.json`/`yarn.lock` half of 222↔249.

1. **253**, then **244**, **241**, then 252 / 257 / 258 / 242 / 243 / 245 — dependabot.
2. **247**, trimmed (gosec dropped).
3. **249**, trimmed (ESLint + tsc + trigger fix only).
4. **259**.
5. **251**, once the `VERSION:2.1` gate is added.
6. **224**, once the version picker is dropped.
7. **222**.
8. **233**, then **227** — both rebase onto 222; conflicts are additive.

Remaining real conflicts after the trims: 222↔247 (`validation.go`), 222↔233
(`DateFormatProvider.tsx` + locales), 222↔227 (`ContactDetailPage.tsx`,
`ContactInformation.tsx` + locales), 251↔241 (`go.mod`). All mechanical.
