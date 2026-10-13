package carddav

import (
	"io"
	"mime/quotedprintable"
	"strings"

	"golang.org/x/text/encoding/ianaindex"
)

// NormalizeLegacyVCF rewrites the vCard 2.1 constructs go-vcard misreads, so a file
// exported by an older phone imports with all its fields:
//   - bare parameters ("TEL;CELL:...") become TYPE=CELL, or ENCODING=... for an
//     encoding name. go-vcard reads "CELL" as a parameter name and moves the value
//     into it, leaving the property empty.
//   - QUOTED-PRINTABLE values, including their "=" soft line breaks, are decoded.
//   - values in a CHARSET other than UTF-8 are converted to UTF-8.
//
// Lines without bare parameters, ENCODING=QUOTED-PRINTABLE or CHARSET are emitted
// unchanged (unfolded), so vCard 3.0 and 4.0 input decodes as before.
func NormalizeLegacyVCF(data []byte) []byte {
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")

	var out strings.Builder
	var cur string
	var curQP bool
	flush := func() {
		if cur != "" {
			out.WriteString(normalizeLegacyLine(cur))
			out.WriteString("\r\n")
		}
	}
	for _, line := range lines {
		switch {
		case curQP && strings.HasSuffix(cur, "="):
			// Quoted-printable soft line break: the value continues verbatim.
			cur = cur[:len(cur)-1] + line
		case line != "" && (line[0] == ' ' || line[0] == '\t'):
			cur += line[1:]
		default:
			flush()
			cur = line
			curQP = isQuotedPrintable(cur)
		}
	}
	flush()
	return []byte(out.String())
}

// isQuotedPrintable reports whether a property line declares a quoted-printable
// value, either as ENCODING=QUOTED-PRINTABLE or as the bare vCard 2.1 parameter.
func isQuotedPrintable(line string) bool {
	colon := indexUnquoted(line, ':')
	if colon < 0 {
		return false
	}
	for _, p := range splitUnquoted(line[:colon], ';')[1:] {
		_, v, _ := strings.Cut(p, "=")
		if strings.EqualFold(v, "QUOTED-PRINTABLE") || strings.EqualFold(p, "QUOTED-PRINTABLE") {
			return true
		}
	}
	return false
}

// normalizeLegacyLine applies the NormalizeLegacyVCF rewrites to one unfolded line.
func normalizeLegacyLine(line string) string {
	colon := indexUnquoted(line, ':')
	if colon < 0 {
		return line
	}
	parts := splitUnquoted(line[:colon], ';')
	if len(parts) == 1 {
		return line
	}
	value := line[colon+1:]

	params := []string{parts[0]}
	var qp, changed bool
	var charset string
	for _, p := range parts[1:] {
		k, v, hasValue := strings.Cut(p, "=")
		if !hasValue {
			changed = true
			if p == "" {
				continue
			}
			switch strings.ToUpper(p) {
			case "QUOTED-PRINTABLE", "BASE64", "8BIT", "7BIT":
				k, v = "ENCODING", p
			default:
				k, v = "TYPE", p
			}
		}
		switch {
		case strings.EqualFold(k, "ENCODING") && strings.EqualFold(v, "QUOTED-PRINTABLE"):
			qp, changed = true, true
			continue
		case strings.EqualFold(k, "CHARSET"):
			charset, changed = v, true
			continue
		}
		params = append(params, k+"="+v)
	}
	if !changed {
		return line
	}

	if qp {
		if decoded, err := io.ReadAll(quotedprintable.NewReader(strings.NewReader(value))); err == nil {
			value = string(decoded)
		}
	}
	if charset != "" {
		if enc, err := ianaindex.IANA.Encoding(charset); err == nil && enc != nil {
			if converted, err := enc.NewDecoder().String(value); err == nil {
				value = converted
			}
		}
	}
	if qp {
		// A decoded value may contain line breaks, which a vCard line must escape.
		value = strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\n", `\n`)
	}
	return strings.Join(params, ";") + ":" + value
}

// indexUnquoted returns the index of the first c outside double quotes, or -1.
func indexUnquoted(s string, c byte) int {
	quoted := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			quoted = !quoted
		case c:
			if !quoted {
				return i
			}
		}
	}
	return -1
}

// splitUnquoted splits s on every sep outside double quotes.
func splitUnquoted(s string, sep byte) []string {
	var parts []string
	for {
		i := indexUnquoted(s, sep)
		if i < 0 {
			return append(parts, s)
		}
		parts = append(parts, s[:i])
		s = s[i+1:]
	}
}
