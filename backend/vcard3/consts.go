// Package vcard3 implements the legacy vCard 3.0 format adapter (RFC 2426),
// part of the neutral-model conversion hub described in
// docs/fork-plan/00-overview.md. It is a deliberately separate, duplicated
// adapter tuned to 3.0 (see docs/fork-plan/30-adapters.md §30.C) — it shares
// no emit code and no imports with backend/vcard4.
//
// This file (consts.go, WP-50) declares the exported property and parameter
// name constants transcribed verbatim from docs/fork-plan/30-adapters.md
// §30.C's "consts.go" block, itself drawn from RFC 2426 §5 (the standard 3.0
// property set) plus the X- extension properties this table actually maps to
// in docs/fork-plan/20-correspondence.md. The string VALUES must exactly
// equal the RFC 2426 spellings.
//
// Naming convention mirrors vcard4/consts.go: properties are named
// Prop<Name>, parameters are named Param<Name>, using a Go-safe (hyphen-free)
// identifier for any name whose spelling contains a hyphen (e.g.
// PropSortString = "SORT-STRING").
//
// Deliberately absent: GENDER. RFC 2426 defines no GENDER property — see
// docs/specs/rfc2426-v3-baseline.md §5. Do not add one.
package vcard3

// Property names (RFC 2426 §5 standard set, plus the X- extensions this
// adapter's correspondence rows require).
const (
	PropBegin          = "BEGIN"
	PropEnd            = "END"
	PropSource         = "SOURCE"
	PropName           = "NAME"
	PropFN             = "FN"
	PropN              = "N"
	PropNickname       = "NICKNAME"
	PropPhoto          = "PHOTO"
	PropBday           = "BDAY"
	PropAdr            = "ADR"
	PropLabel          = "LABEL"
	PropTel            = "TEL"
	PropEmail          = "EMAIL"
	PropMailer         = "MAILER"
	PropTz             = "TZ"
	PropGeo            = "GEO"
	PropTitle          = "TITLE"
	PropRole           = "ROLE"
	PropLogo           = "LOGO"
	PropAgent          = "AGENT"
	PropOrg            = "ORG"
	PropCategories     = "CATEGORIES"
	PropNote           = "NOTE"
	PropProdid         = "PRODID"
	PropRev            = "REV"
	PropSortString     = "SORT-STRING"
	PropSound          = "SOUND"
	PropUID            = "UID"
	PropURL            = "URL"
	PropVersion        = "VERSION"
	PropClass          = "CLASS"
	PropKey            = "KEY"
	PropImpp           = "IMPP"
	PropCaluri         = "CALURI"
	PropFburl          = "FBURL"
	PropCaladruri      = "CALADRURI"
	PropXSocialProfile = "X-SOCIALPROFILE"
	PropXServiceType   = "X-SERVICE-TYPE"
	PropXUsername      = "X-USERNAME"
	PropXAnniversary   = "X-ANNIVERSARY"

	// propXABLabel is the (non-standard, Apple-originated but ubiquitous)
	// property used to carry a human-readable custom label for a grouped
	// value. Not part of RFC 2426 §5's registered set, so it is not named
	// Prop* like the rest — kept lower-case/unexported-in-spirit here but
	// still a package-level const since components.go/adapter.go both need
	// it and this package has no internal sub-file boundary to hide it
	// behind. (Mirrors vcard4/components.go's xabLabelProp.)
	propXABLabel = "X-ABLabel"
)

// Parameter names (RFC 2426 §5 standard set).
const (
	ParamType     = "TYPE"
	ParamValue    = "VALUE"
	ParamEncoding = "ENCODING"
	ParamCharset  = "CHARSET"
	ParamLanguage = "LANGUAGE"
)
