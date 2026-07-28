# 40 — Directional tests, fixtures, acceptance  ·  applies to WP-30b / WP-40b / WP-50 / WP-60

Tests are **directional and per-format**. Expected values come from the correspondence table (WP-20).
There is **no** test that loads all three formats and cross-compares. Std `testing` only.

## 40.1 The four test kinds per adapter

For each format package `P` ∈ {jscontact, vcard4, vcard3}:

1. **Import field tests** — `P → neutral`. Given a format fixture, assert specific
   `contactmodel.Record` fields hold the expected values. One focused test per concept group.
2. **Export field tests** — `neutral → P`. Given a curated `Record`, assert the emitted bytes contain
   the expected property/param/value **for that format** (expected value read from the correspondence
   row's `vN_prop`/`vN_params`/`transform`). e.g. the phone export test in `vcard4` asserts a `TEL`
   line; in `jscontact` asserts `phones/{id}/number`; in `vcard3` asserts `TEL` — three separate tests,
   each citing the same `phone` concept row. (No single `outputPhoneTest` across formats.)
3. **Degradation tests** (vcard3 especially) — assert a `Diagnostic{warn}` is returned for each 20.6
   no-home concept and that the field is absent from output but still present in the source Record.
4. **Spot-check round-trips** — a *handful* (≈5) of `P → neutral → P` and `neutral → P → neutral` on
   whole fixtures, asserting equality of the neutral projection / of key fields. Confidence only.

## 40.2 Test naming & layout

```
backend/<pkg>/import_<group>_test.go   // TestImport_Phones, TestImport_Addresses, …
backend/<pkg>/export_<group>_test.go   // TestExport_Phones, …
backend/<pkg>/degrade_test.go          // vcard3 only
backend/<pkg>/roundtrip_test.go        // spot checks
backend/<pkg>/testdata/*.{json,vcf}    // fixtures
```

## 40.3 Per-concept expectation source (how an agent writes an export assertion)

Pseudo for a `vcard4` export test of concept `phone`:
```
row := correspondence.ByConcept()["phone"]          // v4_prop=TEL, v4_params=PREF;TYPE, transform=identity
rec := recordWith(Phone{Number:"+15551234567", Features:[]string{"cell"}, Pref:ptr(1)})
out,_,_ := vcard4.Adapter{}.Export(rec)
assertVCardLine(out, "TEL", wantParams{"PREF":"1","TYPE":"cell"}, wantValue:"+15551234567")
```
The `feat2type`/`pref` transforms define `cell`/`1`. The agent copies expected tokens from the
transform spec (20.4), never invents them.

## 40.4 Fixtures & golden corpus (WP-60)

`backend/internal/rfctest/`:
- `helpers.go` — `LoadFixture(name string) []byte`; `AssertVCardLine(...)`; `AssertJSONPointer(...)`;
  `NeutralFromJSON(...)` builders.
- `fixtures/`:
  - **Spec examples** — the worked example cards from RFC 9553 §Appendix, RFC 6350 §6/§7,
    RFC 9554 examples, and RFC 9555 mapping examples, saved as paired `NNN.jscontact.json` /
    `NNN.v4.vcf` / `NNN.v3.vcf` where the RFC provides them. These are the highest-authority fixtures.
  - **Real-world** — a few exported Apple Contacts and Google `.vcf` files (3.0-heavy, `X-ABLabel`,
    grouped items) to exercise legacy edge cases.
  - **Per-concept minimal** — one tiny fixture per concept group for the focused import tests.

Fixture provenance is recorded in `fixtures/SOURCES.md` (which RFC section / which client each came
from). Do not hand-fabricate values that contradict a spec example.

## 40.5 Coverage gate

A meta-test `coverage_test.go` (in each adapter package) iterates `correspondence.Load()` and asserts
that **every concept with a non-`-` prop for that format** has at least one import test and one export
test referencing it (enforced by a registry of `t.Run` subtest names, or a simple concept→testname
map the agent fills in). This makes "full field coverage" mechanically checkable, not aspirational.

## 40.6 Acceptance (P0 gate, from 0.9)

```
cd backend
go build ./...
go vet  ./contactmodel/... ./correspondence/... ./jscontact/... ./vcard4/... ./vcard3/... ./internal/rfctest/...
go test ./contactmodel/... ./correspondence/... ./jscontact/... ./vcard4/... ./vcard3/...
```
All green; `git status` shows no edits under `carddav/`, `models/`, `controllers/`, `frontend/`.
