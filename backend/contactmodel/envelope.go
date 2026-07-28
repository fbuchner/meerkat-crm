package contactmodel

// CRMEnvelope holds Mycorrhizal-specific data that is NOT part of any contact-exchange
// standard. Format adapters MUST ignore it entirely.
type CRMEnvelope struct {
	Circles            []string          `json:"circles,omitempty"`
	HowWeMet           string            `json:"how_we_met,omitempty"`
	FoodPreference     string            `json:"food_preference,omitempty"`
	WorkInformation    string            `json:"work_information,omitempty"`
	ContactInformation string            `json:"contact_information,omitempty"`
	CustomFields       map[string]string `json:"custom_fields,omitempty"`
	// Reminders/Activities/Relationships remain separate GORM tables keyed by contact ID;
	// they are NOT embedded here.
}
