# RFC 2426 — vCard 3.0 baseline (confirmed reference for `vcard3`)

Transcribed from RFC 2426. This is the baseline `backend/vcard3` implements (WP-50).

## 1. Canonical example vCard (§7, Authors' Addresses) — golden fixture `rfc2426-baseline`

```
BEGIN:vCard
VERSION:3.0
FN:Frank Dawson
ORG:Lotus Development Corporation
ADR;TYPE=WORK,POSTAL,PARCEL:;;6544 Battleford Drive
 ;Raleigh;NC;27613-3502;U.S.A.
TEL;TYPE=VOICE,MSG,WORK:+1-919-676-9515
TEL;TYPE=FAX,WORK:+1-919-676-9564
EMAIL;TYPE=INTERNET,PREF:Frank_Dawson@Lotus.com
EMAIL;TYPE=INTERNET:fdawson@earthlink.net
URL:http://home.earthlink.net/~fdawson
END:vCard
```
This is the primary v3 golden fixture — it exercises ORG, multi-valued ADR/TEL/EMAIL with TYPE lists,
and URL, all in the real 3.0 idiom (comma-joined TYPE values, no PREF-as-separate-param — `PREF` is
itself a TYPE token in 3.0, unlike 4.0 where it's a dedicated parameter).

## 2. N / ADR structured syntax (§3.1.2, §3.2.1)

**N**: exactly 5 components — `Family;Given;Additional;Prefix;Suffix`. No secondary-surname/generation
(those are RFC 9554/4.0-only — see `20-correspondence.md`'s degradation table).
Example: `N:Public;John;Quinlan;Mr.;Esq.`

**ADR**: exactly 7 components — `POBox;Ext;Street;Locality;Region;PostalCode;Country`. No room/floor/
building/etc (those are RFC 9554/4.0-only). TYPE values: `dom, intl, postal, parcel, home, work, pref`
(default `intl,postal,parcel,work`).
Example: `ADR;TYPE=dom,home,postal,parcel:;;123 Main Street;Any Town;CA;91921-1234`

## 3. TYPE parameter conventions (§3.3.1–3.3.2)

- **EMAIL**: `INTERNET`, `X400`, `PREF`. Example: `EMAIL;TYPE=internet,pref:jane_doe@abc.com`
- **TEL**: `home, msg, work, pref, voice, fax, cell, video, pager, bbs, modem, car, isdn, pcs`.
  Example: `TEL;TYPE=work,voice,pref,msg:+1-213-555-1234`
- Note the 3.0 idiom: TYPE is a **comma-joined list of tokens on one parameter**, not the 4.0 style of
  repeatable `TYPE=` occurrences — `correspondence.tsv`'s `v3_params` values should be read/emitted
  accordingly (this is exactly what `backend/carddav/vcard_mapper.go`'s existing `addTypedField`/
  `typeTokens` already do — salvage, don't reinvent, per `30-adapters.md` §30.C).

## 4. PHOTO inline encoding (§3.1.4)

`PHOTO;ENCODING=b;TYPE=JPEG:<base64>` — `ENCODING=b` switches to inline base64; `TYPE` names the image
format. (Existing `backend/carddav/vcard_mapper.go` `SaveContactPhoto`/`readContactPhoto` already
implement this exact form.)

## 5. GENDER — confirmed absent (binding fact for the correspondence table)

**RFC 2426 defines no GENDER property.** The 3.0 standard property set is: `FN, N, NICKNAME, PHOTO,
BDAY, ADR, LABEL, TEL, EMAIL, MAILER, TZ, GEO, TITLE, ROLE, LOGO, AGENT, ORG, CATEGORIES, NOTE, PRODID,
REV, SORT-STRING, SOUND, UID, URL, VERSION, CLASS, KEY`. No sex/gender concept exists at all. This
matches the existing Meerkat behavior confirmed during exploration: `ContactToVCard` never emits
GENDER on any output. There is no `v3_prop` for `speakToAs`/`gramgender`/`pronouns`/CRM-`gender` in
`20-correspondence.md` — this is intentional, not an oversight, and needs no `v3_prop` column entry
beyond the existing `-` (degrade).

## 6. `LABEL` (§3.2.2) — 3.0-specific, superseded by 4.0's `LABEL` parameter

vCard 3.0 has a **separate `LABEL` property** (not a parameter) carrying a free-text formatted address,
paired with `ADR` by shared `TYPE`. RFC 9554/6350 fold this into an `ADR` **parameter** instead (see
`rfc9554-vcard-extensions.md` §2 `LABEL` param example). `vcard3`'s adapter must emit/parse the
**property** form (`LABEL;TYPE=...:<text>` as its own line, matched to its `ADR` by `TYPE`); `vcard4`'s
must use the **parameter** form (`ADR;LABEL="...":...`). This is reflected in the `adr` row of
`20-correspondence.md` (`v3_params = TYPE` only, with the property-vs-parameter distinction noted).
