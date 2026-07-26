# RFC 9554 — vCard extension properties/parameters (verbatim syntax + examples)

Transcribed from RFC 9554. Every example line is verbatim; use these to seed golden fixtures.

## 1. New properties

**CREATED** (§3.1) — creation timestamp of the vCard. Single TIMESTAMP.
```
CREATED:20220705T093412Z
CREATED;VALUE=TIMESTAMP:20211022T140000-05
```

**GRAMGENDER** (§3.2) — grammatical gender for salutations. Text, enum:
`animate, common, feminine, inanimate, masculine, neuter` (+ iana-token/x-name).
```
GRAMGENDER;LANGUAGE=de:feminine
```

**LANGUAGE** (§3.3) — default language of human-readable values. Language-Tag.
```
LANGUAGE:de-AT
```

**PRONOUNS** (§3.4) — free-form text; repeatable; LANGUAGE + PREF.
```
PRONOUNS;LANGUAGE=en;PREF=1:xe/xir
PRONOUNS;LANGUAGE=en;PREF=2:they/them
```

**SOCIALPROFILE** (§3.5) — URI (default) or TEXT. SERVICE-TYPE required when VALUE=text, optional when
VALUE=uri (MUST NOT repeat). USERNAME stores the username alongside a URI.
```
SOCIALPROFILE;SERVICE-TYPE=Mastodon:https://example.com/@foo
SOCIALPROFILE:https://example.com/ietf
SOCIALPROFILE;SERVICE-TYPE=SomeSite;VALUE=text:peter94
SOCIALPROFILE;USERNAME="The Foo":https://example.com/@foo
```

## 2. New parameters

**AUTHOR** (§4.1) URI (quoted).  `NOTE;AUTHOR="mailto:john@example.com":This is some note.`
**AUTHOR-NAME** (§4.2) free text (quoted if needed).
```
NOTE;AUTHOR-NAME=John Doe:This is some note.
NOTE;AUTHOR-NAME="_:l33tHckr:_":A note by an unusual author name.
```
**CREATED (param)** (§4.3) TIMESTAMP.  `NOTE;CREATED=20221122T151823Z:This is some note.`
**DERIVED** (§4.4) true/false (default false).
```
N:;John;Quinlan;Mr.;
FN;DERIVED=TRUE:Mr. John Quinlan
```
**LABEL** (§4.5) formatted-address text (on ADR).
```
ADR;LABEL="Mr. John Q. Public, Esq.\nMail Drop: TNE QB\n123
  Main Street\nAny Town, CA  91921-1234\nU.S.A.":
  ;;123 Main Street;Any Town;CA;91921-1234;U.S.A.
```
**PHONETIC** (§4.6) ipa/jyut/piny/script (+iana/x). Same value type as related prop; ALTID must match;
applies to N and ADR.
```
N;ALTID=1;LANGUAGE=zh-Hant:孫;中山;文,逸仙;;;;
N;ALTID=1;PHONETIC=jyut;
  SCRIPT=Latn;LANGUAGE=yue:syun1;zung1saan1;man4,jat6sin1;;;;
```
**PROP-ID** (§4.7) 1–255 [A-Za-z0-9_-].  `PHOTO;PROP-ID=p827:data:image/jpeg;base64,…`
**SCRIPT** (§4.8) script subtag (RFC 5646 §2.2.3), e.g. `SCRIPT=Latn`.
**SERVICE-TYPE** (§4.9) case-sensitive free text (on SOCIALPROFILE/IMPP).
**USERNAME** (§4.10) case-sensitive free text (URI-type props).

## 3. Expanded structured values

**N** (§2.2) — 7 components in order:
`Family; Given; Additional; Prefix; Suffix; SecondarySurname; Generation`
```
N:Public;John;Quinlan;Mr.;Esq.
N:Stevenson;John;Philip,Paul;Dr.;Jr.,M.D.,A.C.P.;;Jr.
```

**ADR** (§2.1) — 18 components in order:
`POBox; Ext; Street; Locality; Region; Code; Country; Room; Apartment; Floor; StreetNumber;
StreetName; Building; Block; Subdistrict; District; Landmark; Direction`
```
ADR;GEO="geo:12.3457,78.910":
  ;;123 Main Street;Any Town;CA;91921-1234;U.S.A
  ;;;;123;Main Street;;;;;;
```
Map JSContact `AddressComponent.kind` → ADR position:
`postOfficeBox`→1, (ext, legacy)→2, `name`(street)→3, `locality`→4, `region`→5, `postcode`→6,
`country`→7, `room`→8, `apartment`→9, `floor`→10, `number`→11, (streetname)→12, `building`→13,
`block`→14, `subdistrict`→15, `district`→16, `landmark`→17, `direction`→18. `JSCOMPS` records order.

## 4. New ADR TYPE values (§5)

`billing` (§5.1), `delivery` (§5.2).
```
ADR;TYPE=billing:;;123 Main Street;Any Town;CA;91921-1234;U.S.A.
ADR;TYPE=delivery:;;123 Main Street;Any Town;CA;91921-1234;U.S.A.
```
