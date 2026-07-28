# Implementation Plan — RFC 9553 / 9554 / 9555-native Contacts (Meerkat fork)

> **What this is.** A decomposed, execution-ready implementation plan derived from the feasibility
> assessment (`/home/drew/.claude/plans/what-would-the-difficulty-spicy-brooks.md`). It is written to
> be handed, section by section, to *code-producing sub-agents that make no architectural decisions*.
> Every judgment call is pre-made here; a sub-agent's job is transcription + wiring + making the named
> tests green.
>
> **Deliverable of THIS plan:** the code, in a committable state, on a true fork of Meerkat on the
> user's own GitHub (later rebranded — see `50-integration-and-rebrand.md`).

## 0.1 Documents in this plan

| File | Contents | Primary consumer |
|---|---|---|
| `00-overview.md` (this) | Goals, layout, conventions, the **work-package DAG**, build order | Orchestrator |
| `10-neutral-model.md` | The neutral internal model package — every Go type & field | WP-10 agent |
| `20-correspondence.md` | **The oracle**: neutral ↔ JSContact ↔ vCard4 ↔ vCard3 mapping table + its materialized `testdata` form | WP-20 agent; all adapter/test agents |
| `30-adapters.md` | `jscontact`, `vcard4`, `vcard3` packages — types, constants, signatures, per-format rules | WP-30/40/50 agents |
| `40-testing.md` | Directional test conventions, fixtures, per-field templates, acceptance criteria | every adapter agent |
| `50-integration-and-rebrand.md` | P1–P4 (model swap, migration, API, frontend, CardDAV) + rebrand, file-level | later-phase agents |
| `60-review-gates.md` | Locked artifacts, model allocation, per-WP reviewer checklist, escalation protocol | reviewer / orchestrator |
| `70-environment.md` | **Operational facts**: no local `go` toolchain (Docker build/test workflow), branch/remote target, CI status | any agent, read first |
| `../specs/*` | Curated RFC 6350/2426/9553/9554/9555 excerpts — **ground truth** behind the oracle | any agent, read-only |
| `golden-fixtures/*` | Verbatim RFC example cards (incl. vCard 3.0 and 4.0 baselines) — the external test oracle | WP-60 (copies to `backend/internal/rfctest/fixtures/`) |

## 0.2 Goal & scope (restated, binding)

- **Hub-and-spoke.** One **neutral internal superset model** (the hub). Independent **format adapters**
  (`jscontact`, `vcard4`, `vcard3`), each doing import **and** export. No adapter depends on another.
- **Conversion is emergent**, never coded: import format → neutral → export another. **RFC 9555 is
  never executed**; it is the *authoring oracle* that fixes what each format's output must be (encoded
  once in `20-correspondence.md`).
- **Tests are directional & per-format**: import (`format→neutral`) asserts correct neutral properties;
  export (`neutral→format`) asserts correct format properties. Expected values come from the
  correspondence table. Plus a few spot-check round-trips. **No cross-format runtime comparison.**
- **Full field coverage** of both IANA registries (enumerated in `10`/`20`).
- **Degradation is graceful, not strict** (see 0.5).

## 0.3 Repository layout (new packages)

Go module stays `meerkat` for P0 (renamed in the rebrand phase). All new backend packages live under
`backend/`. **Leaf-first dependency order — this is load-bearing:**

```
backend/
  contactmodel/        WP-10  neutral model types (NO logic, NO deps on other new pkgs)
    model.go
    envelope.go        CRM-only sibling data
    passthrough.go     unmapped-property stores
    projection.go      derived/projected scalar columns
  correspondence/      WP-20  the oracle, as data + typed accessor
    table.go           parses testdata/correspondence.tsv into []Row
    correspondence_test.go
    testdata/correspondence.tsv
  jscontact/           WP-30  RFC 9553
    types.go           Card + sub-objects (mirrors registry)
    codec.go           JSON (de)serialize: @type/@version, Id-map collections
    adapter.go         ToNeutral / FromNeutral
    *_test.go
  vcard4/              WP-40  vCard 4.0 + RFC 9554/6474/6715/8605
    consts.go          property + parameter name constants
    components.go      structured-value codec (salvaged from carddav/vcard_mapper.go)
    adapter.go         ToNeutral / FromNeutral
    *_test.go
  vcard3/              WP-50  legacy vCard 3.0 (separate, tuned)
    consts.go
    adapter.go
    *_test.go
  internal/rfctest/    WP-60  shared test helpers + fixtures loader
    helpers.go
    fixtures/…
```

`emersion/go-vcard` (already a dependency) is reused by `vcard4` and `vcard3` as the low-level
line/property codec — **do not fork it**; it accepts unknown properties via its generic
`map[string][]*vcard.Field` model.

## 0.4 Adapter interface (every format implements this exactly)

Defined once in `contactmodel` (so adapters depend only on `contactmodel`):

```go
// backend/contactmodel/adapter.go
package contactmodel

// Diagnostic is a non-fatal data-handling event (see degradation policy).
type Diagnostic struct {
    Severity string // "warn" | "info"
    Concept  string // correspondence concept_id, or "" 
    Message  string
}

// Importer parses one serialized format into the neutral model.
// It MUST NOT return an error for unmappable/unknown data — it preserves it
// (passthrough) and appends a Diagnostic. errors are reserved for malformed input
// (bytes that are not valid instances of the format at all).
type Importer interface {
    Import(raw []byte) (*Record, []Diagnostic, error)
}

// Exporter renders the neutral model into one serialized format.
// Same rule: never error on a field that has no home in the target format —
// drop-with-warning or passthrough, and append a Diagnostic.
type Exporter interface {
    Export(r *Record) ([]byte, []Diagnostic, error)
}
```

`Record` = neutral contact (`card` payload) + CRM envelope + passthrough stores (see `10`).

## 0.5 Degradation policy (binding, applies to every adapter)

Two tiers of "loss", never a hard failure of import/export:

1. **Mappable data that fails to land** (e.g. a phone number) → **defect**. Caught by a red directional
   test at dev time; at runtime emit a `Diagnostic{Severity:"warn"}`. Operation still completes.
2. **Genuinely unmappable/unknown data** (property with no neutral or no target-format home) →
   **preserve, don't reject**:
   - Unknown *vCard* props on import → `Record.Passthrough.VCard []JCardProp` (RFC 9555 `vCardProps`
     shape). Re-emitted verbatim on vCard export.
   - Unknown *JSContact* props on import → `Record.Passthrough.JSContact map[string]json.RawMessage`.
     Re-emitted on JSContact export; on vCard export they become `JSPROP` (RFC 9555) with a `JSPTR`.
   - A neutral field with **no** target-format home (e.g. `GRAMGENDER` when exporting vCard 3.0) →
     `Diagnostic{Severity:"warn"}`, value dropped from that serialization only (still present in the
     neutral model and in other formats).

`error` is returned **only** for input that is not a valid instance of the format at all
(unparseable JSON, malformed vCard framing).

## 0.6 Conventions (all packages)

- Go ≥ the repo's `go.mod` version. `gofmt`/`go vet` clean. Std `testing` only (no testify in new pkgs;
  match the existing `vcard_mapper_test.go` idiom).
- One exported `Import`/`Export` per adapter; everything else unexported.
- No package under `backend/{contactmodel,correspondence,jscontact,vcard4,vcard3}` may import `models`,
  `gorm`, `gin`, or any other new adapter package. They are pure and offline-testable.
- Time/dates stored in the neutral model as RFC 3339 / partial-date structs (see `10`), never as Go
  `time.Time` in the wire types.
- Every mapping decision is justified by a `concept_id` row in `20-correspondence.md`; if a sub-agent
  needs a mapping not in the table, it STOPS and flags it — it does not invent one.

## 0.7 Work-package DAG (hand each row to one sub-agent)

Acceptance = "package compiles, `go vet` clean, and the named tests are green", plus the WP-specific
checks. A sub-agent creates **only** the files its WP lists and modifies nothing else.

| WP | Title | Creates | Depends on | Acceptance |
|---|---|---|---|---|
| **WP-10** | Neutral model types | `backend/contactmodel/*.go` (no adapter logic) | — | Compiles; `go test ./backend/contactmodel/...` (constructor/round-trip-of-struct tests) green |
| **WP-20** | Correspondence oracle | `backend/correspondence/{table.go,correspondence_test.go,testdata/correspondence.tsv}` | WP-10 | TSV parses; every row's `neutral_path` resolves to a real field on `contactmodel.Record` (test enforces); no duplicate `concept_id` |
| **WP-30a** | JSContact types + codec | `backend/jscontact/{types.go,codec.go}` + codec tests | WP-10 | Round-trips every fixture in `jscontact/testdata/*.json` byte-stable under canonical marshal |
| **WP-30b** | JSContact adapter | `backend/jscontact/adapter.go` + directional tests | WP-30a, WP-20 | `40-testing.md` import+export suites green |
| **WP-40a** | vCard4 consts + component codec | `backend/vcard4/{consts.go,components.go}` + unit tests | WP-10 | Component escape/split round-trip tests green |
| **WP-40b** | vCard4 adapter | `backend/vcard4/adapter.go` + directional tests | WP-40a, WP-20 | import+export suites green |
| **WP-50** | vCard3 adapter | `backend/vcard3/*.go` + directional tests (incl. degradation warnings) | WP-40a, WP-20 | import+export suites green; degraded-field warnings asserted |
| **WP-60** | Fixtures + helpers | `backend/internal/rfctest/*`, all `testdata/` corpora | WP-10 | Loader compiles; fixtures parse |
| **WP-70…** | Integration P1–P4 + rebrand | see `50-integration-and-rebrand.md` | WP-10..60 green | per that doc |

**Build order:** WP-10 → WP-20 & WP-60 (parallel) → WP-30a, WP-40a (parallel) → WP-30b, WP-40b, WP-50
(parallel) → WP-70+. **P0 gate:** everything WP-10..60 green with **zero** changes to existing app code.

## 0.8 How a sub-agent should operate (paste into each sub-agent prompt)

1. Read `70-environment.md` first (how to actually build/test in this sandbox, and where the work
   lands) — then this file (0.3–0.6), your WP row (0.7), and the section(s) of `10`/`20`/`30`/`40` your
   WP names. Do not read the rest.
2. Create exactly the files your WP lists. Do not edit files outside your package.
3. Use the correspondence table (`20`) as the *only* source of mapping truth. If a needed mapping is
   missing or ambiguous, stop and report — never invent one.
4. Follow the degradation policy (0.5) literally: no hard errors for unmappable data.
5. Finish only when the Docker build command in `70-environment.md` §70.1, scoped to your package, is
   green (`go build ./... && go vet ./<your-pkg>/... && go test ./<your-pkg>/...` inside the container —
   there is no bare `go` in this environment).
6. Output: the diff and the `go test` results.

## 0.9 Verification of the whole P0

```
cd backend
go build ./...
go vet ./contactmodel/... ./correspondence/... ./jscontact/... ./vcard4/... ./vcard3/...
go test  ./contactmodel/... ./correspondence/... ./jscontact/... ./vcard4/... ./vcard3/...
```
All green, and `git status` shows **no** modifications under `backend/carddav`, `backend/models`,
`backend/controllers`, or `frontend/` — proving P0 is an additive, isolated core.

**Authoritative check (not just "tests green"):** the golden-fixture tests must pass. Because each
adapter agent writes its own code *and* tests, self-authored green tests can share a misconception; the
verbatim RFC fixtures in `golden-fixtures/` are the external ground truth. See `60-review-gates.md` —
the correspondence table and the fixtures are **locked** artifacts a coding agent must not alter.
