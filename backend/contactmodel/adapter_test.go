package contactmodel

import "testing"

func TestDiagnosticRoundTrip(t *testing.T) {
	assertRoundTrip(t, fullDiagnostic())
}

// noopAdapter is a minimal stand-in used only to confirm, at compile time,
// that the Importer/Exporter interface signatures declared in adapter.go are
// implementable exactly as specified (no adapter logic is exercised here).
type noopAdapter struct{}

func (noopAdapter) Import(raw []byte) (*Record, []Diagnostic, error) {
	return &Record{}, nil, nil
}

func (noopAdapter) Export(r *Record) ([]byte, []Diagnostic, error) {
	return nil, nil, nil
}

var (
	_ Importer = noopAdapter{}
	_ Exporter = noopAdapter{}
)

func TestNoopAdapterSatisfiesInterfaces(t *testing.T) {
	var imp Importer = noopAdapter{}
	var exp Exporter = noopAdapter{}

	rec, diags, err := imp.Import([]byte(`{}`))
	if err != nil || rec == nil || diags != nil {
		t.Fatalf("unexpected Import result: rec=%v diags=%v err=%v", rec, diags, err)
	}

	data, diags, err := exp.Export(rec)
	if err != nil || data != nil || diags != nil {
		t.Fatalf("unexpected Export result: data=%v diags=%v err=%v", data, diags, err)
	}
}
