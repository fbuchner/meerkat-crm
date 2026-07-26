package contactmodel

import "testing"

func TestCRMEnvelopeRoundTrip(t *testing.T) {
	assertRoundTrip(t, fullCRMEnvelope())
}
