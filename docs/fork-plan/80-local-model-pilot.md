# 80 — Local-model code-generation pilot (calibration experiment)

> **Status: deferred.** Originally scoped to run as a P2 calibration slice; **decided to defer until
> mobile app development begins instead** — a fresh client codebase (Swift/Kotlin/Dart, framework still
> undecided per `50-integration-and-rebrand.md` WP-71) gives a cleaner transition point: new code, no
> existing backend conventions to match, and naturally more small, self-contained, view-model/mapper-style
> units than anything left in the backend plan. Not tied to any particular WP's commit anymore — pick this
> doc back up when mobile work starts. The pipeline/rules below (§80.1–80.2, §80.4–80.7) are
> framework-agnostic and still the intended process; only §80.3's specific pilot slice is stale (it named
> a backend P2 slice) and needs re-choosing from the actual mobile codebase once that framework is picked.

## 80.1 The bet, and why it needs calibrating

The hypothesis: a strong model (Claude) does the expensive cognitive work — spec, stub design, test
authoring, final review — and a cheap local model fills in function bodies, saving tokens. This works
**only** when the unit of work handed to the local model is *local-reasoning-complete*: everything
needed to write the body fits in its immediate context (signature + precise doc comment + the minimal
types it touches + the already-written test), and correctness is a **deterministic** pass/fail (the
test), not an LLM judgment. The pilot measures whether that condition can actually be met in practice
and what the local model's first-pass hit rate is.

## 80.2 Pipeline (five stages, fixed roles)

| Stage | Actor | Produces | Hard rule |
|---|---|---|---|
| 1. Spec | **Claude Opus** | A precise spec for the pilot slice: exact files, function signatures, behavior per function, and the exact expected values each test will assert. | No implementation, no stubs — just the contract. |
| 2. Stubs + **complete** tests | **Claude Sonnet** | Placeholder `.go` files: each function is a signature + a doc-comment stating its exact required behavior + a `panic("TODO: <what>")` body. **Tests are written in full, not stubbed** — they compile and currently FAIL (red). | The local model must NEVER author a test. A model that writes both impl and its own test passes its own blind spots green (this bit us repeatedly in P0). Tests are the gate; the gate cannot be written by the thing it's gating. |
| 3. Fill bodies | **deepseek-coder-v2:16b** | Function bodies replacing the `panic("TODO")` stubs, one function at a time. | Sees only: the one stub + its doc comment + the minimal types it references + its test. Not the whole codebase. |
| 4. Review (optional, non-gating) | **deepseek-r1:14b** | Comments flagging issues tests can't catch (swallowed errors, obvious O(n²)). | **Not a gate.** The deterministic gate is `go test` (stage 5), not this. Keep only for bonus signal; drop if it adds noise. |
| 5. Verify + evaluate | **Claude Sonnet** | Runs `go test` in the Docker container; iterates with deepseek on failures; writes the evaluation (hit rate, where it failed, was it worth it). | If Sonnet spends more tokens debugging deepseek's misses than writing the code directly would have cost, that's a **failed** experiment — record it honestly. |

## 80.3 The pilot slice — TO BE RE-CHOSEN at mobile kickoff (this section is stale)

The two backend units originally proposed here (`ContactSummary`/`SummaryFromProjection` mapping,
`SniffVCardVersion`) are no longer the plan — deferred per the status note above. When mobile work
starts, re-derive a slice from the *actual* client codebase using the same selection criteria that chose
those two: **pure-function, narrow-signature, no I/O/network/DB, every output trivially pinnable by a
test written before the local model sees the stub.** Good candidate shapes in a typical mobile client
(concrete choice depends on the framework picked): a view-model/DTO mapper (wire JSON → UI-display
struct, the mobile analog of `SummaryFromProjection`), a local cache row ↔ model mapper, a small pure
formatting/validation helper. Avoid anything touching the platform's persistence layer, networking stack,
or UI framework lifecycle on the local model's first pass — those need the same "stays with Claude"
treatment P1 got here. Do not restart this pilot against backend code later just because this doc already
exists — the whole point of deferring was that mobile is the cleaner entry point.

Both are ≤~30-line bodies with all context inlinable into a single ≤8k-token prompt.

## 80.4 Stub-authoring rules (Stage 2 — load-bearing; this is what makes or breaks it)

- **One function per unit.** deepseek gets exactly one stub's worth of context at a time.
- **Inline everything it needs.** If the body needs a type def (`Projection`, `ContactSummary`), paste
  that type into the stub's doc comment or keep it in the same short file. deepseek must not need to go
  read another package.
- **Doc comment states behavior exactly**, including edge cases the test checks ("on missing VERSION
  return a non-nil error; do not default to 4.0").
- **Tests are complete and currently red.** Sonnet runs them once to confirm they fail against the
  `panic("TODO")` stub (proves the test actually exercises the function), before handing to deepseek.
- **No cross-unit dependencies** within the pilot — the two units don't call each other.

## 80.5 Local runtime notes

- Models via `ollama` (`ollama run deepseek-coder-v2:16b`, `deepseek-r1:14b`). deepseek-coder-v2:16b is
  MoE (~2.4B active) — comfortable on 16GB; deepseek-r1:14b is dense/heavier — expect tighter headroom.
- **Usable context after weights (quantized, 16GB): realistically ~8–16k tokens.** Every stub prompt
  must fit well inside that with room for the model's output. This is the binding constraint on 80.4.
- Go verification is unchanged: the Docker `golang:1.25` container in `70-environment.md` §70.1. deepseek
  never runs Go itself; Sonnet runs the gate.

## 80.6 Success criteria (what we actually measure)

Record, at the end:
1. **First-pass hit rate**: of the stub functions, what fraction passed their tests on deepseek's first
   fill, and what fraction after one retry with the failing test output fed back.
2. **Claude token cost of the run** (Opus spec + Sonnet stub/test/verify/iterate) vs. an honest estimate
   of Sonnet writing the same slice directly. Net savings or net loss?
3. **Failure taxonomy**: when deepseek missed, was it context-starvation (needed something not inlined),
   capability (couldn't reason the logic), or spec ambiguity (Sonnet's stub under-specified)?

## 80.7 Decision gate (what a result means)

- **Pass** (high hit rate, real net token savings, failures were fixable by better inlining): consider
  extending the pattern to the *mechanical, pure-function* portions of later WPs — never the migration
  orchestration or round-trip-test design, which stay with Claude.
- **Fail** (low hit rate, or Sonnet-debug cost exceeds direct-write cost): abandon the local-model tier;
  the experiment paid for itself by learning this cheaply on code that couldn't corrupt contacts. Record
  the finding and move on.

Either outcome is a success *for the experiment* — the point is to learn the hit rate at low stakes, not
to prove the approach works.
