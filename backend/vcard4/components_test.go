package vcard4

import (
	"reflect"
	"testing"

	"github.com/emersion/go-vcard"
)

// --- escape / split / join round-trips ---

func TestEscapeUnescapeComponentRoundTrip(t *testing.T) {
	cases := []string{
		"",
		"plain",
		"has;semicolon",
		`has\backslash`,
		`edge\;case`,
		"trailing\\",
		";;;",
		`\;\;\;`,
	}
	for _, in := range cases {
		esc := escapeComponent(in)
		got := unescapeComponent(esc)
		if got != in {
			t.Errorf("escape/unescape round trip: input %q -> escaped %q -> unescaped %q, want %q", in, esc, got, in)
		}
	}
}

func TestEscapeComponentDoesNotEscapeCommaOrNewline(t *testing.T) {
	// Comma and newline escaping is intentionally left to go-vcard's own line
	// encoder (see the comment on escapeComponent); this codec only escapes
	// "\" and ";" so it doesn't collide with that downstream escaping.
	in := "a,b\nc"
	got := escapeComponent(in)
	if got != in {
		t.Errorf("escapeComponent(%q) = %q, want unchanged (comma/newline untouched)", in, got)
	}
}

func TestSplitComponentsHonorsEscapedSemicolon(t *testing.T) {
	// "1 Main St\; Suite 2" must not be split at the escaped semicolon.
	value := `1 Main St\; Suite 2;Anytown`
	got := splitComponents(value)
	want := []string{"1 Main St; Suite 2", "Anytown"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("splitComponents(%q) = %#v, want %#v", value, got, want)
	}
}

func TestSplitComponentsEmptyTrailingFields(t *testing.T) {
	value := ";;a;;"
	got := splitComponents(value)
	want := []string{"", "", "a", "", ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("splitComponents(%q) = %#v, want %#v", value, got, want)
	}
}

func TestJoinSplitComponentsRoundTrip(t *testing.T) {
	cases := [][]string{
		{"a", "b", "c"},
		{"", "", ""},
		{"has;semicolon", "plain", ""},
		{`back\slash`, "mid;dle", ""},
		{"a,b", "c\nd", "e"}, // commas/newlines pass through unescaped at this layer
	}
	for _, parts := range cases {
		joined := joinComponents(parts)
		got := splitComponents(joined)
		if !reflect.DeepEqual(got, parts) {
			t.Errorf("join/split round trip: parts %#v -> joined %q -> split %#v", parts, joined, got)
		}
	}
}

func TestSplitComponentsMinimumOneElement(t *testing.T) {
	got := splitComponents("")
	want := []string{""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("splitComponents(\"\") = %#v, want %#v", got, want)
	}
}

// --- N (7-component) assembly / disassembly round-trips ---

func TestNAssembleDisassembleRoundTrip(t *testing.T) {
	cases := []NComponents{
		{Family: "Public", Given: "John", Additional: "Quinlan", Prefix: "Mr.", Suffix: "Esq."},
		{
			Family: "Stevenson", Given: "John", Additional: "Philip,Paul",
			Prefix: "Dr.", Suffix: "Jr.,M.D.,A.C.P.", Surname2: "", Generation: "Jr.",
		},
		{}, // all-empty
		{Family: "Semi;colon", Given: `Back\slash`},
	}
	for _, n := range cases {
		value := assembleN(n)
		got := disassembleN(value)
		if got != n {
			t.Errorf("N round trip: %+v -> %q -> %+v", n, value, got)
		}
	}
}

func TestNAssembleFieldOrder(t *testing.T) {
	n := NComponents{
		Family: "F", Given: "G", Additional: "A", Prefix: "P",
		Suffix: "S", Surname2: "S2", Generation: "Gen",
	}
	got := assembleN(n)
	want := "F;G;A;P;S;S2;Gen"
	if got != want {
		t.Errorf("assembleN(%+v) = %q, want %q (Family;Given;Additional;Prefix;Suffix;Surname2;Generation)", n, got, want)
	}
}

func TestDisassembleNLegacyFiveComponent(t *testing.T) {
	// Legacy/short-form N values (e.g. the 5-component RFC 6350 form) must
	// leave the two RFC 9554 trailing components empty rather than erroring.
	got := disassembleN("Doe;J.;;;")
	want := NComponents{Family: "Doe", Given: "J."}
	if got != want {
		t.Errorf("disassembleN legacy form = %+v, want %+v", got, want)
	}
}

// --- ADR (18-component) assembly / disassembly round-trips ---

func TestAdrAssembleDisassembleRoundTrip(t *testing.T) {
	cases := []AdrComponents{
		{Street: "123 Main Street", Locality: "Any Town", Region: "CA", Code: "91921-1234", Country: "U.S.A"},
		{}, // all-empty
		{
			POBox: "PO 1", Ext: "Ext", Street: "St", Locality: "Loc", Region: "Reg",
			Code: "Code", Country: "Country", Room: "Rm", Apartment: "Apt", Floor: "Fl",
			Number: "123", StreetName: "Main Street", Building: "Bldg", Block: "Blk",
			Subdistrict: "Subd", District: "Dist", Landmark: "Landmark;here", Direction: `N\W`,
		},
	}
	for _, a := range cases {
		value := assembleAdr(a)
		got := disassembleAdr(value)
		if got != a {
			t.Errorf("ADR round trip: %+v -> %q -> %+v", a, value, got)
		}
	}
}

// TestAdrFieldOrderMatchesRFC9554Example verifies the 18-component order
// against the verbatim RFC 9554 example transcribed in
// docs/specs/rfc9554-vcard-extensions.md §3:
//
//	ADR;GEO="geo:12.3457,78.910":
//	  ;;123 Main Street;Any Town;CA;91921-1234;U.S.A
//	  ;;;;123;Main Street;;;;;;
func TestAdrFieldOrderMatchesRFC9554Example(t *testing.T) {
	a := AdrComponents{
		Street: "123 Main Street", Locality: "Any Town", Region: "CA",
		Code: "91921-1234", Country: "U.S.A", Number: "123", StreetName: "Main Street",
	}
	got := assembleAdr(a)
	want := ";;123 Main Street;Any Town;CA;91921-1234;U.S.A;;;;123;Main Street;;;;;;"
	if got != want {
		t.Errorf("assembleAdr(%+v) = %q, want %q", a, got, want)
	}

	// And the reverse: disassembling the verbatim example string must
	// populate exactly the fields the RFC example shows non-empty, at the
	// documented positions (11 = Number, 12 = StreetName).
	gotBack := disassembleAdr(want)
	if gotBack != a {
		t.Errorf("disassembleAdr(%q) = %+v, want %+v", want, gotBack, a)
	}
}

func TestDisassembleAdrLegacySevenComponent(t *testing.T) {
	got := disassembleAdr(";;123 Main Street;Any Town;CA;91921-1234;U.S.A.")
	want := AdrComponents{
		Street: "123 Main Street", Locality: "Any Town", Region: "CA",
		Code: "91921-1234", Country: "U.S.A.",
	}
	if got != want {
		t.Errorf("disassembleAdr legacy form = %+v, want %+v", got, want)
	}
}

// --- PROP-ID / group / X-ABLabel label handling ---

func TestGroupForIndex(t *testing.T) {
	cases := map[int]string{1: "item1", 2: "item2", 42: "item42"}
	for n, want := range cases {
		if got := groupForIndex(n); got != want {
			t.Errorf("groupForIndex(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestIsXABLabelField(t *testing.T) {
	if !isXABLabelField("X-ABLabel") {
		t.Error("isXABLabelField(\"X-ABLabel\") = false, want true")
	}
	if !isXABLabelField("x-ablabel") {
		t.Error("isXABLabelField should be case-insensitive")
	}
	if isXABLabelField(PropTel) {
		t.Errorf("isXABLabelField(%q) = true, want false", PropTel)
	}
}

func TestLabelFromXABLabel(t *testing.T) {
	// Hand-built field group, generalizing the item{n}.X-ABLabel pattern from
	// backend/carddav/vcard_mapper.go's addTypedField/labelFromGroup.
	labelFields := []*vcard.Field{
		{Group: "item1", Value: "Ski Cabin"},
		{Group: "item2", Value: "  Holiday Home  "},
	}

	if got := labelFromXABLabel("item1", labelFields); got != "Ski Cabin" {
		t.Errorf("labelFromXABLabel(item1) = %q, want %q", got, "Ski Cabin")
	}
	// Trimmed, and matched case-insensitively.
	if got := labelFromXABLabel("ITEM2", labelFields); got != "Holiday Home" {
		t.Errorf("labelFromXABLabel(ITEM2) = %q, want %q", got, "Holiday Home")
	}
	if got := labelFromXABLabel("item3", labelFields); got != "" {
		t.Errorf("labelFromXABLabel(item3) = %q, want \"\"", got)
	}
	if got := labelFromXABLabel("", labelFields); got != "" {
		t.Errorf("labelFromXABLabel(\"\") = %q, want \"\"", got)
	}
}

func TestNormalizeAppleLabel(t *testing.T) {
	cases := map[string]string{
		"_$!<Home>!$_":   "home",
		"_$!<Work>!$_":   "work",
		"_$!<Mobile>!$_": "cell",
		"Ski Cabin":      "Ski Cabin", // custom label passes through unchanged
		"":               "",
	}
	for in, want := range cases {
		if got := normalizeAppleLabel(in); got != want {
			t.Errorf("normalizeAppleLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPropIDGetSet(t *testing.T) {
	f := &vcard.Field{Value: "+15550000000"}
	if got := propID(f); got != "" {
		t.Errorf("propID on fresh field = %q, want \"\"", got)
	}
	setPropID(f, "p827")
	if got := propID(f); got != "p827" {
		t.Errorf("propID after setPropID = %q, want %q", got, "p827")
	}
	if got := f.Params.Get(ParamPropID); got != "p827" {
		t.Errorf("PROP-ID param = %q, want %q", got, "p827")
	}

	if got := propID(nil); got != "" {
		t.Errorf("propID(nil) = %q, want \"\"", got)
	}
}

// TestLabelHandlingHandBuiltFieldGroups exercises groupForIndex,
// isXABLabelField and labelFromXABLabel together against two hand-built
// X-ABLabel-style field groups, simulating how WP-40b's adapter will use
// them: allocate a group for a value field with a custom (non-standard)
// label, then resolve that label back out of the X-ABLabel property's field
// list on import.
func TestLabelHandlingHandBuiltFieldGroups(t *testing.T) {
	card := vcard.Card{}

	g1 := groupForIndex(1)
	emailField := &vcard.Field{Group: g1, Value: "c@school.example"}
	card.Add(PropEmail, emailField)
	card.Add(xabLabelProp, &vcard.Field{Group: g1, Value: "School"})

	g2 := groupForIndex(2)
	phoneField := &vcard.Field{Group: g2, Value: "+15550000000"}
	card.Add(PropTel, phoneField)
	card.Add(xabLabelProp, &vcard.Field{Group: g2, Value: "Ski Cabin"})

	for name := range card {
		if isXABLabelField(name) && name != xabLabelProp {
			t.Errorf("isXABLabelField matched unexpected property name %q", name)
		}
	}

	labelFields := card[xabLabelProp]
	if len(labelFields) != 2 {
		t.Fatalf("expected 2 X-ABLabel fields, got %d", len(labelFields))
	}
	if got := labelFromXABLabel(emailField.Group, labelFields); got != "School" {
		t.Errorf("email label = %q, want %q", got, "School")
	}
	if got := labelFromXABLabel(phoneField.Group, labelFields); got != "Ski Cabin" {
		t.Errorf("phone label = %q, want %q", got, "Ski Cabin")
	}
}
