# Golden fixtures — provenance

Every fixture here is copied VERBATIM from an RFC. They are the external ground truth for the
directional tests (import: file → expected neutral fields; export: expected neutral → these bytes).
WP-60 copies this directory into `backend/internal/rfctest/fixtures/`. Do NOT hand-edit values.

| Fixture | Source | Concept(s) |
|---|---|---|
| johndoe.jscontact.json | RFC 9553 Fig 6 (§2) | uid, kind, name.components, isOrdered |
| title-role.v4.vcf / .jscontact.json | RFC 9555 §2.9.6 | titles(kind), organizationId via GROUP, org |
| pronouns.v4.vcf | RFC 9554 §3.4 | speakToAs.pronouns (LANGUAGE, PREF) |
| gramgender.v4.vcf | RFC 9554 §3.2 | speakToAs.grammaticalGender |
| socialprofile.v4.vcf | RFC 9554 §3.5 | onlineServices (SERVICE-TYPE, USERNAME, VALUE=text) |
| created.v4.vcf | RFC 9554 §3.1 | card.created |
| derived-fn.v4.vcf | RFC 9554 §4.4 | name (N) + FN DERIVED |
| note-author.v4.vcf | RFC 9554 §4.1–4.3 | notes (AUTHOR, AUTHOR-NAME, CREATED) |
| n-expanded.v4.vcf | RFC 9554 §2.2 | name 7-component N |
| adr-expanded.v4.vcf | RFC 9554 §2.1 | addresses 18-component ADR + GEO |
| phonetic-n.v4.vcf | RFC 9554 §4.6 | name phonetic (ALTID, PHONETIC, SCRIPT, LANGUAGE) |
| rfc6350-baseline.v4.vcf | RFC 6350 §7.2.1 | minimal valid 4.0 card; FN, N, EMAIL, UID, PID/CLIENTPIDMAP passthrough |
| rfc2426-baseline.v3.vcf | RFC 2426 §7 | **vCard 3.0** baseline: FN, ORG, multi-ADR/TEL/EMAIL with comma-joined TYPE lists, URL |

## Fixture-coverage fallback rule (binding)

Not every mapped concept has an RFC-provided worked example — most RFCs don't bother illustrating
unchanged RFC 6350 baseline behavior (e.g. a second plain `EMAIL` line). Rule:

- **Concepts introduced or changed by RFC 9553/9554/9555** (pronouns, grammatical gender,
  social profiles, created, derived-FN, expanded N/ADR, phonetic, note author/created, title/role/org
  grouping, the escape hatches) → fixture **MUST** be RFC-verbatim. All such concepts already have one
  above.
- **Concepts that are unchanged RFC 6350/2426 baseline** (plain repeatable EMAIL/TEL/URL, ORG without
  units, plain ADR without new components, CATEGORIES, PHOTO, KEY, etc.) → a **hand-authored** fixture
  is acceptable, since there is no novel semantic to get wrong and no RFC example to be unfaithful to.
  Base such fixtures on the two baseline cards above (`rfc6350-baseline.v4.vcf` /
  `rfc2426-baseline.v3.vcf`) by extension, not invention from scratch.
- Either way, a fixture is added to this directory (and this table) **before** the adapter test that
  consumes it is written — never inline test-literal strings for RFC-sourced concepts.
