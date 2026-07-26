# Spec references (ground truth for the contacts rework)

These are **curated, load-bearing excerpts** transcribed from the RFCs (verbatim where it matters:
mapping tables, value syntax, example cards). They exist so a coding agent cites real spec text instead
of guessing. For anything not covered here, consult the canonical RFC.

| File | Source | Use |
|---|---|---|
| `rfc9555-correspondence.md` | RFC 9555 (JSContact↔vCard conversion) | **The oracle authority** behind `docs/fork-plan/20-correspondence.md` |
| `rfc9553-model.md` | RFC 9553 (JSContact) | Object shapes, value formats, Id-map classification |
| `rfc9554-vcard-extensions.md` | RFC 9554 (vCard extensions) | New properties/parameters + N/ADR expansion, verbatim examples |
| `rfc6350-baseline.md` | RFC 6350 (vCard 4.0 base) | Baseline grammar/cardinality `vcard4` builds on; states the salvage-not-reimplement decision |
| `rfc2426-v3-baseline.md` | RFC 2426 (vCard 3.0) | Baseline grammar `vcard3` builds on; confirms **no GENDER in 3.0**; the LABEL property-vs-parameter difference |
| `rfc6474-birthplace-deathplace.md` | RFC 6474 (BIRTHPLACE/DEATHPLACE/DEATHDATE) | Confirms value types/cardinality/params; documents the `Anniversary.Place` (`*Address`) vs. BIRTHPLACE/DEATHPLACE (TEXT-or-URI) type mismatch and its required lossy transform |
| `rfc6715-personalinfo.md` | RFC 6715 (EXPERTISE/HOBBY/INTEREST/ORG-DIRECTORY) | Confirms value types, `LEVEL`/`INDEX` param enums/semantics; documents Errata 3341 (non-issue for us) |
| `rfc8605-contacturi.md` | RFC 8605 (CONTACT-URI) | Confirms value type/cardinality/`PREF`; documents the mailto/http/https "MUST provide" constraint as unenforced per degradation policy |

Canonical sources (fetch full text if needed — all nine have now been transcribed into this folder):
- RFC 9553 — https://www.rfc-editor.org/rfc/rfc9553.txt — see `rfc9553-model.md`
- RFC 9554 — https://www.rfc-editor.org/rfc/rfc9554.txt — see `rfc9554-vcard-extensions.md`
- RFC 9555 — https://www.rfc-editor.org/rfc/rfc9555.txt — see `rfc9555-correspondence.md`
- RFC 6350 (vCard 4.0) — https://www.rfc-editor.org/rfc/rfc6350.txt — see `rfc6350-baseline.md`
- RFC 2426 (vCard 3.0) — https://www.rfc-editor.org/rfc/rfc2426.txt — see `rfc2426-v3-baseline.md`
- RFC 6474 (BIRTHPLACE/DEATHPLACE/DEATHDATE) — https://www.rfc-editor.org/rfc/rfc6474.txt — see `rfc6474-birthplace-deathplace.md`
- RFC 6715 (EXPERTISE/HOBBY/INTEREST/ORG-DIRECTORY) — https://www.rfc-editor.org/rfc/rfc6715.txt — see `rfc6715-personalinfo.md`
- RFC 8605 (CONTACT-URI) — https://www.rfc-editor.org/rfc/rfc8605.txt — see `rfc8605-contacturi.md`
- Errata 3341 (against RFC 6715) — https://www.rfc-editor.org/errata/eid3341 — verified non-issue, see `rfc6715-personalinfo.md`
- RFC 7095 (jCard, for `JCardProp`) — https://www.rfc-editor.org/rfc/rfc7095.txt — **not yet
  transcribed**; only needed if a `JCardProp`/`vCardProps` escape-hatch bug requires checking the exact
  jCard tuple encoding. Fetch on demand.

**Rule for implementers:** the mapping in `20-correspondence.md` is LOCKED against
`rfc9555-correspondence.md`. Do not invent or alter a mapping. If a needed mapping is missing,
escalate to a reviewer (see `docs/fork-plan/60-review-gates.md`).

## Why RFCs, not just the IANA registries

The two IANA registries (vcard-elements, jscontact) are the authority for **coverage** — "does this
property/parameter/type/value exist, and which RFC defines it." That's exactly how they're used in
`10-neutral-model.md` §10.7 and `30-adapters.md`'s `consts.go` name lists: transcribed directly as
completeness checklists. They are **not** sufficient for **behavior** — a registry entry is a one-line
index (e.g. `GRAMGENDER — RFC 9554`); it carries no value syntax, no cardinality, no examples, and
critically **no cross-format mapping**. IANA does not publish a JSContact↔vCard correspondence
registry at all — that mapping exists only as RFC 9555 prose, which is why `20-correspondence.md` is
built from `rfc9555-correspondence.md`, not from the registries.

**Maintenance caveat:** the registries are living documents (updated as new properties are registered
under future RFCs); this `docs/specs/` folder is a frozen snapshot of eight RFCs (plus one erratum)
fetched 2026-07-24/25. If a new vCard/JSContact element is registered later, the registry will reflect
it before this folder does — re-check the registries periodically and add a new spec excerpt if
coverage drifts.

**Full vCard coverage confirmation:** the IANA vcard-elements registry cites exactly six RFCs as the
source of every registered vCard element: 6350 (base), 2426 (legacy 3.0), 6474, 6715, 8605, and 9554.
All six are now transcribed above, plus 9553/9555 for the JSContact side — this folder has no
remaining citation gaps against either registry.
