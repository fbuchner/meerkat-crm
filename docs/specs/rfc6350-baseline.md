# RFC 6350 — vCard 4.0 baseline mechanics (confirmed reference)

RFC 9553/9554/9555 (the other `docs/specs/` files) all normatively build on RFC 6350. `vcard4` and
`vcard3` (RFC 2426) adapters need this baseline syntax; **the explicit decision for P0 is: reuse the
existing, already-tested `backend/carddav/vcard_mapper.go` logic and the `emersion/go-vcard` library
for this baseline layer rather than reimplementing it from scratch.** This doc exists so that decision
is stated, not assumed, and so an implementer has the core grammar/examples in one place if the salvage
needs extending.

## 1. DATE-AND-OR-TIME / DATE / TIME / TIMESTAMP (§4.3)

```
date          = year [month day] / year "-" month / "--" month [day] / "---" day
date-complete = year month day          (full YYYYMMDD, used by DATE-NOREDUC/timestamp)
time          = hour [minute [second]] [zone] / "-" minute [second] [zone] / "-" "-" second [zone]
date-time     = date-noreduc time-designator time-notrunc
timestamp     = date-complete time-designator time-complete
```
Valid examples: DATE `19850412`, `1985-04`, `1985`, `--0412`, `---12`. TIME `102200`, `1022`, `10`,
`102200Z`, `102200-0800`. DATE-TIME `19961022T140000`, `--1022T1400`, `---22T14`. TIMESTAMP
`19961022T140000Z`, `19961022T140000-05`. UTC-OFFSET (§4.7): `sign hour [minute]`, e.g. `+0500`.

This is the grammar the `date_partial` transform (`20-correspondence.md` §20.4) must parse/emit for
BDAY/ANNIVERSARY/DEATHDATE ↔ `contactmodel.AnniversaryDate`/`PartialDate`. `backend/carddav/
vcard_mapper.go`'s `normalizeBirthday` already handles the common subset (`YYYY-MM-DD`, `--MM-DD`,
`YYYYMMDD`, `--MMDD`) — extend it for the additional reduced forms above if a golden fixture exercises
one that isn't covered.

## 2. TEXT escaping & line folding (§3.2, §3.4)

Escape in TEXT values: `,` → `\,`; `;` → `\;` (compound/structured properties); `\` → `\\`; newline →
`\n`. Line folding: fold at 75 octets (CRLF + one leading whitespace on the continuation; unfold by
removing `CRLF + immediately-following whitespace`). `backend/carddav/vcard_mapper.go`'s
`escapeComponent`/`splitComponents` already implement the structured-value half of this — reuse as
described in `30-adapters.md` §30.B.

## 3. Generic TYPE / PREF parameters (§5.3, §5.6)

`PREF`: integer 1–100, lower = more preferred (maps directly to neutral `Pref *int`).
`TYPE`: multi-purpose; `work`/`home` are the cross-property "context" tokens (→ neutral `Contexts`);
other properties define their own additional TYPE vocabulary (see `TEL` below).

## 4. Canonical baseline example vCard (RFC 6350 §7.2.1) — golden fixture `rfc6350-baseline`

```
BEGIN:VCARD
VERSION:4.0
UID:urn:uuid:4fbe8971-0bc3-424c-9c26-36c3e1eff6b1
FN;PID=1.1:J. Doe
N:Doe;J.;;;
EMAIL;PID=1.1:jdoe@example.com
CLIENTPIDMAP:1;urn:uuid:53e374d9-337e-4727-8803-a1e9c14e0556
END:VCARD
```
This is the simplest possible valid 4.0 card — good as the "does the adapter not choke on a minimal
card" smoke fixture. `CLIENTPIDMAP`/`PID` are not in the neutral model's mapped concept set (no
JSContact/9554 correspondence row references them); they round-trip via `Passthrough.VCard` only.

## 5. Core property value types (§6.2–6.5) — cardinality reference

| Property | Value type | Cardinality |
|---|---|---|
| FN | text | 1..* |
| N | structured text, 5 components (Family;Given;Additional;Prefix;Suffix in 6350 — **expanded to 7 by RFC 9554**, see `rfc9554-vcard-extensions.md`) | 0..1 |
| NICKNAME | text-list | 0..* |
| PHOTO | URI | 0..* |
| BDAY, ANNIVERSARY | date-and-or-time or text | 0..1 |
| GENDER | `sex [";" text]` | 0..1 — **vCard 4.0 only; RFC 2426 (3.0) has no GENDER property at all** (confirmed, see `rfc2426-v3-baseline.md`) |
| ADR | structured text, 7 components in 6350 (POBox;Ext;Street;Locality;Region;Postal;Country — **expanded to 18 by RFC 9554**) | 0..* |
| TEL | text or URI | 0..* |
| EMAIL | text | 0..* |
| IMPP | URI | 0..* |
| LANG | Language-Tag | 0..* |
| TZ | text, URI, or utc-offset | 0..* |
| GEO | URI (`geo:` scheme) | 0..* |

Note: Meerkat's CRM `Gender` field (free text, in `CRMEnvelope` per `10-neutral-model.md`) is
deliberately **not** the same concept as vCard `GENDER` or JSContact `speakToAs`. There is no
correspondence row for it — it never round-trips through the standardized Card, by design.
