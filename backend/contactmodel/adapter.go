package contactmodel

// Diagnostic is a non-fatal data-handling event (see degradation policy).
type Diagnostic struct {
	Severity string // "warn" | "info"
	Concept  string // correspondence concept_id, or ""
	Message  string
}

// Importer parses one serialized format into the neutral model.
// It MUST NOT return an error for unmappable/unknown data — it preserves it
// (passthrough) and appends a Diagnostic. errors are reserved for malformed input
// (bytes that are not valid instances of the format at all).
type Importer interface {
	Import(raw []byte) (*Record, []Diagnostic, error)
}

// Exporter renders the neutral model into one serialized format.
// Same rule: never error on a field that has no home in the target format —
// drop-with-warning or passthrough, and append a Diagnostic.
type Exporter interface {
	Export(r *Record) ([]byte, []Diagnostic, error)
}
