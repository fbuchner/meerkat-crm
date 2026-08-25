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
