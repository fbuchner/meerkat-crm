## RFC 8605 — CONTACT-URI (confirmed reference)

Purpose (§2.1): a URI for contacting the entity when identifying information is otherwise redacted
(e.g. a privacy-preserving form/alias). Value: single URI. Cardinality: `*` (zero or more — confirms
the neutral model's `Card.Links` slice with `Kind: "contact"` is correctly plural, not singular).
Parameters: `PREF` ("MUST be used" to mark the most-preferred when more than one is present).

**Constraint (spec fact, not strictly enforced by us):** "At least one 'mailto', 'http', or 'https' URI
value MUST be provided" when the property is used at all. Per this plan's degradation policy (0.5),
this is documented as a spec fact but not hard-enforced on import/export — an out-of-scheme URI is
preserved via passthrough rather than rejected.

```
CONTACT-URI:https://contact.example.com
CONTACT-URI;PREF=1:mailto:contact@example.com
```

No corrections needed to `20-correspondence.md`'s `contacturi` row — confirms what was already assumed.
