package models

import (
	"mycorrhizal/contactmodel"
	"mycorrhizal/logger"
	"mycorrhizal/photostore"
)

// ApplyRecordToContact is the single, shared mapping from the neutral
// contactmodel.Record shape back onto a *Contact's legacy flat/array fields
// — the exact inverse of RecordFromContact (contact_record.go). Per
// docs/fork-plan/50-integration-and-rebrand.md WP-71 Gap 1, both
// CreateContact and UpdateContact (contact_controller.go) call this one
// function to turn the new nested REST input into a Contact, and the VCF/
// JSContact import path (services/import_service.go, Gap 4) calls it to turn
// an imported Record into a candidate Contact before feeding it through the
// existing (unmodified) DetectDuplicate/MergeImportedContact functions. There
// must be exactly one Record->Contact mapping, mirroring WP-70's read-side
// rule applied here to the write side.
//
// Field-by-field decisions cite the corresponding docs/fork-plan/
// 20-correspondence.md concept_id/row, same convention as RecordFromContact.
// ApplyRecordToContact never panics, including when c or r is nil.
//
// Three things every caller must know:
//
//  1. c.Card / c.CRM / c.Passthrough are set directly from r (the "authoritative
//     full-fidelity copy" — WP-71's own words): whatever richer data r carries
//     that has no flat-field home (SpeakToAs, PersonalInfo, SocialProfiles,
//     OtherOnlineServices, Keywords, extra Organizations/Titles, extra
//     name components, RelatedTo, Members, Localizations, ...) is preserved
//     there even though it isn't mirrored into a flat scalar/array below. The
//     flat-field population is for backward-compat readers only (WP-70's "old
//     fields stay fully functional" rule) — it is not meant to be a complete
//     re-encoding of r.
//  2. This function marks the Contact so BeforeSave (contact.go) will NOT
//     immediately re-derive and overwrite c.Card/c.CRM/c.Passthrough from the
//     (necessarily lossy) flat fields on the next save — see the
//     cardSetDirectly field doc on Contact in contact.go for why that guard
//     exists and what would break without it.
//  3. photoDir gates photo persistence (see applyMedia below): pass the
//     configured profile-photo directory (config.Config.ProfilePhotoDir) for
//     callers that want an incoming Card.Media{Kind:"photo"} entry written to
//     disk and mirrored onto Contact.Photo/PhotoThumbnail immediately (the
//     REST API's CreateContact/UpdateContact, and CardDAV's PutAddressObject).
//     Pass "" to skip photo persistence entirely — this is deliberate, not an
//     oversight, for callers with their own deferred/preview photo pipeline
//     (services/import_service.go's ParseVCF/ParseJSContact: the imported
//     Contact is only a preview at that point, not yet confirmed by the user,
//     and photo persistence there is already handled at confirm-time by
//     import_session.go's ConfirmVCF via extractPhotoFromRecord + this same
//     photostore package — persisting here too would write an orphaned photo
//     file for every previewed-but-never-confirmed contact).
func ApplyRecordToContact(c *Contact, r *contactmodel.Record, photoDir string) {
	if c == nil || r == nil {
		return
	}

	card := r.Card
	proj := contactmodel.DeriveProjection(r)

	applyName(c, card, proj)
	applyNickname(c, card)
	applyOrganization(c, card)
	applyTitles(c, card)
	applyEmails(c, card, proj)
	applyPhones(c, card, proj)
	applyImpp(c, card)
	applyLinks(c, card)
	applyAddresses(c, card)
	applyAnniversaries(c, card, proj)
	applyMedia(c, card, photoDir)

	// CRM envelope -> flat CRM fields (direct 1:1 copy, mirroring
	// RecordFromContact's forward direction exactly).
	env := r.Envelope
	c.Circles = env.Circles
	c.HowWeMet = env.HowWeMet
	c.WorkInformation = env.WorkInformation
	c.ContactInformation = env.ContactInformation
	c.CustomFields = env.CustomFields

	// "uid" row, inverse direction: Card.UID -> VCardUID. Only overwrite when
	// the incoming Card actually carries a UID (e.g. a client PUT that omits
	// it, or a brand-new POST, should not blank out/clobber an existing
	// VCardUID — BeforeCreate already generates one for genuinely new
	// contacts that have none).
	if card.UID != "" {
		c.VCardUID = card.UID
	}

	// Gender is deliberately left untouched: per RecordFromContact's own
	// extensive doc comment, Record has no Gender data at all (it never
	// mapped into Card/Envelope on the forward direction either), so there is
	// nothing here to pull it from. Callers that need to set/update Gender
	// (a legacy-only, non-standardized concept) do so directly on the
	// Contact alongside calling this function — see CreateContact/
	// UpdateContact in contact_controller.go.

	// VCardExtra is also deliberately left untouched here: it is being
	// superseded by Passthrough in spirit (see RecordFromContact's
	// buildPassthrough doc comment) but this WP does not attempt to
	// re-serialize Passthrough.VCard back into VCardExtra's bespoke JSON
	// shape — that data is already fully preserved, losslessly, in
	// c.Passthrough itself (assigned below), which is what every new/updated
	// reader should use going forward.

	// The neutral columns are the authoritative full-fidelity copy (see the
	// doc comment above) — assigned last, after the flat-field derivation
	// above, so nothing here can accidentally clobber it.
	c.Card = r.Card
	c.CRM = r.Envelope
	c.Passthrough = r.Passthrough

	// Tell BeforeSave not to re-derive (and thus truncate) this Card/CRM/
	// Passthrough from the flat fields it just populated above — see the
	// field doc on Contact.cardSetDirectly in contact.go.
	c.cardSetDirectly = true
}

// applyName is the inverse of buildName: Card.Name.Components ->
// Firstname/Lastname/MiddleName/Prefix/Suffix ("name.given"/"name.surname"/
// "name.given2"/"name.title"/"name.credential" rows). Falls back to
// DeriveProjection's Firstname/Lastname (which itself falls back to
// Card.Name.Full when there are no components at all) so a Record built
// with only Name.Full set (e.g. a minimal JSContact import) still lands a
// usable Firstname/Lastname rather than leaving them blank.
func applyName(c *Contact, card contactmodel.Card, proj contactmodel.Projection) {
	c.Firstname, c.Lastname, c.MiddleName, c.Prefix, c.Suffix = "", "", "", "", ""
	if card.Name != nil {
		for _, comp := range card.Name.Components {
			switch comp.Kind {
			case "given":
				c.Firstname = comp.Value
			case "surname":
				c.Lastname = comp.Value
			case "given2":
				c.MiddleName = comp.Value
			case "title":
				c.Prefix = comp.Value
			case "credential":
				c.Suffix = comp.Value
			}
		}
	}
	if c.Firstname == "" {
		c.Firstname = proj.Firstname
	}
	if c.Lastname == "" {
		c.Lastname = proj.Lastname
	}
}

// applyNickname is the inverse of buildNicknames: Card.Nicknames[0].Name ->
// Nickname ("nickname" row). Only the first nickname survives on the flat
// side (Nickname is a single scalar); any additional entries remain fully
// present in c.Card.
func applyNickname(c *Contact, card contactmodel.Card) {
	c.Nickname = ""
	if len(card.Nicknames) > 0 {
		c.Nickname = card.Nicknames[0].Name
	}
}

// applyOrganization is the inverse of buildOrganizations: Card.Organizations[0]
// -> Organization/Department ("org"/"org.unit" rows). Only the first
// organization and its first unit survive on the flat side; additional
// organizations/units remain fully present in c.Card.
func applyOrganization(c *Contact, card contactmodel.Card) {
	c.Organization, c.Department = "", ""
	if len(card.Organizations) > 0 {
		c.Organization = card.Organizations[0].Name
		if len(card.Organizations[0].Units) > 0 {
			c.Department = card.Organizations[0].Units[0].Name
		}
	}
}

// applyTitles is the inverse of buildTitles: Card.Titles[kind=title/role] ->
// JobTitle/Role ("title"/"role" rows). Only the last title/role of each kind
// survives on the flat side if a Record has more than one of a kind (an edge
// case with no legacy analog); all entries remain fully present in c.Card.
func applyTitles(c *Contact, card contactmodel.Card) {
	c.JobTitle, c.Role = "", ""
	for _, t := range card.Titles {
		switch t.Kind {
		case "title":
			c.JobTitle = t.Name
		case "role":
			c.Role = t.Name
		}
	}
}

// applyEmails is the inverse of buildEmails: Card.Emails[] -> Emails[]
// ("email" row; Label -> Type, the same free-text-Type convention the
// forward direction uses). Also mirrors the primary entry into the legacy
// Email scalar immediately (BeforeSave would do this too, but setting it
// here keeps the Contact internally consistent even before a save, e.g. for
// callers/tests that inspect it pre-save).
func applyEmails(c *Contact, card contactmodel.Card, proj contactmodel.Projection) {
	c.Emails = nil
	for _, e := range card.Emails {
		c.Emails = append(c.Emails, ContactEmail{Type: e.Label, Value: e.Address})
	}
	if len(c.Emails) > 0 {
		c.Email = c.Emails[0].Value
	} else {
		c.Email = proj.PrimaryEmail
	}
}

// applyPhones mirrors applyEmails for the "phone" row (Card.Phones[] ->
// Phones[], Label -> Type), including the empty-case fallback: found by
// Tier 3c item 11a's audit (docs/fork-plan/95-backlog-and-priorities.md) as
// a real, live bug — this function cleared c.Phones but left the c.Phone
// scalar untouched when card.Phones was empty, so removing a contact's last
// phone number (via REST PUT, CardDAV sync, or VCF import) silently left a
// stale value on the scalar column while the array correctly went empty.
func applyPhones(c *Contact, card contactmodel.Card, proj contactmodel.Projection) {
	c.Phones = nil
	for _, p := range card.Phones {
		c.Phones = append(c.Phones, ContactPhone{Type: p.Label, Value: p.Number})
	}
	if len(c.Phones) > 0 {
		c.Phone = c.Phones[0].Value
	} else {
		c.Phone = proj.PrimaryPhone
	}
}

// applyImpp is the inverse of buildImpp: Card.ImppAddresses[] -> IMPPs[]
// ("impp" row; Service -> Type, URI -> Value). Only ImppAddresses collapses
// back into the legacy IMPPs array, exactly mirroring the forward direction
// (buildImpp only ever reads from c.IMPPs, never from SocialProfiles/
// OtherOnlineServices) — see the three-array design note in
// docs/fork-plan/20-correspondence.md §20.7. Card.SocialProfiles and
// Card.OtherOnlineServices have no legacy flat-field home at all: that data
// is not lost (it stays fully intact on c.Card, assigned by the caller after
// this function returns), it simply isn't mirrored into any pre-existing
// legacy column, the same way it was never read from one on the forward
// direction either.
func applyImpp(c *Contact, card contactmodel.Card) {
	c.IMPPs = nil
	for _, i := range card.ImppAddresses {
		c.IMPPs = append(c.IMPPs, ContactIMPP{Type: i.Service, Value: i.URI})
	}
}

// applyLinks is the inverse of buildLinks: Card.Links[] -> URLs[] ("link"
// row; Label -> Type, URI -> Value).
func applyLinks(c *Contact, card contactmodel.Card) {
	c.URLs = nil
	for _, l := range card.Links {
		c.URLs = append(c.URLs, ContactURL{Type: l.Label, Value: l.URI})
	}
}

// applyAddresses is the inverse of buildAddresses/addressFromContactAddress:
// Card.Addresses[] -> Addresses[] ("adr" row), reconstructing the flat
// ContactAddress fields from AddressComponent kinds. Also mirrors the first
// entry into the legacy Address scalar via FormatAddress, same as
// BeforeSave's existing sync logic.
//
// Known, accepted lossy case (documented, not a defect): an Address whose
// only content is Full (e.g. produced by RecordFromContact's own
// scalar-only fallback, or a JSContact/vCard LABEL-only address with no
// structured components) reconstructs to an all-blank ContactAddress here —
// there is nowhere structured to recover Street/City/... from a single
// opaque string. The data is not lost: it remains fully present, verbatim,
// in c.Card.Addresses[].Full after this function's caller assigns c.Card.
func applyAddresses(c *Contact, card contactmodel.Card) {
	c.Addresses = nil
	for _, a := range card.Addresses {
		c.Addresses = append(c.Addresses, contactAddressFromNeutral(a))
	}
	if len(c.Addresses) > 0 {
		c.Address = FormatAddress(c.Addresses[0])
	} else {
		c.Address = ""
	}
}

func contactAddressFromNeutral(a contactmodel.Address) ContactAddress {
	var out ContactAddress
	for _, comp := range a.Components {
		switch comp.Kind {
		case "name":
			out.Street = comp.Value
		case "locality":
			out.City = comp.Value
		case "region":
			out.Region = comp.Value
		case "postcode":
			out.Postal = comp.Value
		case "country":
			out.Country = comp.Value
		}
	}
	if len(a.Contexts) > 0 {
		out.Type = a.Contexts[0]
	}
	return out
}

// applyAnniversaries is the inverse of buildAnniversaries:
// Card.Anniversaries[kind=birth/wedding].Date -> Birthday/Anniversary
// ("anniversary.birth"/"anniversary.wedding" rows). Reuses
// contactmodel.DeriveProjection's own date-formatting logic (via proj for
// birth, and a tiny single-anniversary Record for wedding) rather than
// re-implementing the partial-date-to-string formatting by hand a second
// time in this package — the same "one implementation, not two" discipline
// RecordFromContact/parsePartialDateString already follows for the opposite
// direction.
func applyAnniversaries(c *Contact, card contactmodel.Card, proj contactmodel.Projection) {
	c.Birthday = proj.Birthday
	c.Anniversary = ""
	for _, a := range card.Anniversaries {
		if a.Kind == "wedding" {
			tmp := &contactmodel.Record{Card: contactmodel.Card{
				Anniversaries: []contactmodel.Anniversary{{Kind: "birth", Date: a.Date}},
			}}
			c.Anniversary = contactmodel.DeriveProjection(tmp).Birthday
			break
		}
	}
}

// applyMedia is the inverse of buildMedia (contact_record.go): the first
// Card.Media entry with Kind=="photo" is decoded and persisted to disk via
// photostore, mirroring the pre-existing carddav.PutAddressObject photo
// handling (embedded data or a remote URL, both supported) so a photo synced
// in via the REST API or CardDAV round-trips the same way a photo synced in
// via the legacy vcard_mapper.go path always has.
//
// A no-op, by design, when photoDir == "" — see the photoDir doc on
// ApplyRecordToContact above for which callers deliberately pass "" and why.
// Never returns an error: persistence failures are logged and otherwise
// swallowed, same as the legacy carddav.PutAddressObject handling this
// replaces (a bad/corrupt photo must not fail the whole contact save).
func applyMedia(c *Contact, card contactmodel.Card, photoDir string) {
	if photoDir == "" {
		return
	}
	var photo *contactmodel.Resource
	for i := range card.Media {
		if card.Media[i].Kind == "photo" {
			photo = &card.Media[i]
			break
		}
	}
	if photo == nil || photo.URI == "" {
		return
	}

	data, mediaType, photoURL := photostore.DecodePhotoURI(photo.URI, photo.MediaType)
	if len(data) == 0 && photoURL != "" {
		fetched, fetchedMediaType, err := photostore.FetchPhotoFromURL(photoURL)
		if err != nil {
			logger.Warn().Err(err).Str("photo_url", photoURL).Msg("ApplyRecordToContact: failed to fetch photo from URL")
			return
		}
		data, mediaType = fetched, fetchedMediaType
	}
	if len(data) == 0 {
		return
	}

	photoPath, thumbnailBase64, err := photostore.SaveContactPhoto(data, mediaType, photoDir)
	if err != nil {
		logger.Warn().Err(err).Str("media_type", mediaType).Int("photo_size", len(data)).Msg("ApplyRecordToContact: failed to save contact photo")
		return
	}
	c.Photo = photoPath
	c.PhotoThumbnail = thumbnailBase64
}
