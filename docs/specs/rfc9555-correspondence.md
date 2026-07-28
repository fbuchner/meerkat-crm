# RFC 9555 — JSContact ↔ vCard correspondence (oracle authority)

Transcribed from RFC 9555. This is the authority behind `docs/fork-plan/20-correspondence.md`.
Section numbers are RFC 9555 unless noted.

## 1. Property correspondence

| JSContact path | vCard property | vCard params | Notes / § |
|---|---|---|---|
| `name.full` | `FN` | `LANGUAGE, ALTID` | FN mandatory in vCard; if derived from components set `DERIVED=TRUE` (§3.1) |
| `name.components[surname].value` | `N` field 1 (family) | `PHONETIC, SCRIPT, SORT-AS` | Table 1, §2.5.5 |
| `name.components[given].value` | `N` field 2 (given) | — | |
| `name.components[given2].value` | `N` field 3 (additional) | — | |
| `name.components[title].value` | `N` field 4 (prefix) | — | |
| `name.components[credential].value` | `N` field 5 (suffix) | — | |
| `name.components[surname2].value` | `N` field 6 (secondary surname) | — | RFC 9554 N expansion |
| `name.components[generation].value` | `N` field 7 (generation) | — | RFC 9554 N expansion |
| `name.isOrdered` + separators | `N`/`ADR` | `JSCOMPS` | §3.3.1 — if JSCOMPS valid, set `isOrdered=true`, else `false` |
| `name.sortAs` | `N` | `SORT-AS` | |
| `nicknames[].name` | `NICKNAME` | `PREF, TYPE` | §2.5.6 |
| `organizations[].name` | `ORG` (1st comp) | `SORT-AS, TYPE` | §2.9.4 |
| `organizations[].units[].name` | `ORG` (remaining comps) | — | |
| `titles[].name` | `TITLE` or `ROLE` | `ALTID, LANGUAGE` | kind=title→TITLE, kind=role→ROLE; §2.9.6 |
| `titles[].organizationId` | (derived via `GROUP`) | — | organizationId derived when TITLE/ROLE and exactly one ORG share a vCard property GROUP; GROUP itself not preserved; §2.9.6 |
| `emails[].address` | `EMAIL` | `PREF, TYPE` | §2.7.1 |
| `phones[].number` | `TEL` | `PREF, TYPE, VALUE` | §2.7.6 |
| `phones[].features` | `TEL` | `TYPE` | Table 3; JSContact `mobile`↔vCard `cell`, plus fax/voice/video/pager/text/textphone |
| `onlineServices[].uri` | `IMPP` or `SOCIALPROFILE` | `SERVICE-TYPE, USERNAME, PREF, TYPE` | which property preserved in `onlineServices[].vCardName`; §2.7.2, §2.7.5 |
| `addresses[].components` | `ADR` (structured) | `JSCOMPS` | Table 2, §3.3.1 |
| `addresses[].full` | `ADR` | `LABEL` | §2.6.1 |
| `addresses[].coordinates` | `ADR`/`GEO` | `GEO` | §2.8.1 |
| `addresses[].timeZone` | `ADR`/`TZ` | `TZ` | §2.8.2 |
| `addresses[].countryCode` | `ADR` | `CC` | §2.3.5 |
| `addresses[].contexts` | `ADR` | `TYPE` | TYPE work/home ↔ contexts; §2.3.22 |
| `anniversaries[].date` (kind=birth) | `BDAY` | — | §2.5.1 |
| `anniversaries[].date` (kind=wedding) | `ANNIVERSARY` | — | |
| `anniversaries[].date` (kind=death) | `DEATHDATE` | — | RFC 6474 |
| `anniversaries[].place` | `BIRTHPLACE`/`DEATHPLACE` | — | RFC 6474 |
| `speakToAs.grammaticalGender` | `GRAMGENDER` | — | §2.5.4 |
| `speakToAs.pronouns[].pronouns` | `PRONOUNS` | `PREF, LANGUAGE` | §2.5.4 |
| `personalInfo[].value` (kind=expertise) | `EXPERTISE` | `LEVEL, INDEX` | §2.10.1 |
| `personalInfo[].value` (kind=hobby) | `HOBBY` | `LEVEL, INDEX` | §2.10.2 |
| `personalInfo[].value` (kind=interest) | `INTEREST` | `LEVEL, INDEX` | §2.10.3 |
| `notes[].note` | `NOTE` | `LANGUAGE, ALTID` | §2.11.4 |
| `notes[].created` | `NOTE` | `CREATED` | §2.3.6 |
| `notes[].author.name` | `NOTE` | `AUTHOR-NAME` | §2.3.3 |
| `notes[].author.uri` | `NOTE` | `AUTHOR` | §2.3.2 |
| `keywords` | `CATEGORIES` | — | comma-joined; §2.11.1 |
| `media[].uri` (kind=photo/logo/sound) | `PHOTO`/`LOGO`/`SOUND` | `MEDIATYPE, PREF` | §2.5.7, §2.9.2, §2.11.7 |
| `calendars[].uri` (kind=calendar/freeBusy) | `CALURI`/`FBURL` | `PREF` | §2.13.2–3 |
| `schedulingAddresses[].uri` | `CALADRURI` | `PREF` | §2.13.1 |
| `cryptoKeys[].uri` | `KEY` | `PREF, TYPE` | §2.12.1 |
| `directories[].uri` (kind=entry) | `SOURCE` | `PREF` | §2.4.3 |
| `directories[].uri` (kind=directory) | `ORG-DIRECTORY` | `PREF, INDEX` | §2.10.4 |
| `links[].uri` | `URL` | `PREF, TYPE` | §2.11.9 |
| `links[].uri` (kind=contact) | `CONTACT-URI` | `PREF` | §2.9.1 |
| `preferredLanguages[].language` | `LANG` | `PREF, TYPE` | §2.7.3 |
| `relatedTo[target].relation` | `RELATED` | `TYPE` | map key = related URI/text; §2.9.5 |
| `members[uid]` | `MEMBER` | — | map key = member UID; §2.9.3 |
| `uid` | `UID` | — | §2.1.1 |
| `kind` | `KIND` | — | §2.4.2 |
| `prodId` | `PRODID` | — | §2.11.5 |
| `created` | `CREATED` | — | §2.11.3 |
| `updated` | `REV` | — | §2.11.6 |
| `language` | `LANGUAGE` | — | RFC 9554 §3.3 — dedicated vCard4 property, "default language of human-readable values"; direct match for JSContact `language` ("default language for the Card," RFC 9553 §1.4.9). **Correction (found during WP-40b review):** an earlier pass of this transcription omitted this row entirely, which propagated into `20-correspondence.md`'s `language` row incorrectly marking `v4_prop` as `-`. No vCard 3.0 equivalent — RFC 2426 predates this property. |

## 2. Escape-hatch rules (verbatim key sentences)

**vCard → JSContact**
- `vCardProps` (§2.15.1): "Contains vCard properties that are set in the vCard represented by this
  JSContact object. The JCardProp type denotes a jCard-encoded vCard property as defined in Section 3.3
  of [RFC7095]." → array of jCard tuples `[name, params, type, value]`.
- `vCardParams` (§2.15.2): "Contains vCard parameters that are set on the vCard property represented by
  this JSContact object … a JSON object containing vCard property parameters as defined in Section 3.3
  of [RFC7095]." → preserves unknown params on a known property.
- `vCardName` (§2.15.3): preserves which vCard property name produced this object when several map to
  the same JSContact type (e.g. `impp` vs `socialprofile`).

**JSContact → vCard**
- `JSPROP` (§3.2.1): "converts an arbitrary JSContact property from and to vCard. The vCard property
  value is the JSON-encoded value of the JSContact property, represented as a TEXT value. The format …
  MUST be compact, e.g., without insignificant whitespace."
- `JSPTR` (§3.3.2): "set on a JSPROP … Its value is a JSON pointer [RFC6901] that points to the
  JSContact property that has the value of the JSPROP property."
- `JSCOMPS` (§3.3.1): preserves order + separators of structured N/ADR values; semicolon-delimited
  (first entry = default separator, then positional indices / separator entries). Valid JSCOMPS ⇒
  `isOrdered=true`.

## 3. Special parameter rules

- **PROP-ID** (§2.3.18): converts to the Id-typed key of the derived JSContact object. Reverse (§3.1):
  each multivalued property instance sets `PROP-ID` = the JSContact Id key.
- **PREF** (§2.3.17): ↔ `pref`.
- **DERIVED** (§2.3.7): a `DERIVED=TRUE` vCard property MAY be skipped on import. Reverse (§3.1): if
  `FN` is derived from name components, implementations MUST set `DERIVED=TRUE` on `FN`.
- **ALTID** (§2.3.1): no direct JSContact property; used to combine multiple vCard properties into one
  JSContact object (localization); preserve via `vCardParams` if needed.
- **LANGUAGE** (§2.3.11): converts to an entry in the Card's `localizations` for that property; the
  LANGUAGE tag is the localizations key. **P0 decision:** keep `localizations` opaque
  (`json.RawMessage`); treat the untagged/base value as primary; route LANGUAGE-tagged alternates to
  `localizations` (raw) — do not attempt full patch synthesis in P0.

## 4. Worked example (verbatim, §2.9.6) — golden fixture `title-role`

vCard:
```
TITLE:Research Scientist
group1.ROLE:Project Leader
group1.ORG:ABC, Inc.
```
JSContact:
```json
{
  "titles": {
    "TITLE-1": { "kind": "title", "name": "Research Scientist" },
    "TITLE-2": { "kind": "role", "name": "Project Leader", "organizationId": "ORG-1" }
  },
  "organizations": {
    "ORG-1": { "name": "ABC, Inc." }
  }
}
```
Demonstrates: TITLE→kind=title, ROLE→kind=role; `organizationId` derived from a shared property GROUP
with exactly one ORG; the GROUP token itself is not preserved.
