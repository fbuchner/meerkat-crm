package contactmodel

import "encoding/json"

// Passthrough preserves data with no neutral/target home so nothing is silently lost.
type Passthrough struct {
	// Unknown vCard properties captured on vCard import (RFC 9555 "vCardProps" shape).
	VCard []JCardProp `json:"vCardProps,omitempty"`
	// Unknown JSContact properties captured on JSContact import, keyed by JSON pointer.
	JSContact map[string]json.RawMessage `json:"jsContactProps,omitempty"`
}

// JCardProp is one jCard property array: [name, params, valuetype, value...].
type JCardProp struct {
	Name   string          `json:"name"`
	Params map[string]any  `json:"params,omitempty"`
	Type   string          `json:"type"` // jCard value type, e.g. "text"
	Value  json.RawMessage `json:"value"`
}
