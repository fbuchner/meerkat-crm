package contactmodel

// CRMEnvelope holds Mycorrhizal-specific data that is NOT part of any contact-exchange
// standard. Format adapters MUST ignore it entirely.
type CRMEnvelope struct {
	// Kind is the envelope-side entity kind: conventionally individual|pet|
	// animal. Distinct from Card.Kind (model.go — the standard vCard/
	// JSContact KIND: individual|group|org|location|application|device,
	// which has no pet/animal value) — WP-82, docs/fork-plan/91-envelope-
	// data-model.md §91.1. Unenforced at the API boundary, same as Card.Kind
	// and every other CRMEnvelope field: unrecognized values are accepted
	// and preserved, not rejected (see controllers/
	// contact_controller_validation_test.go's header for why).
	Kind               string            `json:"kind,omitempty"`
	Circles            []string          `json:"circles,omitempty"`
	HowWeMet           string            `json:"how_we_met,omitempty"`
	WorkInformation    string            `json:"work_information,omitempty"`
	ContactInformation string            `json:"contact_information,omitempty"`
	CustomFields       map[string]string `json:"custom_fields,omitempty"`
	// Reminders/Activities/Relationships remain separate GORM tables keyed by contact ID;
	// they are NOT embedded here.
}
