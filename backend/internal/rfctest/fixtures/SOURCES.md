# Fixture provenance (`backend/internal/rfctest/fixtures/`)

This directory has two layers:

1. **Golden fixtures** (locked) — copied byte-for-byte from `docs/fork-plan/golden-fixtures/` per
   `docs/fork-plan/60-review-gates.md`. Do NOT hand-edit these. The table below is copied verbatim from
   that directory's own `SOURCES.md`.
2. **Per-concept minimal fixtures** (this WP, WP-60) — small hand-authored/spec-anchored fixtures added
   to cover `docs/fork-plan/20-correspondence.md` §20.3 concepts that have no golden (RFC-worked-example)
   fixture, so later WPs (30b/40b/50) have a focused single-concept fixture to write import tests
   against. See the second table below.

## 1. Golden fixtures (verbatim from RFC text — locked, do not edit)

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

### Fixture-coverage fallback rule (binding, inherited from `docs/fork-plan/golden-fixtures/SOURCES.md`)

Not every mapped concept has an RFC-provided worked example — most RFCs don't bother illustrating
unchanged RFC 6350 baseline behavior (e.g. a second plain `EMAIL` line). Rule:

- **Concepts introduced or changed by RFC 9553/9554/9555** (pronouns, grammatical gender,
  social profiles, created, derived-FN, expanded N/ADR, phonetic, note author/created, title/role/org
  grouping, the escape hatches) → fixture **MUST** be RFC-verbatim. All such concepts already have one
  above.
- **Concepts that are unchanged RFC 6350/2426 baseline** (plain repeatable EMAIL/TEL/URL, ORG without
  units, plain ADR without new components, CATEGORIES, PHOTO, KEY, etc.) → a **hand-authored** fixture
  is acceptable, since there is no novel semantic to get wrong and no RFC example to be unfaithful to.
- Either way, a fixture is added to this directory (and this table) **before** the adapter test that
  consumes it is written — never inline test-literal strings for RFC-sourced concepts.

## 2. Per-concept minimal fixtures (added by WP-60)

Every row below is a concept from `docs/fork-plan/20-correspondence.md` §20.3 that has **no** golden
fixture above. Each is a minimal vCard 4.0 card (`BEGIN/VERSION/UID/FN` + the one property under test),
except `phone`/`email` which are also given as minimal JSContact fragments (illustrating the Id-map
collection shape) since `email`/`phone` are already exercised on the vCard side by the golden baseline
cards. Format: `verbatim RFC NNNN §X.Y` = the property line is copied character-for-character from the
cited `docs/specs/*.md` doc (which itself transcribes the RFC); `hand-authored, minimal,
RFC-syntax-conformant` = no RFC worked example exists for this exact value in `docs/specs/`, so a
minimal value was constructed by hand, following the value-type/cardinality grammar stated in the spec
doc (never contradicting it).

| Fixture | Concept(s) | Format | Provenance |
|---|---|---|---|
| prodid.v4.vcf | prodid | PRODID | hand-authored, minimal, RFC-syntax-conformant (baseline RFC 6350 §6.7.3 property, no verbatim example transcribed in docs/specs) |
| updated.v4.vcf | updated | REV | verbatim RFC 6350 §4.3 TIMESTAMP grammar example (`docs/specs/rfc6350-baseline.md` §1), applied as a REV value |
| language.v4.vcf | language | LANGUAGE | verbatim RFC 9554 §3.3 (`docs/specs/rfc9554-vcard-extensions.md` §1: `LANGUAGE:de-AT`) |
| nickname.v4.vcf | nickname | NICKNAME | hand-authored, minimal, RFC-syntax-conformant (baseline RFC 6350 §6.2.3 text-list property) |
| org-unit.v4.vcf | org.unit | ORG | hand-authored, minimal, RFC-syntax-conformant (baseline RFC 6350 §6.6.4 `name;unit1;unit2` structured value, per `org_units` transform in `20-correspondence.md` §20.4) |
| impp.v4.vcf | impp | IMPP | hand-authored, minimal, RFC-syntax-conformant (baseline RFC 6350 §6.4.3 URI-typed property) |
| adr-tz.v4.vcf | adr.tz | ADR;TZ= | hand-authored, minimal, RFC-syntax-conformant (ADR structure extends the `rfc6350-baseline`/`adr-expanded` golden cards' 7/18-field convention with a TZ param) |
| anniversary-birth.v4.vcf | anniversary.birth | BDAY | verbatim RFC 6350 §4.3 DATE grammar example (`docs/specs/rfc6350-baseline.md` §1: `19850412`), applied as a BDAY value |
| anniversary-wedding.v4.vcf | anniversary.wedding | ANNIVERSARY | hand-authored, minimal, RFC-syntax-conformant (baseline RFC 6350 §6.2.6, same DATE grammar as BDAY) |
| anniversary-death.v4.vcf | anniversary.death | DEATHDATE | verbatim RFC 6474 §2.3 (`docs/specs/rfc6474-birthplace-deathplace.md`: `DEATHDATE:19960415`) |
| anniversary-place-birth.v4.vcf | anniversary.place.birth | BIRTHPLACE | verbatim RFC 6474 §2.1 (`docs/specs/rfc6474-birthplace-deathplace.md`: `BIRTHPLACE:Babies'R'Us Hospital`) |
| anniversary-place-death.v4.vcf | anniversary.place.death | DEATHPLACE | verbatim RFC 6474 §2.2 (`docs/specs/rfc6474-birthplace-deathplace.md`: `DEATHPLACE:Aboard the Titanic\, near Newfoundland`) |
| expertise.v4.vcf | expertise | EXPERTISE | verbatim RFC 6715 §2.1 (`docs/specs/rfc6715-personalinfo.md`: `EXPERTISE;INDEX=1;LEVEL=expert:chemistry`) |
| hobby.v4.vcf | hobby | HOBBY | verbatim RFC 6715 §2.2 (`docs/specs/rfc6715-personalinfo.md`: `HOBBY;INDEX=1;LEVEL=high:reading`) |
| interest.v4.vcf | interest | INTEREST | verbatim RFC 6715 §2.3 (`docs/specs/rfc6715-personalinfo.md`: `INTEREST;INDEX=1;LEVEL=medium:r&b music`) |
| directory.v4.vcf | directory | ORG-DIRECTORY | verbatim RFC 6715 §2.4 (`docs/specs/rfc6715-personalinfo.md`: `ORG-DIRECTORY;PREF=1:ldap://ldap.tech.example/o=Example%20Tech,ou=Engineering`) |
| contacturi.v4.vcf | contacturi | CONTACT-URI | verbatim RFC 8605 (`docs/specs/rfc8605-contacturi.md`: `CONTACT-URI;PREF=1:mailto:contact@example.com`) |
| keywords.v4.vcf | keywords | CATEGORIES | hand-authored, minimal, RFC-syntax-conformant (baseline RFC 6350 §6.7.1 comma-joined text list) |
| photo.v4.vcf | photo | PHOTO | hand-authored, minimal, RFC-syntax-conformant (baseline RFC 6350 §6.2.4 URI-typed property) |
| logo.v4.vcf | logo | LOGO | hand-authored, minimal, RFC-syntax-conformant (baseline RFC 6350 §6.6.3 URI-typed property) |
| sound.v4.vcf | sound | SOUND | hand-authored, minimal, RFC-syntax-conformant (baseline RFC 6350 §6.7.5 URI-typed property) |
| calendar.v4.vcf | calendar | CALURI | hand-authored, minimal, RFC-syntax-conformant (baseline RFC 6350 §6.9.1 URI-typed property) |
| freebusy.v4.vcf | freebusy | FBURL | hand-authored, minimal, RFC-syntax-conformant (baseline RFC 6350 §6.9.2 URI-typed property) |
| caladruri.v4.vcf | caladruri | CALADRURI | hand-authored, minimal, RFC-syntax-conformant (baseline RFC 6350 §6.9.3 URI-typed property) |
| key.v4.vcf | key | KEY | hand-authored, minimal, RFC-syntax-conformant (baseline RFC 6350 §6.8.1 URI-typed property; kept as a URI rather than inline base64 to stay minimal) |
| source.v4.vcf | source | SOURCE | hand-authored, minimal, RFC-syntax-conformant (baseline RFC 6350 §6.1.3 URI-typed property) |
| link.v4.vcf | link | URL | hand-authored, minimal, RFC-syntax-conformant (baseline RFC 6350 §6.7.8 URI-typed property; v3 form of this concept is already exercised by the golden `rfc2426-baseline.v3.vcf`'s `URL` line) |
| lang.v4.vcf | lang | LANG | hand-authored, minimal, RFC-syntax-conformant (baseline RFC 6350 §6.4.4 Language-Tag property) |
| related.v4.vcf | related | RELATED | hand-authored, minimal, RFC-syntax-conformant (baseline RFC 6350 §6.6.6 URI-or-text property with TYPE relation tokens) |
| member.v4.vcf | member | MEMBER, KIND=group | hand-authored, minimal, RFC-syntax-conformant (baseline RFC 6350 §6.6.5; MEMBER is only valid when `KIND:group`, per RFC 6350 §6.1.4) |
| phone.jscontact.json | phone | JSContact `phones` Id-map | hand-authored, minimal, RFC-syntax-conformant (RFC 9553 §2.3.3 `Phone` object shape, `docs/specs/rfc9553-model.md` §1); vCard side of this concept already covered by the golden `rfc2426-baseline.v3.vcf` `TEL` lines |
| email.jscontact.json | email | JSContact `emails` Id-map | hand-authored, minimal, RFC-syntax-conformant (RFC 9553 §2.3.1 `EmailAddress` object shape, `docs/specs/rfc9553-model.md` §1); vCard side of this concept already covered by the golden `rfc6350-baseline.v4.vcf` / `rfc2426-baseline.v3.vcf` `EMAIL` lines |

### Scope note (flagged, not silently decided)

Per `docs/fork-plan/40-testing.md` §40.4, "per-concept minimal" fixtures are "one tiny fixture per
concept group" (singular), not one per concept **per format**. To keep this WP's fixture count
proportionate to its acceptance bar ("Loader compiles; fixtures parse" — `00-overview.md` §0.7), the 30
concepts above got a single vCard 4.0 fixture each (the richest/most-common target format), plus two
illustrative JSContact fixtures (`phone`, `email`) to show the Id-map collection shape referenced in
`docs/fork-plan/40-testing.md` §40.3's pseudo-example. WP-30b (JSContact adapter) and WP-50 (vCard 3.0
adapter) may need additional same-concept fixtures in their own format (JSContact JSON / vCard 3.0) for
their focused import/export tests — those can either be added here (extending this directory) or as
package-local `testdata/` fixtures per `40-testing.md` §40.2's layout; this WP does not attempt to
pre-populate every concept in all three formats.
