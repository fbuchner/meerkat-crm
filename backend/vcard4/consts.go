// Package vcard4 implements the vCard 4.0 format adapter (RFC 6350, plus the
// RFC 9554/6474/6715/8605 extensions), part of the neutral-model conversion
// hub described in docs/fork-plan/00-overview.md.
//
// This file (consts.go, WP-40a) declares the exported property and parameter
// name constants transcribed verbatim from docs/fork-plan/30-adapters.md
// §30.B's "consts.go property names" block, which in turn comes from the IANA
// vcard-elements registry. The string VALUES must exactly equal the IANA
// spellings: WP-20's correspondence test hardcodes the same list and checks
// against it mechanically.
//
// Naming convention: properties are named Prop<Name>, parameters are named
// Param<Name>, using a Go-safe (hyphen-free) identifier for any name whose
// IANA spelling contains a hyphen (e.g. PropOrgDirectory = "ORG-DIRECTORY").
package vcard4

// Property names (IANA vcard-elements registry).
const (
	PropSource        = "SOURCE"
	PropKind          = "KIND"
	PropXML           = "XML"
	PropFN            = "FN"
	PropN             = "N"
	PropNickname      = "NICKNAME"
	PropPhoto         = "PHOTO"
	PropBday          = "BDAY"
	PropAnniversary   = "ANNIVERSARY"
	PropGender        = "GENDER"
	PropAdr           = "ADR"
	PropTel           = "TEL"
	PropEmail         = "EMAIL"
	PropImpp          = "IMPP"
	PropLang          = "LANG"
	PropTz            = "TZ"
	PropGeo           = "GEO"
	PropTitle         = "TITLE"
	PropRole          = "ROLE"
	PropLogo          = "LOGO"
	PropOrg           = "ORG"
	PropMember        = "MEMBER"
	PropRelated       = "RELATED"
	PropCategories    = "CATEGORIES"
	PropNote          = "NOTE"
	PropProdid        = "PRODID"
	PropRev           = "REV"
	PropSound         = "SOUND"
	PropUID           = "UID"
	PropClientPIDMap  = "CLIENTPIDMAP"
	PropURL           = "URL"
	PropVersion       = "VERSION"
	PropKey           = "KEY"
	PropFburl         = "FBURL"
	PropCaladruri     = "CALADRURI"
	PropCaluri        = "CALURI"
	PropBirthplace    = "BIRTHPLACE"
	PropDeathplace    = "DEATHPLACE"
	PropDeathdate     = "DEATHDATE"
	PropExpertise     = "EXPERTISE"
	PropHobby         = "HOBBY"
	PropInterest      = "INTEREST"
	PropOrgDirectory  = "ORG-DIRECTORY"
	PropContactURI    = "CONTACT-URI"
	PropCreated       = "CREATED"
	PropGramgender    = "GRAMGENDER"
	PropLanguage      = "LANGUAGE"
	PropPronouns      = "PRONOUNS"
	PropSocialprofile = "SOCIALPROFILE"
	PropJSProp        = "JSPROP"
)

// Parameter names (IANA vcard-elements registry).
const (
	ParamLanguage    = "LANGUAGE"
	ParamValue       = "VALUE"
	ParamPref        = "PREF"
	ParamAltid       = "ALTID"
	ParamPid         = "PID"
	ParamType        = "TYPE"
	ParamMediatype   = "MEDIATYPE"
	ParamCalscale    = "CALSCALE"
	ParamSortAs      = "SORT-AS"
	ParamGeo         = "GEO"
	ParamTz          = "TZ"
	ParamIndex       = "INDEX"
	ParamLevel       = "LEVEL"
	ParamGroup       = "GROUP"
	ParamCC          = "CC"
	ParamAuthor      = "AUTHOR"
	ParamAuthorName  = "AUTHOR-NAME"
	ParamCreated     = "CREATED"
	ParamDerived     = "DERIVED"
	ParamLabel       = "LABEL"
	ParamPhonetic    = "PHONETIC"
	ParamPropID      = "PROP-ID"
	ParamScript      = "SCRIPT"
	ParamServiceType = "SERVICE-TYPE"
	ParamUsername    = "USERNAME"
	ParamJSPtr       = "JSPTR"
	ParamJSComps     = "JSCOMPS"
)
