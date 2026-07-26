# RFC 6715 — EXPERTISE, HOBBY, INTEREST, ORG-DIRECTORY (confirmed reference)

## EXPERTISE (§2.1)
Purpose: field of expertise. Value: single TEXT. Cardinality: `*` (multi-valued). `LEVEL` values:
**`beginner`, `average`, `expert`**.
```
EXPERTISE;LEVEL=beginner;INDEX=2:chinese literature
EXPERTISE;INDEX=1;LEVEL=expert:chemistry
```

## HOBBY (§2.2)
Value: single TEXT. Cardinality `*`. `LEVEL` values: **`high`, `medium`, `low`**.
```
HOBBY;INDEX=1;LEVEL=high:reading
HOBBY;INDEX=2;LEVEL=high:sewing
```

## INTEREST (§2.3)
Same shape as HOBBY. `LEVEL` values: `high`, `medium`, `low`.
```
INTEREST;INDEX=1;LEVEL=medium:r&b music
INTEREST;INDEX=2;LEVEL=high:rock 'n' roll music
```

## ORG-DIRECTORY (§2.4)
Purpose: a directory of an organization the vCard's entity belongs to. Value: single URI.
Cardinality `*`. Parameters: `INDEX`, `PREF`.
```
ORG-DIRECTORY;INDEX=1:http://directory.mycompany.example.com
ORG-DIRECTORY;PREF=1:ldap://ldap.tech.example/o=Example%20Tech,ou=Engineering
```
**Errata check (verified 2026-07-25):** RFC 6715 Erratum 3341 (verified) corrects a typo in the RFC's
own §5 IANA-registration summary table — `ORG-URI` → `ORG-DIRECTORY`. No effect on our docs; we already
use the correct spelling `ORG-DIRECTORY` throughout `20-correspondence.md`/`30-adapters.md`.

## INDEX parameter (§3.1)
"Used in a multi-valued property to indicate the position of this value within the set of values."
Strictly positive integers (no zero). This is the vCard-side counterpart of JSContact
`PersonalInfo.listAs` / `Directory.listAs` (both `UnsignedInt`, RFC 9553) — already reflected as
`INDEX` in the `personalinfo`/`directory` rows of `20-correspondence.md`. No fix needed; this document
exists to confirm (not correct) what was already assumed from the IANA registry summary alone.
