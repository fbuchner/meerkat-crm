# 20 — Correspondence oracle (`backend/correspondence`)  ·  WP-20

> **LOCKED.** This table is authored and verified against `docs/specs/rfc9555-correspondence.md`
> (RFC 9555). Implementers MUST NOT add, remove, or alter rows. If a needed mapping is missing or
> ambiguous, STOP and escalate per `60-review-gates.md` — do not invent one.

This is the **single source of mapping truth**. Adapters read it to know where each concept goes;
tests read it to know what to assert. It is materialized as a TSV in `testdata/` plus a typed loader.
RFC 9555 §2–3 is the authority behind these rows; where a mapping is not obvious, the `notes` column
cites the rule. **No adapter or test may encode a mapping not present here.**

## 20.1 Row schema (`testdata/correspondence.tsv`, tab-separated, header row)

| col | meaning |
|---|---|
| `concept_id` | stable unique key (e.g. `phone`, `name.given`, `anniversary.birth`) |
| `neutral_path` | path into `contactmodel.Record` — see §20.2's exact grammar and resolution algorithm |
| `js_ptr` | JSContact location (JSON-pointer-ish, `{id}` = collection key) |
| `v4_prop` | vCard 4.0 property name (`-` = none) |
| `v4_params` | vCard 4.0 params applied (`;`-sep, `-` = none) |
| `v3_prop` | vCard 3.0 property name (`-` = none → degrade) |
| `v3_params` | vCard 3.0 params (`-` = none) |
| `transform` | named value transform (20.4) |
| `notes` | rule / RFC 9555 reference / degradation |

## 20.2 `table.go` — loader + verifier

```go
package correspondence

type Row struct {
    ConceptID, NeutralPath, JSPtr string
    V4Prop, V4Params, V3Prop, V3Params, Transform, Notes string
}
// Load parses testdata/correspondence.tsv (embedded via //go:embed) into []Row.
func Load() []Row
// ByConcept indexes rows by ConceptID (unique; loader panics on dup).
func ByConcept() map[string]Row
```

### `neutral_path` grammar (exact — do not invent a different resolution scheme)

```
neutral_path := segment ("." segment)*
segment      := FieldName
              | FieldName "[]"                 // descend into a slice's element type
              | FieldName "[" Key "=" Value "]"  // descend into a slice's element type;
                                                   // Key/Value are documentation only (see below)
```
`FieldName` is a literal exported Go field name (e.g. `Card`, `Name`, `Components`, `Value`).
`Key` in every row of this table is currently always `kind`; `Value` is one of that concept's enum
members (e.g. `surname`, `birth`, `role`) — written for human readability, matching the `Kind` field's
intended value, but **not mechanically checked against the enum** (Go reflection cannot validate
business-rule enum membership; that check is done by a human against `docs/specs/` when authoring the
row).

**Resolution algorithm (this is exactly what `correspondence_test.go` implements — no more, no less):**
```
cur := reflect.TypeOf(contactmodel.Record{})
for each segment in path.split("."):
    name, bracket := parseSegment(segment)      // bracket ∈ {none, "[]", "[kind=X]"}
    field, ok := cur.FieldByName(name)
    if !ok { FAIL "unknown field " + name + " on " + cur.String() }
    t := field.Type
    if t.Kind() == reflect.Ptr { t = t.Elem() }  // *Name, *Address, *SpeakToAs, etc.
    if bracket != none:
        if t.Kind() != reflect.Slice { FAIL "field " + name + " is not a slice, cannot use [] / [kind=X]" }
        elem := t.Elem()
        if bracket == "[kind=X]":
            key := "Kind"  // Go field name for the predicate key "kind" — Title-cased
            if _, ok := elem.FieldByName(key); !ok {
                FAIL "predicate key 'kind' has no corresponding field on " + elem.String()
            }
        cur = elem   // descend into the element type for the next segment
    else:
        cur = t
// after the loop, `cur` is the resolved leaf type — no further check needed;
// the loop already failed fast on any unresolvable segment.
```
This is pure `reflect` — no custom path DSL beyond the grammar above, and no enum-value checking (that
stays a human review step per `60-review-gates.md`).

**Worked example:** `Card.Name.Components[kind=surname].Value` resolves as:
`Record` →(`Card`)→ `Card` →(`Name`, `*Name`→deref)→ `Name` →(`Components[kind=surname]`, slice→elem,
verify `NameComponent` has field `Kind`)→ `NameComponent` →(`Value`)→ `string`. ✓.

`correspondence_test.go` asserts: (a) no duplicate `concept_id`; (b) every `neutral_path` resolves per
the algorithm above; (c) every `transform` name exists in the transforms registry (20.4); (d) every
`v4_prop`/`v3_prop` (when not `-`) is in the IANA property set listed in `30-adapters.md` §consts.

## 20.3 The table (author these rows into the TSV)

Grouped for reading; in the TSV they are flat rows. `ctx`=Contexts, `pref`=Pref, `[]`=per element.

### Identity / meta
| concept_id | neutral_path | js_ptr | v4_prop | v4_params | v3_prop | v3_params | transform | notes |
|---|---|---|---|---|---|---|---|---|
| uid | Card.UID | /uid | UID | - | UID | - | identity | |
| kind | Card.Kind | /kind | KIND | - | - | - | identity | 3.0: no KIND → warn-drop |
| prodid | Card.ProdID | /prodId | PRODID | - | PRODID | - | identity | |
| updated | Card.Updated | /updated | REV | - | REV | - | ts_rfc3339 | |
| created | Card.Created | /created | CREATED | - | - | - | ts_rfc3339 | **Corrected (was wrongly `-`, see `docs/specs/rfc9553-model.md`'s Card.created entry).** RFC 9553 §2.1.3 defines `created` as a Card-level UTCDateTime property; RFC 9554 defines vCard4 `CREATED`. Direct match. 3.0 none. |
| language | Card.Language | /language | LANGUAGE | - | - | - | identity | **Corrected (was wrongly `-`, see `docs/specs/rfc9555-correspondence.md`'s language row).** RFC 9554 §3.3 defines vCard4 `LANGUAGE` as a dedicated property, "default language of human-readable values" — a direct 1:1 match for this concept. v3 (RFC 2426) predates this property entirely → no v3 home, warn-drop on 3.0 export only (§20.6). |

### Name  (vCard N order per RFC 9554 §2.2 = Family;Given;Additional;Prefix;Suffix;Surname2;Generation)
| name.full | Card.Name.Full | /name/full | FN | DERIVED | FN | - | identity | FN required; if `Full` empty, derive from components and set `DERIVED=TRUE` on v4 FN (RFC 9555 §3.1) |
| name.surname | Card.Name.Components[kind=surname].Value | /name/components | N[0] | - | N[0] | - | n_component | family |
| name.given | Card.Name.Components[kind=given].Value | /name/components | N[1] | - | N[1] | - | n_component | |
| name.given2 | Card.Name.Components[kind=given2].Value | /name/components | N[2] | - | N[2] | - | n_component | additional/middle |
| name.title | Card.Name.Components[kind=title].Value | /name/components | N[3] | - | N[3] | - | n_component | honorific prefix |
| name.credential | Card.Name.Components[kind=credential].Value | /name/components | N[4] | - | N[4] | - | n_component | honorific suffix |
| name.surname2 | Card.Name.Components[kind=surname2].Value | /name/components | N[5] | - | - | - | n_component | 9554; v3 N has only 5 fields → warn-drop |
| name.generation | Card.Name.Components[kind=generation].Value | /name/components | N[6] | - | - | - | n_component | 9554; v3 → warn-drop |
| name.phonetic | Card.Name.PhoneticScript | /name/phoneticScript | N | PHONETIC;SCRIPT;ALTID | - | - | identity | 9554 §4.6; paired via ALTID; v3 none. `PhoneticScript` is the anchor field; the same code path also jointly populates `Card.Name.PhoneticSystem` (from the `PHONETIC` param's own value) and per-component `NameComponent.Phonetic` (same "joint anchor field" convention as `org`/`social`, now documented explicitly — implementation already does this correctly, this note was previously missing). |

### Nicknames
| nickname | Card.Nicknames[].Name | /nicknames/{id}/name | NICKNAME | PREF;TYPE | NICKNAME | TYPE | identity | ctx→TYPE via ctx2type; pref→PREF |

### Organizations / titles
| org | Card.Organizations[].Name | /organizations/{id}/name | ORG | - | ORG | - | org_units | ORG = name;unit1;unit2 (units joined) |
| org.unit | Card.Organizations[].Units[].Name | /organizations/{id}/units | ORG | - | ORG | - | org_units | |
| title | Card.Titles[kind=title].Name | /titles/{id}/name | TITLE | - | TITLE | - | identity | Title.Kind=title |
| role | Card.Titles[kind=role].Name | /titles/{id}/name | ROLE | - | ROLE | - | identity | Title.Kind=role |

### Emails / phones / online services
| email | Card.Emails[].Address | /emails/{id}/address | EMAIL | PREF;TYPE | EMAIL | TYPE | identity | v3 EMAIL also carries TYPE=INTERNET; ctx2type; pref→PREF |
| phone | Card.Phones[].Number | /phones/{id}/number | TEL | PREF;TYPE | TEL | TYPE | identity | features→TYPE via feat2type; ctx2type; pref→PREF |
| impp | Card.ImppAddresses[].URI | /onlineServices/{id}/uri | IMPP | PREF;TYPE;SERVICE-TYPE;USERNAME | IMPP | X-SERVICE-TYPE | identity | **Corrected — discrete field, not shared, three-array design (see §20.7).** `ImppAddresses`, `SocialProfiles`, and `OtherOnlineServices` are three separate neutral fields — vCard import/export of `ImppAddresses`/`SocialProfiles` is always unambiguous (no per-element tag needed). **`Card.OtherOnlineServices[]` is NOT covered by this row** — it has no vCard export at all (see §20.7: neither IMPP nor SOCIALPROFILE is a safe default guess for genuinely unclassified data, so it warn-drops rather than guessing). RFC 9554 §4.9/§4.10: SERVICE-TYPE/USERNAME "MAY be specified on an IMPP or a SOCIALPROFILE property" — read/written on both. |
| social | Card.SocialProfiles[].Service | /onlineServices/{id} | SOCIALPROFILE | SERVICE-TYPE;USERNAME | X-SOCIALPROFILE | TYPE | onlineservice | 9554 SOCIALPROFILE for 4.0; 3.0 → X-SOCIALPROFILE + X-SERVICE-TYPE. `Service` is the anchor field; the `onlineservice` transform jointly reads/writes the sibling `.User` field on the same element (same convention as the `org`/`org_units` row anchoring on `.Name` while jointly handling `.Units[]`) — `.User` has no separate correspondence row. **Discrete field, superseded by the three-array `vCardName` design (§20.7) — this note previously described a presence-based heuristic that is no longer correct and was never the implemented behavior.** vCard import/export is always unambiguous since the source property is known. JSContact import routes by the `vCardName` hint (RFC 9555 §2.15.3) when present — `"impp"`→`ImppAddresses`, `"socialprofile"`→`SocialProfiles` — and to `Card.OtherOnlineServices[]` (never a presence-based guess) when the hint is absent; see `impp` row and §20.7 for the full rule. |

### Addresses
| adr | Card.Addresses[] | /addresses/{id} | ADR | LABEL;GEO;TZ;CC;PREF;TYPE | ADR | TYPE | adr_components | ADR value = POBox;Ext;Street;Locality;Region;Postal;Country built from components; 9554 adds JSCOMPS/component params; v3 packs into 7 legacy fields, extra 9554 components → warn-drop. **`addresses[].full` on v3 is NOT an ADR parameter** (RFC 2426 has no ADR `LABEL` param) — it emits/parses as a separate `LABEL` *property*, paired to its ADR by matching `TYPE`. See `docs/specs/rfc2426-v3-baseline.md` §6. |
| adr.geo | Card.Addresses[].Coordinates | /addresses/{id}/coordinates | ADR | GEO | - | - | geo_uri | v3: separate GEO property (lat;lon) |
| adr.tz | Card.Addresses[].TimeZone | /addresses/{id}/timeZone | ADR | TZ | - | - | identity | v3: separate TZ property |

### Anniversaries
| anniversary.birth | Card.Anniversaries[kind=birth].Date | /anniversaries/{id}/date | BDAY | CALSCALE | BDAY | - | date_partial | |
| anniversary.wedding | Card.Anniversaries[kind=wedding].Date | /anniversaries/{id}/date | ANNIVERSARY | CALSCALE | - | - | date_partial | 3.0: no ANNIVERSARY → X-ANNIVERSARY (warn). **Corrected**: RFC 6350's IANA parameter registry groups BDAY/ANNIVERSARY together as both taking CALSCALE — was previously omitted here (documentation-only fix; the code already shares one CALSCALE-handling path across birth/wedding/death, so no code change needed). |
| anniversary.death | Card.Anniversaries[kind=death].Date | /anniversaries/{id}/date | DEATHDATE | VALUE;CALSCALE | - | - | date_partial | RFC 6474 §2.3; cardinality *1; `VALUE=text` variant ("circa 1800") preserved via passthrough since neutral `AnniversaryDate` has no free-text case; 3.0 none |
| anniversary.place.birth | Card.Anniversaries[kind=birth].Place | /anniversaries/{id}/place | BIRTHPLACE | VALUE | - | - | place_text | RFC 6474 §2.1. **Type mismatch, not a 1:1 field map**: neutral `Place` is a full `*Address` (matches JSContact `Anniversary.place`, RFC 9553 §2.8.1) but BIRTHPLACE is TEXT-or-URI only. Export: `Address.Full` → TEXT; else `Address.Coordinates` (geo: URI) → `VALUE=uri`; else join components → TEXT (structure lossy, warn). Import: TEXT → `Address{Full}`; `VALUE=uri` geo: → `Address{Coordinates}`; other URI schemes → passthrough. See `docs/specs/rfc6474-birthplace-deathplace.md`. |
| anniversary.place.death | Card.Anniversaries[kind=death].Place | /anniversaries/{id}/place | DEATHPLACE | VALUE | - | - | place_text | RFC 6474 §2.2; same transform/type-mismatch handling as `anniversary.place.birth` above. |

### SpeakToAs
| gramgender | Card.SpeakToAs.GrammaticalGenders[].Value | /speakToAs/grammaticalGender | GRAMGENDER | LANGUAGE | - | - | enum_lower | 9554 §3.2; enum: animate/common/feminine/inanimate/masculine/neuter; 3.0 none. **Resolved — now multi-valued, matching RFC 9554's actual cardinality `*`.** `Card.SpeakToAs.GrammaticalGenders` is a slice (element type `GrammaticalGender{ID,Value,Language}`, see `10-neutral-model.md`); vCard4 import stores every occurrence losslessly (never drops data on import — see 0.5). Export to JSContact (whose own `speakToAs.grammaticalGender` is a scalar, RFC 9553 §2.2.4 — a real, RFC-inherent single-valued limit, not a gap in our model) selects: if `Card.Language` is set and a `GrammaticalGenders[]` entry's `Language` matches it, use that entry's `Value`; otherwise use the first entry. This loss is expected and happens only on export to a structurally single-valued format, per the "loss is fine on export, never on import" principle — not a defect to warn about. Export to vCard4 re-emits every stored entry with its own `LANGUAGE` param (full fidelity, vCard4-to-vCard4). |
| pronouns | Card.SpeakToAs.Pronouns[].Pronouns | /speakToAs/pronouns/{id}/pronouns | PRONOUNS | LANGUAGE;PREF;TYPE | - | - | identity | 9554 §3.4 (params: LANGUAGE, PREF, TYPE, ALTID); 3.0 none. **Corrected**: `TYPE` was missing from `v4_params`, and `Card.SpeakToAs.Pronouns[].Contexts` (mirrors JSContact `Pronouns.contexts`, already round-tripped correctly by the `jscontact` adapter) was never wired to vCard4's `TYPE` param in `vcard4/adapter.go` despite the neutral field existing — a `PRONOUNS;TYPE=home:...` import silently lost context, and a JSContact `pronouns[].contexts` value silently lost it on vCard4 export. |

### Personal info
| expertise | Card.PersonalInfo[kind=expertise] | /personalInfo/{id} | EXPERTISE | LEVEL;INDEX | - | - | personalinfo | 6715; level: beginner/average/expert; 3.0 none |
| hobby | Card.PersonalInfo[kind=hobby] | /personalInfo/{id} | HOBBY | LEVEL;INDEX | - | - | personalinfo | 6715 level high/medium/low |
| interest | Card.PersonalInfo[kind=interest] | /personalInfo/{id} | INTEREST | LEVEL;INDEX | - | - | personalinfo | 6715 |

### Notes / keywords
| note | Card.Notes[].Note | /notes/{id}/note | NOTE | AUTHOR;AUTHOR-NAME;CREATED | NOTE | - | identity | author/created params 9554; 3.0 drops params |
| keywords | Card.Keywords | /keywords | CATEGORIES | - | CATEGORIES | - | csv_join | comma-joined |

### Resources
| photo | Card.Media[kind=photo].URI | /media/{id}/uri | PHOTO | MEDIATYPE;PREF | PHOTO | ENCODING;TYPE | media_uri | v3 inline uses ENCODING=b + TYPE=JPEG |
| logo | Card.Media[kind=logo].URI | /media/{id}/uri | LOGO | MEDIATYPE | LOGO | ENCODING;TYPE | media_uri | |
| sound | Card.Media[kind=sound].URI | /media/{id}/uri | SOUND | MEDIATYPE | SOUND | ENCODING;TYPE | media_uri | |
| calendar | Card.Calendars[].URI | /calendars/{id}/uri | CALURI | PREF | CALURI | - | identity | **Discrete field** (was `Card.Calendars[kind=calendar]`) — `Calendars` now holds CALURI only, `FreeBusyURLs` holds FBURL only; JSContact import routes by its own `kind` sub-field (`calendar`→`Calendars`, `freeBusy`→`FreeBusyURLs`), never ambiguous. |
| freebusy | Card.FreeBusyURLs[].URI | /calendars/{id}/uri | FBURL | PREF | FBURL | - | identity | **Discrete field** (was `Card.Calendars[kind=freeBusy]`) — see `calendar` row. |
| caladruri | Card.SchedulingAddresses[].URI | /schedulingAddresses/{id}/uri | CALADRURI | PREF | CALADRURI | - | identity | |
| key | Card.CryptoKeys[].URI | /cryptoKeys/{id}/uri | KEY | MEDIATYPE | KEY | ENCODING;TYPE | identity | |
| directory | Card.Directories[kind=directory].URI | /directories/{id}/uri | ORG-DIRECTORY | PREF;INDEX | - | - | identity | 6715; 3.0 none |
| source | Card.Directories[kind=entry].URI | /directories/{id}/uri | SOURCE | PREF | SOURCE | - | identity | |
| link | Card.Links[].URI | /links/{id}/uri | URL | PREF;TYPE | URL | TYPE | identity | **Discrete field** — `Links` now holds URL only (was Kind-shared with CONTACT-URI); JSContact import routes by its own `kind` sub-field (absent/other→`Links`, `contact`→`ContactURIs`). |
| contacturi | Card.ContactURIs[].URI | /links/{id}/uri | CONTACT-URI | PREF | - | - | identity | 8605; 3.0 none. **Discrete field** (was `Card.Links[kind=contact]`) — see `link` row. |

### Langs / related / members
| lang | Card.PreferredLanguages[].Language | /preferredLanguages/{id}/language | LANG | PREF;TYPE | - | - | identity | 3.0: no LANG → warn-drop |
| related | Card.RelatedTo[] | /relatedTo/{target} | RELATED | TYPE | - | - | related | TYPE = relation tokens; 3.0: no RELATED → AGENT/warn |
| member | Card.Members | /members | MEMBER | - | - | - | identity | group kind only; 3.0 none |

### Passthrough (both directions)
| pt.vcard | Passthrough.VCard | /vCardProps | *verbatim* | *verbatim* | *verbatim* | *verbatim* | passthrough_vcard | re-emit stored jCard props on vCard export (skip any now-mapped name) |
| pt.jscontact | Passthrough.JSContact | (pointer keys) | JSPROP | JSPTR | - | - | passthrough_js | JSContact-only unknowns; on vCard4 export become JSPROP+JSPTR (9555); 3.0 warn-drop |

## 20.4 Transforms registry (define each in `correspondence` or the adapters, one impl, shared)

| transform | spec |
|---|---|
| `identity` | copy string value unchanged |
| `ts_rfc3339` | neutral `Timestamp.UTC` ⇄ vCard TIMESTAMP (`YYYYMMDDThhmmssZ`) |
| `date_partial` | neutral `AnniversaryDate` ⇄ vCard DATE-AND-OR-TIME (`YYYY-MM-DD`, `--MM-DD`, `YYYY`) |
| `n_component` | one N structured field ⇄ NameComponent; N order = Family;Given;Additional;Prefix;Suffix |
| `adr_components` | AddressComponent[] ⇄ ADR `POBox;Ext;Street;Locality;Region;Postal;Country`; extra 9553 kinds (room,floor,block,…) use 9554 component params on v4, warn-drop on v3 |
| `org_units` | `Organization.Name` + `Units[]` ⇄ ORG `name;unit1;unit2` |
| `ctx2type` | Contexts `private→home`, `work→work` ⇄ vCard `TYPE` |
| `feat2type` | Phone.Features ⇄ TEL TYPE: `voice,fax,cell,video,pager,text,textphone,main-number`; JSContact `mobile`↔vCard `cell` |
| `pref` | neutral `Pref` (1..100, lower=more preferred) ⇄ vCard `PREF` param |
| `enum_lower` | lowercase enum passthrough (values already aligned across formats) |
| `csv_join` | []string ⇄ comma-joined property value |
| `geo_uri` | `geo:lat,lon` ⇄ v4 GEO param / v3 GEO property `lat;lon` |
| `media_uri` | data: URI ⇄ v3 inline base64 (ENCODING=b, TYPE) / v4 MEDIATYPE |
| `onlineservice` | Service+User+URI ⇄ SOCIALPROFILE(SERVICE-TYPE,USERNAME) / IMPP |
| `personalinfo` | value + level + listAs ⇄ EXPERTISE/HOBBY/INTEREST(LEVEL,INDEX); map JSContact level (high/med/low) ↔ 6715 EXPERTISE (beginner/average/expert) per notes |
| `related` | Relation.Relations[] ⇄ RELATED TYPE tokens |
| `place_text` | Address ⇄ BIRTHPLACE/DEATHPLACE text value |
| `passthrough_vcard` / `passthrough_js` | escape-hatch re-emit (see 0.5) |

## 20.5 Escape-hatch precedence (binding)

On **vCard export**: emit mapped props first; then re-emit `Passthrough.VCard` skipping any property
name that a mapped row already produced (prevents duplicates — mirrors the existing
`mappedVCardFields()` guard in `carddav/vcard_mapper.go`); then emit `Passthrough.JSContact` as `JSPROP`
(v4 only). On **JSContact export**: emit mapped props; then splice `Passthrough.JSContact`; then emit
`Passthrough.VCard` under `vCardProps`.

## 20.6 vCard 3.0 degradation summary (drives WP-50 warning assertions)

No-3.0-home concepts → `Diagnostic{warn}` + drop from the 3.0 serialization only: `kind`, `created`,
`anniversary.wedding`(→X-ANNIVERSARY), `anniversary.death`, `anniversary.place.birth`,
`anniversary.place.death`, `gramgender`, `pronouns`, `expertise`, `hobby`, `interest`, `directory`,
`contacturi`, `lang`, `language`, `related`, `member`, `name.surname2`, `name.generation`, name/adr
`PHONETIC`, extra ADR component kinds, note AUTHOR/CREATED params, `pt.jscontact` (no vCard-3.0 home
for JSContact-originated unknowns — they simply don't survive a 3.0 round trip, which is expected: 3.0
predates JSContact entirely). Everything else maps. (This list was originally incomplete — three
rows genuinely have `v3_prop = "-"` but were missing from the prose above; WP-50's agent caught this by
checking every row mechanically rather than trusting the prose list, which is exactly the discipline
`60-review-gates.md` calls for.)

## 20.7 Locked refinements (verified against `docs/specs/`, binding)

- **ADR components** (`adr_components`): vCard 4.0 ADR has **18** positional components
  (RFC 9554 §2.1). The authoritative `AddressComponent.kind → position` map and the 18-field example
  are in `docs/specs/rfc9554-vcard-extensions.md §3`. `JSCOMPS` records component order and, when
  present/valid, sets `Address.IsOrdered=true` (RFC 9555 §3.3.1). vCard 3.0 ADR has 7 fields; extra
  kinds → warn-drop.
- **onlineServices ↔ IMPP vs SOCIALPROFILE — three-array design (implemented, not just noted).**
  `Card.ImppAddresses[]`/`Card.SocialProfiles[]`/`Card.OtherOnlineServices[]` are three separate neutral
  fields (all element type `OnlineService`), not one shared collection:
  - **vCard import/export is always unambiguous** — `IMPP` always reads/writes `ImppAddresses[]`,
    `SOCIALPROFILE` always reads/writes `SocialProfiles[]`. Which array an element lives in **is** its
    provenance record; no per-element tag is needed for the vCard side at all.
  - **JSContact has no `kind`-style discriminator for `onlineServices`** (unlike `links`/`calendars`,
    which do — that's why only this concept needs the escape hatch). RFC 9555 §2.15.3's `vCardName`
    JSON property is the mechanism: on **JSContact import**, a `vCardName: "impp"` or
    `"socialprofile"` hint (RFC 9555 §2.15.3's own Figure 17/47 examples use lowercase; comparison
    MUST be case-insensitive per the RFC, so match case-insensitively on import but emit lowercase on
    export to match real RFC-9555 producers/consumers) routes the entry into the matching array;
    absent → `OtherOnlineServices[]`
    (unclassified — either genuinely JSContact-native data, or a GUI-added entry with no vCard-property
    preference). On **JSContact export**, `ImppAddresses[]`/`SocialProfiles[]` entries are tagged with
    the matching `vCardName` so a later re-import (by us or any other RFC-9555 tool) is fully faithful,
    not just heuristic; `OtherOnlineServices[]` entries get no `vCardName` tag.
  - **vCard export of `OtherOnlineServices[]`**: no fallback — entries here are **dropped from vCard
    output entirely** (both v4 and v3), with a `Diagnostic{warn}` per element, same as any other
    no-target-home concept. Neither `IMPP` nor `SOCIALPROFILE` is a safe default guess for genuinely
    unclassified data; picking one would assert something we don't actually know. The data is not
    destroyed — it stays on the neutral `Record` and is retrievable/exportable to JSContact — it's only
    absent from *that* vCard serialization until a user (or future import) classifies it. The intended
    product shape: the GUI surfaces `OtherOnlineServices` entries as needing classification into IMPP or
    Social; the bucket is meant to be transient, and an unclassified entry exporting incompletely to
    vCard is an acceptable, expected consequence of not yet classifying it — not a bug to route around.
  - `contactmodel.OnlineService` itself is unchanged (`ID,Service,URI,User,Contexts,Pref,Label`) — only
    `backend/jscontact`'s wire type gains the `vCardName` field; it is a JSContact-wire-level escape
    hatch, not a neutral-model field, since the neutral model already encodes provenance structurally
    via which of the three arrays an element lives in.
- **titles.organizationId ↔ vCard GROUP**: derived, not a direct property. On import, set
  `organizationId` when a TITLE/ROLE and exactly one ORG share a vCard property group; on export, emit
  a shared `groupN.` prefix on the TITLE/ROLE and its ORG. The GROUP token itself is not otherwise
  preserved (RFC 9555 §2.9.6). See fixture `title-role`.
- **LANGUAGE param ↔ localizations**: a `LANGUAGE`-tagged alternate value of a property becomes a
  `localizations[lang]` entry (RFC 9555 §2.3.11). **P0:** keep `localizations` opaque; treat the
  untagged value as primary; route tagged alternates into `Localizations` raw (or `Passthrough`).
- **phones features**: JSContact `mobile` ↔ vCard `cell` (`feat2type`); all others identical.
- **DERIVED**: never import a `DERIVED=TRUE` value as authoritative; on export set `DERIVED=TRUE` on
  `FN` when it was derived from name components (RFC 9555 §3.1; fixture `derived-fn`).
