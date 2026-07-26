// Package correspondence is the typed accessor for the correspondence oracle:
// the neutral ⇄ JSContact ⇄ vCard4 ⇄ vCard3 mapping table authored in
// docs/fork-plan/20-correspondence.md and materialized as testdata/correspondence.tsv.
// This package contains no mapping logic itself — it only parses and indexes
// the table so adapters and tests can consume it.
package correspondence

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"strings"
)

//go:embed testdata/correspondence.tsv
var correspondenceTSV []byte

// Row is one row of the correspondence table (see 20.1 for column meanings).
type Row struct {
	ConceptID, NeutralPath, JSPtr                        string
	V4Prop, V4Params, V3Prop, V3Params, Transform, Notes string
}

// Load parses testdata/correspondence.tsv (embedded) into []Row, in file order.
func Load() []Row {
	scanner := bufio.NewScanner(bytes.NewReader(correspondenceTSV))
	// Lines can be long (the notes column), so grow the buffer generously.
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var rows []Row
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first {
			// header row — skip
			first = false
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 9 {
			panic(fmt.Sprintf("correspondence: malformed row (want 9 tab-separated fields, got %d): %q", len(fields), line))
		}
		rows = append(rows, Row{
			ConceptID:   fields[0],
			NeutralPath: fields[1],
			JSPtr:       fields[2],
			V4Prop:      fields[3],
			V4Params:    fields[4],
			V3Prop:      fields[5],
			V3Params:    fields[6],
			Transform:   fields[7],
			Notes:       fields[8],
		})
	}
	if err := scanner.Err(); err != nil {
		panic(fmt.Sprintf("correspondence: error scanning testdata/correspondence.tsv: %v", err))
	}
	return rows
}

// ByConcept indexes Load()'s rows by ConceptID. It panics on a duplicate
// concept_id — this is a hard invariant of the oracle, not a soft warning.
func ByConcept() map[string]Row {
	rows := Load()
	m := make(map[string]Row, len(rows))
	for _, r := range rows {
		if _, dup := m[r.ConceptID]; dup {
			panic(fmt.Sprintf("correspondence: duplicate concept_id %q", r.ConceptID))
		}
		m[r.ConceptID] = r
	}
	return m
}
