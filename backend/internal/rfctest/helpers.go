// Package rfctest provides shared test fixtures and assertion helpers for the
// jscontact/vcard4/vcard3 adapter test suites (WP-30b/40b/50). It intentionally
// depends on nothing but the standard library, contactmodel (for
// NeutralFromJSON's target type), and github.com/emersion/go-vcard (for
// AssertVCardLine's parsing) — no gorm/gin/models imports.
package rfctest

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	vcard "github.com/emersion/go-vcard"
)

// fixturesFS embeds the fixtures/ directory so LoadFixture works regardless of
// the caller's working directory (e.g. when invoked from another package's
// _test.go via `go test ./...` run from backend/).
//
//go:embed fixtures
var fixturesFS embed.FS

// LoadFixture reads a file from fixtures/ by name (e.g. "rfc6350-baseline.v4.vcf").
// It panics on error rather than taking a *testing.T: a missing fixture is a
// programmer/setup error, not a test assertion failure, and LoadFixture is
// commonly used outside of a test body (e.g. in table-driven test data setup).
func LoadFixture(name string) []byte {
	data, err := fixturesFS.ReadFile("fixtures/" + name)
	if err != nil {
		panic(fmt.Sprintf("rfctest.LoadFixture(%q): %v", name, err))
	}
	return data
}

// tReporter is the subset of *testing.T's API the internal assert
// implementations need. AssertVCardLine/AssertJSONPointer/NeutralFromJSON take
// a concrete *testing.T (which satisfies tReporter) as their public API per
// 40-testing.md §40.4; routing through this interface internally lets
// helpers_test.go self-test the "reports a clear failure, does not panic"
// paths with a fake reporter instead of failing the real test binary.
type tReporter interface {
	Helper()
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}

// AssertVCardLine parses raw as a single vCard and asserts that at least one
// field named propName has a value equal to wantValue and parameters that are
// a superset of wantParams (every key in wantParams must be present on the
// field with a matching value among that parameter's values; extra params on
// the field, or extra fields with the same name, are fine).
//
// On mismatch it calls t.Errorf with a message describing what was found
// instead; on a fixture that fails to parse as a vCard at all it calls
// t.Fatalf. It never panics the test binary.
func AssertVCardLine(t *testing.T, raw []byte, propName string, wantParams map[string]string, wantValue string) {
	t.Helper()
	assertVCardLine(t, raw, propName, wantParams, wantValue)
}

func assertVCardLine(t tReporter, raw []byte, propName string, wantParams map[string]string, wantValue string) {
	t.Helper()

	dec := vcard.NewDecoder(bytes.NewReader(raw))
	card, err := dec.Decode()
	if err != nil {
		t.Fatalf("rfctest.AssertVCardLine: failed to parse vCard: %v", err)
		return
	}

	key := strings.ToUpper(propName)
	fields := card[key]
	if len(fields) == 0 {
		t.Errorf("rfctest.AssertVCardLine: no %s field found in vCard", propName)
		return
	}

	var candidates []string
	for _, f := range fields {
		if vcardFieldMatches(f, wantParams, wantValue) {
			return
		}
		candidates = append(candidates, fmt.Sprintf("value=%q params=%v", f.Value, map[string][]string(f.Params)))
	}

	t.Errorf("rfctest.AssertVCardLine: no %s field matched want(value=%q, params=%v); found: %s",
		propName, wantValue, wantParams, strings.Join(candidates, "; "))
}

func vcardFieldMatches(f *vcard.Field, wantParams map[string]string, wantValue string) bool {
	if f.Value != wantValue {
		return false
	}
	for k, v := range wantParams {
		if !vcardParamHasValue(f, k, v) {
			return false
		}
	}
	return true
}

// vcardParamHasValue reports whether field f's parameter k (case-insensitive
// name) contains value v among its (possibly comma-split, e.g. TYPE=A,B,C)
// values.
func vcardParamHasValue(f *vcard.Field, k, v string) bool {
	for _, got := range f.Params[strings.ToUpper(k)] {
		if got == v {
			return true
		}
	}
	return false
}

// AssertJSONPointer unmarshals raw as generic JSON and resolves an RFC 6901
// JSON pointer (e.g. "/phones/k1/number") against it, asserting the resolved
// value equals want. want may be any Go value produced by json.Unmarshal into
// `any` (string, bool, nil, []any, map[string]any) or a numeric literal (int,
// float64, ...) which is compared numerically against a decoded JSON number.
//
// On mismatch or an unresolvable pointer it calls t.Errorf with a clear
// message; it never panics the test binary.
func AssertJSONPointer(t *testing.T, raw []byte, pointer string, want any) {
	t.Helper()
	assertJSONPointer(t, raw, pointer, want)
}

func assertJSONPointer(t tReporter, raw []byte, pointer string, want any) {
	t.Helper()

	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("rfctest.AssertJSONPointer: raw is not valid JSON: %v", err)
		return
	}

	got, err := resolveJSONPointer(doc, pointer)
	if err != nil {
		t.Errorf("rfctest.AssertJSONPointer: %v", err)
		return
	}

	if !jsonValuesEqual(got, want) {
		t.Errorf("rfctest.AssertJSONPointer: pointer %q = %#v, want %#v", pointer, got, want)
	}
}

// resolveJSONPointer implements RFC 6901 pointer resolution against a generic
// JSON document (map[string]any / []any / scalars, as produced by
// json.Unmarshal into `any`).
func resolveJSONPointer(doc any, pointer string) (any, error) {
	if pointer == "" {
		return doc, nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil, fmt.Errorf("invalid JSON pointer %q: must be empty or start with '/'", pointer)
	}

	cur := doc
	tokens := strings.Split(pointer[1:], "/")
	for i, raw := range tokens {
		tok := unescapeJSONPointerToken(raw)
		switch v := cur.(type) {
		case map[string]any:
			next, ok := v[tok]
			if !ok {
				return nil, fmt.Errorf("pointer %q: no key %q at segment %d", pointer, tok, i)
			}
			cur = next
		case []any:
			idx, err := strconv.Atoi(tok)
			if err != nil || idx < 0 || idx >= len(v) {
				return nil, fmt.Errorf("pointer %q: invalid array index %q at segment %d (len %d)", pointer, tok, i, len(v))
			}
			cur = v[idx]
		default:
			return nil, fmt.Errorf("pointer %q: cannot descend into %T at segment %d (%q)", pointer, cur, i, tok)
		}
	}
	return cur, nil
}

func unescapeJSONPointerToken(tok string) string {
	tok = strings.ReplaceAll(tok, "~1", "/")
	tok = strings.ReplaceAll(tok, "~0", "~")
	return tok
}

// jsonValuesEqual compares a JSON-decoded value (got) against an
// expected Go value (want), treating any numeric want (int, int64, float64,
// ...) as comparable-by-value against a decoded JSON number (float64).
func jsonValuesEqual(got, want any) bool {
	if reflect.DeepEqual(got, want) {
		return true
	}
	gf, gok := toFloat64(got)
	wf, wok := toFloat64(want)
	if gok && wok {
		return gf == wf
	}
	return false
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	default:
		return 0, false
	}
}

// NeutralFromJSON unmarshals jsonStr into target (typically a pointer to a
// contactmodel.Record or one of its fields), failing the test clearly on
// error. It exists purely to reduce test boilerplate in adapter test suites
// that build a Record/Card literal from a JSON fragment.
func NeutralFromJSON(t *testing.T, jsonStr string, target any) {
	t.Helper()
	neutralFromJSON(t, jsonStr, target)
}

func neutralFromJSON(t tReporter, jsonStr string, target any) {
	t.Helper()
	if err := json.Unmarshal([]byte(jsonStr), target); err != nil {
		t.Fatalf("rfctest.NeutralFromJSON: %v", err)
	}
}
