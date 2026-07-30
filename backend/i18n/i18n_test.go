package i18n

import "testing"

// TestNormalizeSupportedLanguage is the regression test for backlog item 12
// (docs/fork-plan/95-backlog-and-priorities.md): IsValidLanguage used to
// normalize via normalizeLanguage, which falls back to DefaultLanguage for
// any unrecognized input, so it returned true for literally any string.
// NormalizeSupportedLanguage must genuinely reject empty/unsupported input.
func TestNormalizeSupportedLanguage(t *testing.T) {
	cases := []struct {
		name     string
		lang     string
		wantNorm string
		wantOK   bool
	}{
		{"exact supported code", "de", "de", true},
		{"case-insensitive", "DE", "de", true},
		{"BCP-47 region subtag stripped", "en-US", "en", true},
		{"BCP-47 region subtag stripped, uppercase", "DE-AT", "de", true},
		{"empty input rejected", "", "", false},
		{"garbage rejected", "xx", "", false},
		{"garbage with region subtag rejected", "xx-YY", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotNorm, gotOK := NormalizeSupportedLanguage(tc.lang)
			if gotOK != tc.wantOK {
				t.Fatalf("NormalizeSupportedLanguage(%q) ok = %v, want %v", tc.lang, gotOK, tc.wantOK)
			}
			if gotNorm != tc.wantNorm {
				t.Fatalf("NormalizeSupportedLanguage(%q) = %q, want %q", tc.lang, gotNorm, tc.wantNorm)
			}
		})
	}
}

// TestIsValidLanguage_RejectsGarbage pins down the actual bug: before the
// fix, this returned true for any input at all.
func TestIsValidLanguage_RejectsGarbage(t *testing.T) {
	if IsValidLanguage("this-is-not-a-real-language-code") {
		t.Fatal("IsValidLanguage should reject an unrecognized code, not silently accept it as English")
	}
	if IsValidLanguage("") {
		t.Fatal("IsValidLanguage should reject an empty code")
	}
	if !IsValidLanguage("fr") {
		t.Fatal("IsValidLanguage should accept a genuinely supported code")
	}
}

// TestNormalizeLanguage_StillFallsBackForDisplayLookup confirms the
// unexported normalizeLanguage (used only by T()) keeps its own, deliberately
// different fallback-to-English behavior -- that one is correct for display
// lookups and was never the bug; only the validation path (IsValidLanguage/
// NormalizeSupportedLanguage) needed to stop reusing it.
func TestNormalizeLanguage_StillFallsBackForDisplayLookup(t *testing.T) {
	if got := normalizeLanguage("this-is-not-a-real-language-code"); got != DefaultLanguage {
		t.Fatalf("normalizeLanguage(garbage) = %q, want fallback %q", got, DefaultLanguage)
	}
	if got := normalizeLanguage(""); got != DefaultLanguage {
		t.Fatalf("normalizeLanguage(\"\") = %q, want fallback %q", got, DefaultLanguage)
	}
}
