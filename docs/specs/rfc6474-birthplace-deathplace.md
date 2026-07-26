# RFC 6474 — BIRTHPLACE, DEATHPLACE, DEATHDATE (confirmed reference)

## BIRTHPLACE (§2.1)
Purpose: "specify the place of birth." Value: **TEXT (default) or URI**. Cardinality: `*1` (zero or
one — **not** repeatable). Parameters: `VALUE, LANGUAGE, ALTID`, any-param.
```
BIRTHPLACE:Babies'R'Us Hospital
BIRTHPLACE;VALUE=uri:http://example.com/hospitals/babiesrus.vcf
BIRTHPLACE;VALUE=uri:geo:46.769307,-71.283079
```

## DEATHPLACE (§2.2)
Same shape as BIRTHPLACE. Value: TEXT (default) or URI. Cardinality `*1`.
```
DEATHPLACE:Aboard the Titanic\, near Newfoundland
DEATHPLACE;VALUE=uri:http://example.com/ships/titanic.vcf
DEATHPLACE;VALUE=uri:geo:41.731944,-49.945833
```

## DEATHDATE (§2.3)
Value: **date-and-or-time (default) or TEXT**. Cardinality `*1`. Parameters: `VALUE`, `CALSCALE`
(date-and-or-time only), `LANGUAGE` (text only), `ALTID`.
```
DEATHDATE:19960415
DEATHDATE:--0415
DEATHDATE;19531015T231000Z
DEATHDATE;VALUE=text:circa 1800
```

## ⚠ Correction to `10-neutral-model.md` / `20-correspondence.md`

**`Anniversary.Place` is typed `*Address` (a full structured postal address) in the neutral model —
matching JSContact's `Anniversary.place` (RFC 9553 §2.8.1, also `Address`). But vCard's BIRTHPLACE/
DEATHPLACE is a plain TEXT-or-URI scalar, never a structured address.** This is a genuine, asymmetric
type mismatch between the two spokes, not a copy-paste error — it needs a real (lossy) transform, not
a 1:1 field mapping:

- **vCard export** (`place_text` transform, refined): if `Address.Full` is set, emit it as
  `BIRTHPLACE:<Full>` (TEXT). Else if `Address.Coordinates` is set (a `geo:` URI) and no other
  component is populated, emit `BIRTHPLACE;VALUE=uri:<Coordinates>`. Else if any structured component
  is populated, join them into a single text value (reuse the `FormatAddress`-style join already used
  elsewhere in this codebase) and emit as TEXT — structure is lost on this leg, which is expected and
  acceptable (not every structured component has a place to go in a scalar TEXT value).
- **vCard import**: TEXT value → `Address{Full: <value>}`. `VALUE=uri` with a `geo:` scheme →
  `Address{Coordinates: <value>}`. `VALUE=uri` with any other scheme → treat as unmappable-URI, warn +
  passthrough (no structured-address field fits an arbitrary URI).
- This is a **defect-detectable-but-not-strictly-fatal** case per the degradation policy (0.5): losing
  structured components on vCard export is expected and warned, not an error.

Update `20-correspondence.md`'s `anniversary.place.birth`/`anniversary.place.death` rows' `notes`
column to state this explicitly (done — see that file).
