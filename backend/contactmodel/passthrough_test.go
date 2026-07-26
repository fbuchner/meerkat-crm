package contactmodel

import "testing"

func TestJCardPropRoundTrip(t *testing.T) {
	assertRoundTrip(t, fullJCardProp())
}

func TestPassthroughRoundTrip(t *testing.T) {
	assertRoundTrip(t, fullPassthrough())
}
