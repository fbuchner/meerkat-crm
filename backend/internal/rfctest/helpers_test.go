package rfctest

import (
	"fmt"
	"strings"
	"testing"

	"meerkat/contactmodel"
)

// These are self-tests of the helpers themselves (LoadFixture, AssertVCardLine,
// AssertJSONPointer, NeutralFromJSON) — not tests of fixture content, which is
// exercised by the jscontact/vcard4/vcard3 adapter suites in later WPs.
//
// Happy-path behavior is tested by calling the public, *testing.T-typed
// functions directly. The "reports a clear failure, does not panic" paths
// are tested by calling the internal tReporter-typed implementations with a
// fakeT below — calling AssertVCardLine/AssertJSONPointer with an
// intentionally-mismatching input against the real *testing.T would mark
// this test (and the whole package run) as failed, which is exactly the
// behavior under test, not a bug to observe from outside.

// fakeT is a minimal tReporter that records whether a failure was reported,
// without touching a real *testing.T's internal state.
type fakeT struct {
	failed   bool
	messages []string
}

func (f *fakeT) Helper() {}

func (f *fakeT) Errorf(format string, args ...any) {
	f.failed = true
	f.messages = append(f.messages, fmt.Sprintf(format, args...))
}

func (f *fakeT) Fatalf(format string, args ...any) {
	f.failed = true
	f.messages = append(f.messages, fmt.Sprintf(format, args...))
	// Real testing.T.Fatalf stops the goroutine via runtime.Goexit. Every
	// call site in helpers.go already does `return` immediately after
	// Fatalf, so a plain mark-and-continue is equivalent here.
}

func TestLoadFixture(t *testing.T) {
	data := LoadFixture("rfc6350-baseline.v4.vcf")
	if len(data) == 0 {
		t.Fatalf("LoadFixture returned empty data")
	}
	if !strings.Contains(string(data), "BEGIN:VCARD") {
		t.Errorf("LoadFixture(%q) doesn't look like a vCard: %s", "rfc6350-baseline.v4.vcf", data)
	}
	if !strings.Contains(string(data), "urn:uuid:4fbe8971-0bc3-424c-9c26-36c3e1eff6b1") {
		t.Errorf("LoadFixture(%q) missing expected UID content", "rfc6350-baseline.v4.vcf")
	}
}

func TestLoadFixture_missingPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("LoadFixture(missing) should have panicked")
		}
	}()
	LoadFixture("does-not-exist.vcf")
}

const tinyVCard = "BEGIN:VCARD\r\n" +
	"VERSION:4.0\r\n" +
	"UID:helper-self-test\r\n" +
	"FN:Helper Test\r\n" +
	"TEL;TYPE=cell;PREF=1:+15551234567\r\n" +
	"END:VCARD\r\n"

func TestAssertVCardLine_matches(t *testing.T) {
	AssertVCardLine(t, []byte(tinyVCard), "TEL", map[string]string{"TYPE": "cell"}, "+15551234567")
}

func TestAssertVCardLine_subsetParamsIgnoreExtras(t *testing.T) {
	// wantParams only checks PREF; TYPE=cell is an "extra" param on the field
	// and must not cause a mismatch (subset-match rule).
	AssertVCardLine(t, []byte(tinyVCard), "TEL", map[string]string{"PREF": "1"}, "+15551234567")
}

func TestAssertVCardLine_reportsMismatchWithoutPanicking(t *testing.T) {
	ft := &fakeT{}
	assertVCardLine(ft, []byte(tinyVCard), "TEL", map[string]string{"TYPE": "fax"}, "+15551234567")
	if !ft.failed {
		t.Errorf("expected assertVCardLine to report a failure for a non-matching TYPE param")
	}
}

func TestAssertVCardLine_missingPropertyReportsError(t *testing.T) {
	ft := &fakeT{}
	assertVCardLine(ft, []byte(tinyVCard), "EMAIL", nil, "nobody@example.com")
	if !ft.failed {
		t.Errorf("expected assertVCardLine to report a failure for a missing property")
	}
}

func TestAssertVCardLine_unparsableReportsFatal(t *testing.T) {
	ft := &fakeT{}
	assertVCardLine(ft, []byte("not a vcard at all"), "TEL", nil, "x")
	if !ft.failed {
		t.Errorf("expected assertVCardLine to report a failure for unparsable input")
	}
}

const tinyJSON = `{
  "phones": {
    "k1": {"number": "+15551234567", "pref": 1}
  },
  "keywords": {"family": true, "work": true},
  "nested": [1, 2, {"three": 3}]
}`

func TestAssertJSONPointer_matchesString(t *testing.T) {
	AssertJSONPointer(t, []byte(tinyJSON), "/phones/k1/number", "+15551234567")
}

func TestAssertJSONPointer_matchesNumber(t *testing.T) {
	// want as plain int; resolved JSON number decodes as float64 — helper must
	// coerce for the comparison to succeed.
	AssertJSONPointer(t, []byte(tinyJSON), "/phones/k1/pref", 1)
}

func TestAssertJSONPointer_matchesArrayIndex(t *testing.T) {
	AssertJSONPointer(t, []byte(tinyJSON), "/nested/2/three", 3)
}

func TestAssertJSONPointer_matchesBool(t *testing.T) {
	AssertJSONPointer(t, []byte(tinyJSON), "/keywords/family", true)
}

func TestAssertJSONPointer_unresolvableReportsError(t *testing.T) {
	ft := &fakeT{}
	assertJSONPointer(ft, []byte(tinyJSON), "/phones/does-not-exist/number", "x")
	if !ft.failed {
		t.Errorf("expected assertJSONPointer to report a failure for an unresolvable pointer")
	}
}

func TestAssertJSONPointer_mismatchReportsError(t *testing.T) {
	ft := &fakeT{}
	assertJSONPointer(ft, []byte(tinyJSON), "/phones/k1/number", "+19998887777")
	if !ft.failed {
		t.Errorf("expected assertJSONPointer to report a failure for a value mismatch")
	}
}

func TestAssertJSONPointer_invalidJSONReportsFatal(t *testing.T) {
	ft := &fakeT{}
	assertJSONPointer(ft, []byte("{not valid json"), "/x", "y")
	if !ft.failed {
		t.Errorf("expected assertJSONPointer to report a failure for invalid JSON input")
	}
}

func TestNeutralFromJSON_intoContactmodelRecord(t *testing.T) {
	var rec contactmodel.Record
	NeutralFromJSON(t, `{"card": {"uid": "abc-123", "kind": "individual"}}`, &rec)
	if rec.Card.UID != "abc-123" {
		t.Errorf("rec.Card.UID = %q, want %q", rec.Card.UID, "abc-123")
	}
	if rec.Card.Kind != "individual" {
		t.Errorf("rec.Card.Kind = %q, want %q", rec.Card.Kind, "individual")
	}
}

func TestNeutralFromJSON_badJSONReportsError(t *testing.T) {
	ft := &fakeT{}
	var rec contactmodel.Record
	neutralFromJSON(ft, `{not valid json`, &rec)
	if !ft.failed {
		t.Errorf("expected neutralFromJSON to report a failure for invalid JSON")
	}
}
