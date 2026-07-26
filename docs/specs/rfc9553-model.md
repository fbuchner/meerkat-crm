# RFC 9553 — JSContact object shapes & value formats (confirmed)

Transcribed from RFC 9553. Resolves the open `VERIFY` points in `docs/fork-plan/10-neutral-model.md`.

## 1. Object property lists (name · type · required)

**Card.created** (§2.1.3) — "The date and time when the Card was created." Value: UTCDateTime.
Cardinality: optional. **Correction (found during post-P0 review):** this property was missing from
this transcription entirely, which propagated into `20-correspondence.md`'s `created` row wrongly
marking `js_ptr` as `-` (no JSContact home) — a real bug (silent data loss on vCard4→JSContact export),
now fixed. Confirmed directly against RFC 9553 §2.1.3 (fetched fresh, not just IANA-registry-inferred).
**Pronouns** (§2.2.4): `pronouns` String **required** (the pronoun text) · `contexts` String[Boolean]
opt · `pref` UnsignedInt opt.  → confirms neutral `Pronouns.Pronouns` holds the text.
**SpeakToAs** (§2.2.4): `grammaticalGender` String opt · `pronouns` Id[Pronouns] opt.
**Name** (§2.2.1.1): `components` NameComponent[] · `isOrdered` Boolean (default false) ·
`defaultSeparator` String · `full` String · `sortAs` String[String] · `phoneticScript` String ·
`phoneticSystem` String.
**NameComponent** (§2.2.1.2): `value` String **req** · `kind` String **req** · `phonetic` String opt.
**Address** (§2.5.1.1): `components` AddressComponent[] · `isOrdered` Boolean(false) · `countryCode`
String · `coordinates` String · `timeZone` String · `contexts` String[Boolean] · `full` String ·
`defaultSeparator` String · `pref` UnsignedInt · `phoneticScript` · `phoneticSystem`.
**AddressComponent** (§2.5.1.2): `value` **req** · `kind` **req** · `phonetic` opt.
**Anniversary** (§2.8.1): `kind` String **req** · `date` PartialDate|Timestamp **req**
(defaultType PartialDate) · `place` Address opt.
**PartialDate** (§2.8.1): `year`/`month`/`day` UnsignedInt opt · `calendarScale` String opt.
**Timestamp** (§2.8.1): `utc` UTCDateTime **req**.
**Phone** (§2.3.3): `number` **req** · `features` String[Boolean] · `contexts` · `pref` · `label`.
**EmailAddress** (§2.3.1): `address` **req** · `contexts` · `pref` · `label`.
**OnlineService** (§2.3.2): `service` opt · `uri` opt · `user` opt · `contexts` · `pref` · `label`.
**PersonalInfo** (§2.8.4): `kind` **req** · `value` **req** · `level` opt · `listAs` UnsignedInt opt ·
`label` opt.
**Nickname** (§2.2.2): `name` **req** · `contexts` · `pref`.
**LanguagePref** (§2.3.4): `language` **req** · `contexts` · `pref`.
**Relation** (§2.1.8): `relation` String[Boolean] (default empty object).

## 2. Value formats & enums

- **Anniversary.date**: PartialDate (optional y/m/d) **or** Timestamp (`{"@type":"Timestamp","utc":…}`).
- **NameComponent.kind**: `title, given, given2, surname, surname2, credential, generation, separator`.
- **AddressComponent.kind**: `room, apartment, floor, building, number, name, block, subdistrict,
  district, locality, region, postcode, country, direction, landmark, postOfficeBox, separator`.
- **phoneticSystem**: `ipa, jyut, piny`. **phoneticScript**: script subtag per RFC 5646 §2.2.3.
- `isOrdered` false ⇒ `defaultSeparator` unused. `sortAs` = map component-kind → sort key.

## 3. Collection typing (drives the JSContact codec, WP-30a)

Id-keyed maps (`Id[Object]` → neutral **slice + `ID`**): `emails, phones, addresses, nicknames,
organizations, titles, onlineServices, media, calendars, schedulingAddresses, cryptoKeys, directories,
links, preferredLanguages, personalInfo, anniversaries, pronouns, notes` (RFC 9553 §2.7.2 defines
`notes` as `Id[Note]`, same shape as the rest of this list — omitted from an earlier pass of this doc).

Boolean sets (`String[Boolean]` → neutral `[]string`): `keywords, members`; and all `contexts` /
`features` / `relation` inner maps.

Keyed-by-value maps: `relatedTo` = `String[Relation]` (key = related URI/text);
`localizations` = `String[PatchObject]` (key = language tag; **P0: opaque `json.RawMessage`**).

## 4. Example Card (verbatim, Figure 6) — golden fixture `johndoe`

```json
{
  "@type": "Card",
  "version": "1.0",
  "uid": "22B2C7DF-9120-4969-8460-05956FE6B065",
  "kind": "individual",
  "name": {
    "components": [
      { "kind": "given", "value": "John" },
      { "kind": "surname", "value": "Doe" }
    ],
    "isOrdered": true
  }
}
```
