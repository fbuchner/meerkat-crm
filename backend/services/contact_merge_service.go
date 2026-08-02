package services

import (
	"fmt"
	"sort"
	"strings"

	"mycorrhizal/models"

	"gorm.io/gorm"
)

// mergeScalarFields is the field table backing both ComputeContactMergeResolution
// and ApplyContactMergeResolution: every scalar covered by ticket N1's
// "Resolution by field class" table (docs/fork-plan/tickets/01-N1-contact-merge.md).
// Deliberately excludes Email/Phone/Address: those are legacy scalars
// Contact.BeforeSave (models/contact.go) derives from Emails[0]/Phones[0]/
// Addresses[0] on every save -- resolving them here and then letting
// BeforeSave immediately overwrite them again would be silently
// self-defeating.
type scalarFieldSpec struct {
	Key   string
	Label string
	Get   func(c *models.Contact) string
	Set   func(c *models.Contact, v string)
}

var mergeScalarFields = []scalarFieldSpec{
	{"firstname", "First Name", func(c *models.Contact) string { return c.Firstname }, func(c *models.Contact, v string) { c.Firstname = v }},
	{"lastname", "Last Name", func(c *models.Contact) string { return c.Lastname }, func(c *models.Contact, v string) { c.Lastname = v }},
	{"middle_name", "Middle Name", func(c *models.Contact) string { return c.MiddleName }, func(c *models.Contact, v string) { c.MiddleName = v }},
	{"prefix", "Prefix", func(c *models.Contact) string { return c.Prefix }, func(c *models.Contact, v string) { c.Prefix = v }},
	{"suffix", "Suffix", func(c *models.Contact) string { return c.Suffix }, func(c *models.Contact, v string) { c.Suffix = v }},
	{"nickname", "Nickname", func(c *models.Contact) string { return c.Nickname }, func(c *models.Contact, v string) { c.Nickname = v }},
	{"gender", "Gender", func(c *models.Contact) string { return c.Gender }, func(c *models.Contact, v string) { c.Gender = v }},
	{"birthday", "Birthday", func(c *models.Contact) string { return c.Birthday }, func(c *models.Contact, v string) { c.Birthday = v }},
	{"anniversary", "Anniversary", func(c *models.Contact) string { return c.Anniversary }, func(c *models.Contact, v string) { c.Anniversary = v }},
	{"organization", "Organization", func(c *models.Contact) string { return c.Organization }, func(c *models.Contact, v string) { c.Organization = v }},
	{"department", "Department", func(c *models.Contact) string { return c.Department }, func(c *models.Contact, v string) { c.Department = v }},
	{"job_title", "Job Title", func(c *models.Contact) string { return c.JobTitle }, func(c *models.Contact, v string) { c.JobTitle = v }},
	{"role", "Role", func(c *models.Contact) string { return c.Role }, func(c *models.Contact, v string) { c.Role = v }},
	{"how_we_met", "How We Met", func(c *models.Contact) string { return c.HowWeMet }, func(c *models.Contact, v string) { c.HowWeMet = v }},
	{"work_information", "Work Information", func(c *models.Contact) string { return c.WorkInformation }, func(c *models.Contact, v string) { c.WorkInformation = v }},
	{"contact_information", "Contact Information", func(c *models.Contact) string { return c.ContactInformation }, func(c *models.Contact, v string) { c.ContactInformation = v }},
}

// ComputeContactMergeResolution is pure (no DB I/O). Shared by the preview
// and commit controllers so they can never compute a different answer for
// the same pair.
func ComputeContactMergeResolution(keeper, loser *models.Contact) *models.ContactMergeResolution {
	res := &models.ContactMergeResolution{ResolvedScalars: map[string]string{}}
	for _, f := range mergeScalarFields {
		kv, lv := f.Get(keeper), f.Get(loser)
		switch {
		case kv == lv:
			res.ResolvedScalars[f.Key] = kv
		case kv == "":
			res.ResolvedScalars[f.Key] = lv
		case lv == "":
			res.ResolvedScalars[f.Key] = kv
		default:
			res.Conflicts = append(res.Conflicts, models.ContactMergeFieldConflict{
				Field: f.Key, Label: f.Label, KeeperValue: kv, LoserValue: lv,
			})
		}
	}
	sort.Slice(res.Conflicts, func(i, j int) bool { return res.Conflicts[i].Field < res.Conflicts[j].Field })

	res.Emails = unionEmails(keeper.Emails, loser.Emails)
	res.Phones = unionPhones(keeper.Phones, loser.Phones)
	res.Addresses = unionAddresses(keeper.Addresses, loser.Addresses)
	res.URLs = unionURLs(keeper.URLs, loser.URLs)
	res.IMPPs = unionIMPPs(keeper.IMPPs, loser.IMPPs)
	res.Circles = unionCircles(keeper.Circles, loser.Circles)
	res.CustomFields = unionCustomFields(keeper.CustomFields, loser.CustomFields)
	return res
}

// ApplyContactMergeResolution mutates keeper in place: every auto-resolved
// scalar, every unioned multi-valued/Circles/CustomFields field from res,
// plus (for the conflict set) the caller-supplied resolutions. Does not call
// db.Save -- the caller does that, inside its own transaction. Returns an
// error naming the first unresolved conflict field; the controller is
// expected to have already validated this (see contact_merge_controller.go),
// so this is a defense-in-depth check, not the primary rejection path.
func ApplyContactMergeResolution(keeper *models.Contact, res *models.ContactMergeResolution, resolutions map[string]string) error {
	for _, f := range mergeScalarFields {
		if v, ok := res.ResolvedScalars[f.Key]; ok {
			f.Set(keeper, v)
		}
	}
	for _, conflict := range res.Conflicts {
		v, ok := resolutions[conflict.Field]
		if !ok {
			return fmt.Errorf("contact merge: unresolved conflict for field %q", conflict.Field)
		}
		for _, f := range mergeScalarFields {
			if f.Key == conflict.Field {
				f.Set(keeper, v)
			}
		}
	}
	keeper.Emails = res.Emails
	keeper.Phones = res.Phones
	keeper.Addresses = res.Addresses
	keeper.URLs = res.URLs
	keeper.IMPPs = res.IMPPs
	keeper.Circles = res.Circles
	keeper.CustomFields = res.CustomFields
	return nil
}

// unionEmails unions a and b, de-duplicating on (case-insensitive type,
// case-insensitive value). Keeper's entries (a) are added first so its
// casing/order wins ties.
func unionEmails(a, b []models.ContactEmail) []models.ContactEmail {
	seen := map[string]bool{}
	var out []models.ContactEmail
	add := func(e models.ContactEmail) {
		key := strings.ToLower(strings.TrimSpace(e.Type)) + "|" + strings.ToLower(strings.TrimSpace(e.Value))
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, e)
	}
	for _, e := range a {
		add(e)
	}
	for _, e := range b {
		add(e)
	}
	return out
}

// unionPhones unions on (case-insensitive type, normalized digits) -- reuses
// normalizePhoneForComparison (import_service.go, same package) so "phone
// equality" means the same thing here as it already does for
// DetectDuplicate's phone-match path.
func unionPhones(a, b []models.ContactPhone) []models.ContactPhone {
	seen := map[string]bool{}
	var out []models.ContactPhone
	add := func(p models.ContactPhone) {
		key := strings.ToLower(strings.TrimSpace(p.Type)) + "|" + normalizePhoneForComparison(p.Value)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, p)
	}
	for _, p := range a {
		add(p)
	}
	for _, p := range b {
		add(p)
	}
	return out
}

// unionURLs dedups on (case-insensitive type, case-insensitive value).
func unionURLs(a, b []models.ContactURL) []models.ContactURL {
	seen := map[string]bool{}
	var out []models.ContactURL
	add := func(u models.ContactURL) {
		key := strings.ToLower(strings.TrimSpace(u.Type)) + "|" + strings.ToLower(strings.TrimSpace(u.Value))
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, u)
	}
	for _, u := range a {
		add(u)
	}
	for _, u := range b {
		add(u)
	}
	return out
}

// unionIMPPs dedups on (case-insensitive type, case-insensitive value).
func unionIMPPs(a, b []models.ContactIMPP) []models.ContactIMPP {
	seen := map[string]bool{}
	var out []models.ContactIMPP
	add := func(i models.ContactIMPP) {
		key := strings.ToLower(strings.TrimSpace(i.Type)) + "|" + strings.ToLower(strings.TrimSpace(i.Value))
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, i)
	}
	for _, i := range a {
		add(i)
	}
	for _, i := range b {
		add(i)
	}
	return out
}

// unionAddresses dedups on every field lowercased+trimmed -- a postal
// address has no single "value" to normalize on; its identity is the whole
// tuple.
func unionAddresses(a, b []models.ContactAddress) []models.ContactAddress {
	seen := map[string]bool{}
	var out []models.ContactAddress
	norm := func(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
	add := func(addr models.ContactAddress) {
		key := strings.Join([]string{norm(addr.Type), norm(addr.Street), norm(addr.City), norm(addr.Region), norm(addr.Postal), norm(addr.Country)}, "|")
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, addr)
	}
	for _, addr := range a {
		add(addr)
	}
	for _, addr := range b {
		add(addr)
	}
	return out
}

// unionCircles unions with case-insensitive dedup, keeping the keeper's
// casing on a tie (a appears first).
func unionCircles(a, b []string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		key := strings.ToLower(strings.TrimSpace(s))
		if key == "" || seen[key] {
			return
		}
		seen[key] = true
		out = append(out, s)
	}
	for _, s := range a {
		add(s)
	}
	for _, s := range b {
		add(s)
	}
	return out
}

// unionCustomFields unions the map; keeper wins on key collision (loser's
// map is copied first, then keeper's overwrites it).
func unionCustomFields(keeper, loser map[string]string) map[string]string {
	if len(keeper) == 0 && len(loser) == 0 {
		return nil
	}
	out := make(map[string]string, len(keeper)+len(loser))
	for k, v := range loser {
		out[k] = v
	}
	for k, v := range keeper {
		out[k] = v
	}
	return out
}

// ComputeFieldValueConflicts finds every FieldDefinition where both keeper
// and loser have a FieldValue row -- these cannot be unioned (the schema
// allows only one value per field per contact, unlike every other
// association type), so each is a conflict the user must resolve, keyed by
// FieldDefinition.ID rather than a fixed field name.
func ComputeFieldValueConflicts(db *gorm.DB, userID uint, keeperVCardUID, loserVCardUID string) ([]models.ContactMergeFieldConflict, error) {
	var keeperValues, loserValues []models.FieldValue
	if err := db.Where("entity_id = ? AND user_id = ?", keeperVCardUID, userID).Find(&keeperValues).Error; err != nil {
		return nil, err
	}
	if err := db.Where("entity_id = ? AND user_id = ?", loserVCardUID, userID).Find(&loserValues).Error; err != nil {
		return nil, err
	}
	if len(keeperValues) == 0 || len(loserValues) == 0 {
		return nil, nil
	}

	keeperByDef := make(map[string]models.FieldValue, len(keeperValues))
	for _, v := range keeperValues {
		keeperByDef[v.FieldDefinitionID] = v
	}

	var conflicts []models.ContactMergeFieldConflict
	var defIDs []string
	loserByDef := make(map[string]models.FieldValue, len(loserValues))
	for _, v := range loserValues {
		if _, ok := keeperByDef[v.FieldDefinitionID]; ok {
			loserByDef[v.FieldDefinitionID] = v
			defIDs = append(defIDs, v.FieldDefinitionID)
		}
	}
	if len(defIDs) == 0 {
		return nil, nil
	}

	var defs []models.FieldDefinition
	if err := db.Where("id IN ? AND user_id = ?", defIDs, userID).Find(&defs).Error; err != nil {
		return nil, err
	}
	labelByID := make(map[string]string, len(defs))
	for _, d := range defs {
		labelByID[d.ID] = d.Label
	}

	for _, defID := range defIDs {
		conflicts = append(conflicts, models.ContactMergeFieldConflict{
			Field:       defID,
			Label:       labelByID[defID],
			KeeperValue: string(keeperByDef[defID].Value),
			LoserValue:  string(loserByDef[defID].Value),
		})
	}
	sort.Slice(conflicts, func(i, j int) bool { return conflicts[i].Field < conflicts[j].Field })
	return conflicts, nil
}

// ComputeContactMergeAssociationCounts runs the COUNT(*) queries the preview
// endpoint returns and the commit path snapshots (pre-repoint) for the audit
// note. db may be either the plain *gorm.DB (preview, read-only) or a
// *gorm.DB transaction handle (commit, so the counts reflect the same
// row-visibility as the repoint that follows).
func ComputeContactMergeAssociationCounts(db *gorm.DB, userID uint, loserID uint, loserVCardUID string) (models.ContactMergeAssociationCounts, error) {
	var c models.ContactMergeAssociationCounts
	steps := []struct {
		dst *int64
		q   *gorm.DB
	}{
		{&c.Notes, db.Model(&models.Note{}).Where("contact_id = ? AND user_id = ?", loserID, userID)},
		{&c.Reminders, db.Model(&models.Reminder{}).Where("contact_id = ? AND user_id = ?", loserID, userID)},
		{&c.ReminderCompletions, db.Model(&models.ReminderCompletion{}).Where("contact_id = ? AND user_id = ?", loserID, userID)},
		{&c.RelationshipEdges, db.Model(&models.RelationshipEdge{}).Where("user_id = ? AND (source_id = ? OR target_id = ?)", userID, loserVCardUID, loserVCardUID)},
		{&c.HouseholdMemberships, db.Model(&models.HouseholdMember{}).Where("member_vcard_uid = ? AND user_id = ?", loserVCardUID, userID)},
		{&c.CircleMemberships, db.Model(&models.CircleMember{}).Where("member_vcard_uid = ? AND user_id = ?", loserVCardUID, userID)},
		{&c.Tags, db.Model(&models.ContactTag{}).Where("contact_vcard_uid = ? AND user_id = ?", loserVCardUID, userID)},
		{&c.LifeEvents, db.Model(&models.LifeEvent{}).Where("entity_id = ? AND user_id = ?", loserVCardUID, userID)},
		{&c.LifeEventReferences, db.Model(&models.LifeEvent{}).Where(
			"user_id = ? AND related_entity_ids IS NOT NULL AND EXISTS "+
				"(SELECT 1 FROM json_each(life_events.related_entity_ids) WHERE json_each.value = ?)",
			userID, loserVCardUID)},
		{&c.FieldValues, db.Model(&models.FieldValue{}).Where("entity_id = ? AND user_id = ?", loserVCardUID, userID)},
		{&c.ContactSyncLinks, db.Model(&models.ContactSyncLink{}).Where("contact_id = ? AND user_id = ?", loserID, userID)},
	}
	for _, s := range steps {
		if err := s.q.Count(s.dst).Error; err != nil {
			return c, err
		}
	}
	if err := db.Raw(
		"SELECT COUNT(*) FROM activity_contacts ac JOIN activities a ON a.id = ac.activity_id "+
			"WHERE ac.contact_id = ? AND a.user_id = ?",
		loserID, userID,
	).Scan(&c.Activities).Error; err != nil {
		return c, err
	}
	return c, nil
}

// RepointContactAssociations moves every association off loser onto keeper,
// inside tx (must already be inside a db.Transaction). Does NOT touch
// ContactSyncLink and does NOT delete the loser row itself -- both stay the
// job of the shared deleteContactAssociations/tx.Delete(&loser) calls the
// controller makes right after this returns, so ContactSyncLink cleanup is
// never duplicated and can never drift from DeleteContact's own behavior.
// fvConflicts/resolutions apply the user's chosen value onto whichever
// FieldValue row survives a collision (see the FieldValue section below).
// Returns the number of RelationshipEdge rows dropped as a self-loop or
// semantic duplicate, for the audit note.
func RepointContactAssociations(
	tx *gorm.DB, userID uint, keeper, loser *models.Contact,
	fvConflicts []models.ContactMergeFieldConflict, resolutions map[string]string,
) (int, error) {
	// Notes / Reminders / ReminderCompletions: plain repoint, no unique
	// constraint on contact_id.
	if err := tx.Model(&models.Note{}).Where("contact_id = ? AND user_id = ?", loser.ID, userID).
		Update("contact_id", keeper.ID).Error; err != nil {
		return 0, err
	}
	if err := tx.Model(&models.Reminder{}).Where("contact_id = ? AND user_id = ?", loser.ID, userID).
		Update("contact_id", keeper.ID).Error; err != nil {
		return 0, err
	}
	if err := tx.Model(&models.ReminderCompletion{}).Where("contact_id = ? AND user_id = ?", loser.ID, userID).
		Update("contact_id", keeper.ID).Error; err != nil {
		return 0, err
	}

	// activity_contacts: composite PK (activity_id, contact_id), no Go
	// model. Dedup-delete the loser's row where the keeper is already on the
	// same activity (would otherwise violate the PK), then repoint the rest.
	// Raw SQL matches the one existing precedent for touching this table
	// (DeleteContact) -- no ON CONFLICT/INSERT OR IGNORE idiom exists
	// elsewhere in this codebase to prefer instead.
	if err := tx.Exec(
		"DELETE FROM activity_contacts WHERE contact_id = ? AND activity_id IN "+
			"(SELECT activity_id FROM activity_contacts WHERE contact_id = ?) "+
			"AND activity_id IN (SELECT id FROM activities WHERE user_id = ?)",
		loser.ID, keeper.ID, userID,
	).Error; err != nil {
		return 0, err
	}
	if err := tx.Exec(
		"UPDATE activity_contacts SET contact_id = ? WHERE contact_id = ? "+
			"AND activity_id IN (SELECT id FROM activities WHERE user_id = ?)",
		keeper.ID, loser.ID, userID,
	).Error; err != nil {
		return 0, err
	}

	// household_members / circle_members / contact_tags: each has a
	// uniqueIndex on (container_id, member_vcard_uid) -- same dedup-then-
	// repoint shape as activity_contacts, one pass per table.
	if err := tx.Exec(
		"DELETE FROM household_members WHERE member_vcard_uid = ? AND user_id = ? AND household_id IN "+
			"(SELECT household_id FROM household_members WHERE member_vcard_uid = ? AND user_id = ?)",
		loser.VCardUID, userID, keeper.VCardUID, userID,
	).Error; err != nil {
		return 0, err
	}
	if err := tx.Exec(
		"UPDATE household_members SET member_vcard_uid = ? WHERE member_vcard_uid = ? AND user_id = ?",
		keeper.VCardUID, loser.VCardUID, userID,
	).Error; err != nil {
		return 0, err
	}

	if err := tx.Exec(
		"DELETE FROM circle_members WHERE member_vcard_uid = ? AND user_id = ? AND circle_id IN "+
			"(SELECT circle_id FROM circle_members WHERE member_vcard_uid = ? AND user_id = ?)",
		loser.VCardUID, userID, keeper.VCardUID, userID,
	).Error; err != nil {
		return 0, err
	}
	if err := tx.Exec(
		"UPDATE circle_members SET member_vcard_uid = ? WHERE member_vcard_uid = ? AND user_id = ?",
		keeper.VCardUID, loser.VCardUID, userID,
	).Error; err != nil {
		return 0, err
	}

	if err := tx.Exec(
		"DELETE FROM contact_tags WHERE contact_vcard_uid = ? AND user_id = ? AND tag_id IN "+
			"(SELECT tag_id FROM contact_tags WHERE contact_vcard_uid = ? AND user_id = ?)",
		loser.VCardUID, userID, keeper.VCardUID, userID,
	).Error; err != nil {
		return 0, err
	}
	if err := tx.Exec(
		"UPDATE contact_tags SET contact_vcard_uid = ? WHERE contact_vcard_uid = ? AND user_id = ?",
		keeper.VCardUID, loser.VCardUID, userID,
	).Error; err != nil {
		return 0, err
	}

	// field_values: uniqueIndex on (field_definition_id, entity_id). Where
	// both sides had a value (fvConflicts), drop the loser's row and apply
	// the user's chosen value onto the keeper's surviving row -- a value
	// genuinely cannot be unioned, unlike every other association. Where
	// only one side had a value, plain repoint.
	if err := tx.Exec(
		"DELETE FROM field_values WHERE entity_id = ? AND user_id = ? AND field_definition_id IN "+
			"(SELECT field_definition_id FROM field_values WHERE entity_id = ? AND user_id = ?)",
		loser.VCardUID, userID, keeper.VCardUID, userID,
	).Error; err != nil {
		return 0, err
	}
	for _, conflict := range fvConflicts {
		chosen, ok := resolutions[conflict.Field]
		if !ok {
			return 0, fmt.Errorf("contact merge: unresolved field value conflict for %q", conflict.Field)
		}
		if err := tx.Exec(
			"UPDATE field_values SET value = ? WHERE entity_id = ? AND user_id = ? AND field_definition_id = ?",
			chosen, keeper.VCardUID, userID, conflict.Field,
		).Error; err != nil {
			return 0, err
		}
	}
	if err := tx.Exec(
		"UPDATE field_values SET entity_id = ? WHERE entity_id = ? AND user_id = ?",
		keeper.VCardUID, loser.VCardUID, userID,
	).Error; err != nil {
		return 0, err
	}

	// LifeEvent: two independent sub-parts. (1) the loser's own subject
	// (EntityID) -- plain bulk UPDATE, no unique constraint. Run first so
	// part (2)'s self-reference check below also catches rows whose
	// EntityID was just re-pointed in this same transaction.
	if err := tx.Model(&models.LifeEvent{}).Where("entity_id = ? AND user_id = ?", loser.VCardUID, userID).
		Update("entity_id", keeper.VCardUID).Error; err != nil {
		return 0, err
	}

	// (2) RelatedEntityIDs -- a JSON array serialized into a TEXT column
	// with no per-element SQL write path. json_each cheaply finds candidate
	// rows; the rewrite itself happens in Go.
	var candidates []models.LifeEvent
	if err := tx.Where(
		"user_id = ? AND related_entity_ids IS NOT NULL AND EXISTS "+
			"(SELECT 1 FROM json_each(life_events.related_entity_ids) WHERE json_each.value = ?)",
		userID, loser.VCardUID,
	).Find(&candidates).Error; err != nil {
		return 0, err
	}
	for i := range candidates {
		ev := &candidates[i]
		changed := false
		for j, id := range ev.RelatedEntityIDs {
			if id == loser.VCardUID {
				ev.RelatedEntityIDs[j] = keeper.VCardUID
				changed = true
			}
		}
		if !changed {
			continue
		}
		ev.RelatedEntityIDs = dedupStrings(ev.RelatedEntityIDs)
		if ev.EntityID == keeper.VCardUID {
			ev.RelatedEntityIDs = removeString(ev.RelatedEntityIDs, keeper.VCardUID)
		}
		if err := tx.Save(ev).Error; err != nil {
			return 0, err
		}
	}

	// RelationshipEdge: bulk repoint both endpoints, then drop self-loops
	// and semantic duplicates.
	if err := tx.Model(&models.RelationshipEdge{}).Where("source_id = ? AND user_id = ?", loser.VCardUID, userID).
		Update("source_id", keeper.VCardUID).Error; err != nil {
		return 0, err
	}
	if err := tx.Model(&models.RelationshipEdge{}).Where("target_id = ? AND user_id = ?", loser.VCardUID, userID).
		Update("target_id", keeper.VCardUID).Error; err != nil {
		return 0, err
	}

	selfLoopResult := tx.Where("user_id = ? AND source_id = ? AND target_id = ?", userID, keeper.VCardUID, keeper.VCardUID).
		Delete(&models.RelationshipEdge{})
	if selfLoopResult.Error != nil {
		return 0, selfLoopResult.Error
	}
	edgesDropped := int(selfLoopResult.RowsAffected)

	var edges []models.RelationshipEdge
	if err := tx.Where("user_id = ? AND (source_id = ? OR target_id = ?)", userID, keeper.VCardUID, keeper.VCardUID).
		Find(&edges).Error; err != nil {
		return 0, err
	}
	toDelete := map[string]bool{}
	for i := 0; i < len(edges); i++ {
		if toDelete[edges[i].ID] {
			continue
		}
		for j := i + 1; j < len(edges); j++ {
			if toDelete[edges[j].ID] || !edgesConflict(edges[i], edges[j]) {
				continue
			}
			toDelete[pickEdgeToDrop(edges[i], edges[j]).ID] = true
		}
	}
	if len(toDelete) > 0 {
		ids := make([]string, 0, len(toDelete))
		for id := range toDelete {
			ids = append(ids, id)
		}
		if err := tx.Where("id IN ?", ids).Delete(&models.RelationshipEdge{}).Error; err != nil {
			return 0, err
		}
		edgesDropped += len(toDelete)
	}

	return edgesDropped, nil
}

// edgesConflict reports whether a and b express the same relationship fact
// -- literally identical, or mutual inverses (A rel_of B and
// B InverseRelationType(rel)_of A).
func edgesConflict(a, b models.RelationshipEdge) bool {
	exact := a.SourceID == b.SourceID && a.TargetID == b.TargetID && a.Type == b.Type
	inv := models.InverseRelationType(a.Type)
	inverse := inv != "" && a.SourceID == b.TargetID && a.TargetID == b.SourceID && b.Type == inv
	return exact || inverse
}

// pickEdgeToDrop picks which of two conflicting edges to remove, keeping the
// more authoritative one: higher Confidence wins; ties broken by Status
// (confirmed beats suggested -- confirmed edges are always Confidence 1.0
// per RelationshipEdge.Source's doc comment, so this mostly agrees with the
// Confidence check and only matters on an exact tie); remaining ties broken
// by earlier CreatedAt (the longer-standing record survives); final
// tiebreak is the lexicographically smaller ID (edge IDs are random UUIDs,
// not meaningfully ordered, but this guarantees a deterministic result with
// no other signal left to break on).
func pickEdgeToDrop(a, b models.RelationshipEdge) models.RelationshipEdge {
	keep := a
	switch {
	case a.Confidence != b.Confidence:
		if b.Confidence > a.Confidence {
			keep = b
		}
	case a.Status != b.Status:
		if b.Status == models.RelationshipStatusConfirmed {
			keep = b
		}
	case !a.CreatedAt.Equal(b.CreatedAt):
		if b.CreatedAt.Before(a.CreatedAt) {
			keep = b
		}
	default:
		if b.ID < a.ID {
			keep = b
		}
	}
	if keep.ID == a.ID {
		return b
	}
	return a
}

// dedupStrings removes duplicate values from s, preserving first-occurrence order.
func dedupStrings(s []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(s))
	for _, v := range s {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

// removeString returns s with every occurrence of v removed.
func removeString(s []string, v string) []string {
	out := make([]string, 0, len(s))
	for _, x := range s {
		if x != v {
			out = append(out, x)
		}
	}
	return out
}

// BuildContactMergeNoteContent composes the audit note written to the
// keeper, recording what a merge changed. Not built via import_service.go's
// CreateMergeNote -- that function's newValues map[string]interface{}
// parameter only diffs a fixed scalar set (plus "circles" specially) and has
// no seam for arrays/associations/FieldValue conflicts; force-fitting it
// would fight its shape more than it would save. CreateMergeNote itself is
// untouched, and its own pinned tests (import_service_merge_test.go) stay
// green.
func BuildContactMergeNoteContent(
	loser *models.Contact, res *models.ContactMergeResolution, resolutions map[string]string,
	counts models.ContactMergeAssociationCounts, edgesDropped int,
) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Merged contact #%d (%s) into this record.\n", loser.ID, strings.TrimSpace(loser.Firstname+" "+loser.Lastname))

	if len(res.Conflicts) > 0 {
		b.WriteString("\nResolved conflicts:\n")
		for _, conflict := range res.Conflicts {
			chosen := resolutions[conflict.Field]
			switch chosen {
			case conflict.KeeperValue:
				fmt.Fprintf(&b, "- %s: kept %q (merged contact had %q)\n", conflict.Label, chosen, conflict.LoserValue)
			case conflict.LoserValue:
				fmt.Fprintf(&b, "- %s: took %q from the merged contact (was %q)\n", conflict.Label, chosen, conflict.KeeperValue)
			default:
				fmt.Fprintf(&b, "- %s: set to %q (was %q, merged contact had %q)\n", conflict.Label, chosen, conflict.KeeperValue, conflict.LoserValue)
			}
		}
	}
	if len(res.FieldValueConflicts) > 0 {
		b.WriteString("\nResolved custom field conflicts:\n")
		for _, conflict := range res.FieldValueConflicts {
			chosen := resolutions[conflict.Field]
			fmt.Fprintf(&b, "- %s: set to %q (was %q, merged contact had %q)\n", conflict.Label, chosen, conflict.KeeperValue, conflict.LoserValue)
		}
	}

	b.WriteString("\nRe-pointed: ")
	fmt.Fprintf(&b, "%d notes, %d activities, %d reminders, %d reminder completions, "+
		"%d relationship edges (%d dropped as duplicate/self-loop), %d household memberships, "+
		"%d circle memberships, %d tags, %d life events (%d references), %d custom field values.\n",
		counts.Notes, counts.Activities, counts.Reminders, counts.ReminderCompletions,
		counts.RelationshipEdges, edgesDropped, counts.HouseholdMemberships,
		counts.CircleMemberships, counts.Tags, counts.LifeEvents, counts.LifeEventReferences, counts.FieldValues)

	if counts.ContactSyncLinks > 0 {
		fmt.Fprintf(&b, "%d CardDAV sync link(s) on the merged contact were discarded (not re-pointed).\n", counts.ContactSyncLinks)
	}
	return b.String()
}
