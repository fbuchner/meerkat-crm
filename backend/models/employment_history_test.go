// backend/models/employment_history_test.go
package models

import (
	"testing"
)

func TestPartialDateStart(t *testing.T) {
	cases := []struct {
		input   string
		wantOK  bool
		wantYMD string // formatted "2006-01-02" if wantOK
	}{
		{"2020", true, "2020-01-01"},
		{"2020-05", true, "2020-05-01"},
		{"2020-05-03", true, "2020-05-03"},
		{"", false, ""},
		{"not-a-date", false, ""},
		{"2020-13", false, ""},
	}

	for _, tc := range cases {
		got, ok := PartialDateStart(tc.input)
		if ok != tc.wantOK {
			t.Errorf("PartialDateStart(%q) ok = %v, want %v", tc.input, ok, tc.wantOK)
			continue
		}
		if ok && got.Format("2006-01-02") != tc.wantYMD {
			t.Errorf("PartialDateStart(%q) = %v, want %v", tc.input, got.Format("2006-01-02"), tc.wantYMD)
		}
	}
}

func TestPartialDateEnd(t *testing.T) {
	for in, want := range map[string]string{"2020": "2020-12-31", "2020-02": "2020-02-29", "2020-05-03": "2020-05-03"} {
		got, ok := PartialDateEnd(in)
		if !ok || got.Format("2006-01-02") != want {
			t.Errorf("PartialDateEnd(%q) = %v, want %s", in, got, want)
		}
	}
}
