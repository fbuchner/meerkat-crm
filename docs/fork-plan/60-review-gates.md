# 60 — Review gates, model allocation, locked artifacts

The architecture is delegatable, but two artifacts are load-bearing "truth" that a coding agent can get
wrong *invisibly*. This doc closes that gap. Read alongside `00-overview.md`.

## 60.1 Locked artifacts (a coding agent MUST NOT author or alter these)

| Artifact | Authored by | Rule |
|---|---|---|
| `docs/fork-plan/20-correspondence.md` + the materialized `correspondence.tsv` | strong model / human, from `docs/specs/rfc9555-correspondence.md` | Coding agents consume it read-only. Missing mapping → escalate, never invent. |
| `docs/fork-plan/golden-fixtures/*` | copied verbatim from RFCs | Coding agents copy into `backend/internal/rfctest/fixtures/` unchanged. Never edit a value. |
| `docs/specs/*` | transcribed from RFCs | Reference only. |

**Why this matters:** every adapter and its directional tests derive from the correspondence table. If
a row is wrong, all of them are *consistently, confidently* wrong — and green. And because each adapter
agent writes its own code and tests, self-authored tests cannot catch a shared misconception. The
golden fixtures (external RFC ground truth) are the only thing that can. Therefore: **green tests ≠
correct.** Acceptance requires the golden-fixture tests to pass, and those fixtures must stay verbatim.

## 60.2 Model allocation (recommended)

| Work | Model |
|---|---|
| Correspondence table content (`20`) + `correspondence.tsv` | **Opus / human** |
| Golden-fixture curation & the `SOURCES.md` provenance | **Opus / human** |
| WP-10, WP-20 loader, WP-30a, WP-40a, WP-60 (mechanical) | **Sonnet** |
| WP-30b, WP-40b, WP-50 (adapter code + directional tests) | **Sonnet**, after the table + fixtures are locked |
| Per-WP review gate (60.3) | **Opus / human** |

## 60.3 Per-WP review checklist (run before merging each WP)

**Every WP**
- [ ] Only the files the WP lists were created/edited; nothing else touched (`git status`).
- [ ] `go build ./... && go vet ./<pkg>/... && go test ./<pkg>/...` green.
- [ ] No mapping appears in code that is absent from `20-correspondence.md` (grep property names).
- [ ] No `error` returned for unmappable data; a `Diagnostic` is emitted instead (0.5).

**Adapter WPs (30b / 40b / 50) — the high-risk gate**
- [ ] Every assertion's *expected value* traces to a correspondence row or a golden fixture — not to a
      value the agent chose. Spot-check 5 assertions against `docs/specs/`.
- [ ] Golden-fixture tests exist and pass: each fixture in `golden-fixtures/` is imported and/or
      exported and checked against RFC-verbatim bytes/fields. These are the authoritative checks.
- [ ] Fixtures are byte-identical to `docs/fork-plan/golden-fixtures/` (no silent edits to make a test
      pass). Diff them.
- [ ] `PROP-ID`/`ID` round-trips (import→export keeps the key).
- [ ] Passthrough is a true inverse: a fixture carrying an unknown property re-emits it unchanged; the
      de-dup guard (20.5) prevents a mapped property from also appearing via passthrough.
- [ ] vCard3 degradation: each `20.6` concept present in a source Record yields a `Diagnostic{warn}` and
      is absent from output but still present in the neutral Record.
- [ ] Coverage meta-test (`40 §40.5`) is green: every for-this-format concept has an import and an
      export test.

**Correspondence / fixtures (if regenerated)**
- [ ] Every row cites a section of `docs/specs/rfc9555-correspondence.md`.
- [ ] Reflection test: every `neutral_path` resolves on `contactmodel.Record`.
- [ ] No duplicate `concept_id`; every `transform` is implemented.

## 60.4 Escalation protocol (for the coding agent)

If, while implementing, an agent finds: a concept with no correspondence row; an enum value not in the
registry lists; a fixture that won't parse; or two plausible mappings — it **stops**, writes a
`ESCALATION.md` note in its package describing the ambiguity with the RFC/section it consulted, and
does **not** proceed on that concept. A reviewer resolves it into `20`/`docs/specs` (the locked
sources), then the agent resumes. This is the mechanism that keeps "don't invent a mapping" from
depending on the agent's self-restraint.
