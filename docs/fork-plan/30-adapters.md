# 30 — Format adapters (`jscontact`, `vcard4`, `vcard3`)

Three independent packages, each implementing `contactmodel.Importer` + `contactmodel.Exporter`
(0.4). Each reads mappings **only** from `correspondence` (WP-20). None imports another adapter.
All follow the degradation policy (0.5): never hard-error on unmappable data.

Shared driver pattern (all three adapters): iterate `correspondence.Load()`; for each row apply the
named `transform` in the row's direction; collect `Diagnostic`s. Property/param name constants live in
each package's `consts.go` and MUST equal the IANA registry spellings.

---

## 30.A `backend/jscontact` — RFC 9553  ·  WP-30a (types+codec), WP-30b (adapter)

### Files
- `types.go` — Go structs mirroring the JSContact registry objects (Card, Name, NameComponent,
  Address, AddressComponent, Organization, OrgUnit, Title, Phone, EmailAddress, OnlineService,
  Calendar, SchedulingAddress, CryptoKey, Directory, Link, Media, Nickname, LanguagePref, Pronouns,
  SpeakToAs, Anniversary, PartialDate, Timestamp, PersonalInfo, Note, Author, Relation). These are the
  **wire** types (distinct from `contactmodel`), with exact JSON tags and `@type`.
  **`JCardProp` is NOT redefined here** — `jscontact` is permitted to import `contactmodel` (per 0.6's
  dependency direction: adapters depend only on `contactmodel`), so the `vCardProps` field on the wire
  `Card` type uses `contactmodel.JCardProp` (defined in `10-neutral-model.md` §10.4) directly:
  `VCardProps []contactmodel.JCardProp \`json:"vCardProps,omitempty"\``. One type, one owner.
- `codec.go` — `Marshal(*Card) ([]byte,error)` / `Unmarshal([]byte) (*Card,error)`.
- `adapter.go` — `type Adapter struct{}` with `Import`/`Export`.
- tests per WP.

### Codec rules (WP-30a, binding)
1. **`@type`**: every object marshals with its `@type` (e.g. `"@type":"Card"`, `"Name"`, `"Address"`,
   `"Phone"`, …). On unmarshal, `@type` is validated if present, defaulted if absent.
2. **`@version`**: `Card.version` = `"1.0"` on export; accept any on import.
3. **Id-keyed collections**: JSContact stores emails/phones/etc as `{"<id>": {…}}` maps (`Id[Object]`).
   The codec converts between the JSON map and the neutral **ordered slice + `ID`**: preserve map keys
   into `element.ID` on import; emit `element.ID` (or a generated `"k"+index` when empty) as the map
   key on export. Iteration order on export = slice order (stable).
4. **Boolean-set maps** (`contexts`, `features`, `keywords`, `members`, `relation`): JSON
   `{"work":true}` ⇄ neutral `[]string{"work"}`.
5. **Unknown top-level or object properties** on import → `Record.Passthrough.JSContact[pointer]` (do
   not error). Known-but-unmapped-to-neutral (none, if 20 is complete) is impossible by construction.

### Adapter (WP-30b)
```go
func (Adapter) Import(raw []byte) (*contactmodel.Record, []contactmodel.Diagnostic, error)
func (Adapter) Export(r *contactmodel.Record) ([]byte, []contactmodel.Diagnostic, error)
```
- Import: `Unmarshal` → walk correspondence rows with `js_ptr` set → populate `Record.Card` via each
  `transform`'s JSContact→neutral direction → stash unknowns in passthrough.
- Export: walk rows → build a `jscontact.Card` → splice `Passthrough.JSContact`, then
  `Passthrough.VCard` under `vCardProps` (20.5) → `Marshal`.
- JSContact is the near-identity spoke; most transforms are `identity`/structural.

---

## 30.B `backend/vcard4` — vCard 4.0 + RFC 9554/6474/6715/8605  ·  WP-40a (consts+components), WP-40b (adapter)

### Files
- `consts.go` — property + parameter name constants (below).
- `components.go` — structured-value codec, **salvaged and generalized** from
  `backend/carddav/vcard_mapper.go`: `escapeComponent`, `splitComponents`, N/ADR field assembly,
  `PROP-ID`/group/`X-ABLabel` label handling. Extend ADR to the 9554 component-parameter form.
- `adapter.go` — `Import`/`Export` over `github.com/emersion/go-vcard` (`vcard.Card =
  map[string][]*vcard.Field`). Emit `VERSION:4.0`.

### `consts.go` property names (from IANA vcard-elements; all must appear)
`SOURCE KIND XML FN N NICKNAME PHOTO BDAY ANNIVERSARY GENDER ADR TEL EMAIL IMPP LANG TZ GEO TITLE ROLE
LOGO ORG MEMBER RELATED CATEGORIES NOTE PRODID REV SOUND UID CLIENTPIDMAP URL VERSION KEY FBURL
CALADRURI CALURI BIRTHPLACE DEATHPLACE DEATHDATE EXPERTISE HOBBY INTEREST ORG-DIRECTORY CONTACT-URI
CREATED GRAMGENDER LANGUAGE PRONOUNS SOCIALPROFILE JSPROP`
Parameter names:
`LANGUAGE VALUE PREF ALTID PID TYPE MEDIATYPE CALSCALE SORT-AS GEO TZ INDEX LEVEL GROUP CC AUTHOR
AUTHOR-NAME CREATED DERIVED LABEL PHONETIC PROP-ID SCRIPT SERVICE-TYPE USERNAME JSPTR JSCOMPS`

### Adapter rules (WP-40b, binding)
- Use `correspondence` rows where `v4_prop != "-"`, applying `v4_params`.
- Multi-valued props (EMAIL/TEL/ADR/…): one vCard field per neutral slice element; carry `PROP-ID` =
  `element.ID` (generate if empty) so identity survives; `PREF`/`TYPE` from transforms.
- `DERIVED=TRUE`: never import a derived value as authoritative; on export, do not set DERIVED (we emit
  real values). Drop DERIVED props on import into passthrough only if not otherwise mapped.
- Unknown properties on import → `Record.Passthrough.VCard` as `contactmodel.JCardProp` (skip names in
  the mapped set). `JSPROP` on import → `Record.Passthrough.JSContact` keyed by its `JSPTR`.
- Export order + de-dup guard per 20.5.

---

## 30.C `backend/vcard3` — legacy vCard 3.0 (RFC 2426)  ·  WP-50

A **separate, deliberately-duplicated** adapter tuned to 3.0. It does not share emit code with vcard4.

**Correction (no adapter may import another adapter — `00-overview.md` §0.6, load-bearing):** an
earlier draft of this section said the pure `components.go` codec "may be reused by import" from
`vcard4`. That contradicts §0.6's dependency rule and doesn't compile anyway (`vcard4`'s component
functions are unexported). The actual, correct instruction: `vcard3` has **its own**
`components.go`, a duplicated-and-adapted copy of `vcard4`'s pure escape/split/N-assembly/ADR-assembly
logic, trimmed to 3.0's legacy-only field counts (5-field N, 7-field ADR — no 9554 expansion, no
`PROP-ID`/component params). This is a Go copy, not an import — consistent with the "deliberately
duplicated" philosophy already stated for this whole adapter. Write it fresh referencing
`docs/specs/rfc2426-v3-baseline.md`'s N/ADR syntax; you may look at `vcard4/components.go`'s *shape*
for inspiration on structuring the copy, but do not import it.

### Files
- `consts.go` — the 3.0 property/param subset. Per RFC 2426 §5 (see `docs/specs/rfc2426-v3-baseline.md`),
  plus the `X-` extension properties this table actually maps to in `20-correspondence.md`:
  `BEGIN END SOURCE NAME FN N NICKNAME PHOTO BDAY ADR LABEL TEL EMAIL MAILER TZ GEO TITLE ROLE LOGO
  AGENT ORG CATEGORIES NOTE PRODID REV SORT-STRING SOUND UID URL VERSION CLASS KEY IMPP CALURI FBURL
  CALADRURI X-SOCIALPROFILE X-SERVICE-TYPE X-ANNIVERSARY` (`GENDER` deliberately absent — vCard 3.0 has
  no such property; **do not add one**, per `docs/specs/rfc2426-v3-baseline.md`'s confirmed finding).
  Parameter names: `TYPE VALUE ENCODING CHARSET LANGUAGE`. Use the same `Prop`/`Param` constant-naming
  convention as `vcard4`'s `consts.go` (§30.B) for consistency across the two adapters.
- `components.go` — the duplicated-and-adapted pure component codec described above (escape/split/join,
  5-field N assemble/disassemble, 7-field ADR assemble/disassemble; no label/PROP-ID helpers needed since
  3.0 uses grouped `item{n}.X-ABLabel`, a distinct-enough idiom to write directly in `adapter.go` rather
  than factor out).
- `adapter.go` — `Import`/`Export`.

### Rules (binding)
- Use `correspondence` rows where `v3_prop != "-"`, applying `v3_params`.
- For every 20.6 "no-3.0-home" concept present in the Record on **export**: append
  `Diagnostic{Severity:"warn", Concept: <concept_id>}` and omit it from the 3.0 output (it remains in
  the neutral model). WP-50 tests assert these warnings fire.
- 3.0 specifics the agent must honor: `EMAIL;TYPE=INTERNET`; binary media via `ENCODING=b;TYPE=…`;
  custom labels via grouped `item{n}.X-ABLabel` (reuse the salvaged label logic); `X-SOCIALPROFILE`
  and `X-ANNIVERSARY` for the degraded 4.0/9554 concepts noted in the table; ADR/N packed into the
  legacy 7/5-field structured forms (extra 9553 component kinds → warn-drop).
- Import: 3.0 → neutral; unknown props → passthrough; `X-ANNIVERSARY`/`X-SOCIALPROFILE` recognized and
  lifted back into their neutral homes.

---

## 30.D Shared correctness invariants (all adapters)

- Round-trip identity of `PROP-ID`/`ID`: an element imported then exported keeps its key.
- No adapter emits a property not backed by a correspondence row (except verbatim passthrough).
- No adapter returns `error` except on structurally invalid input (0.5).
- `go test ./backend/<pkg>/...` offline (no network, no DB).
