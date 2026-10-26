package services

import (
	"encoding/json"
	"errors"
	"maps"
	"meerkat/carddav"
	"meerkat/models"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/emersion/go-vcard"
	"gorm.io/gorm"
)

const (
	DuplicateReasonName  = "name"
	DuplicateReasonEmail = "email"
	DuplicateReasonPhone = "phone"

	// A key shared by more contacts than this (a company switchboard number, a
	// family's shared address) says nothing about any two of them being the
	// same person, and would flood the list with n² pairs.
	maxDuplicateBucketSize = 10

	// Phone numbers are compared on their trailing digits so that "+49 171 …"
	// and "0171 …" match. Numbers shorter than the minimum are too ambiguous.
	phoneMatchDigits    = 9
	minPhoneMatchDigits = 6

	MergeResolutionSource = "source"
	mergeFieldPhoto       = "photo"
	mergeFieldDeceased    = "is_deceased"
	mergeFieldArchived    = "archived"
	mergeCustomPrefix     = "custom_fields."
)

// ErrMergeSameContact is returned when a contact is merged into itself.
var ErrMergeSameContact = errors.New("cannot merge a contact into itself")

// mergeScalarFields are the single-valued text fields a merge reconciles,
// keyed by their JSON name (which is also the conflict/resolution key).
var mergeScalarFields = []struct {
	key string
	get func(*models.Contact) *string
}{
	{"firstname", func(c *models.Contact) *string { return &c.Firstname }},
	{"lastname", func(c *models.Contact) *string { return &c.Lastname }},
	{"nickname", func(c *models.Contact) *string { return &c.Nickname }},
	{"prefix", func(c *models.Contact) *string { return &c.Prefix }},
	{"middle_name", func(c *models.Contact) *string { return &c.MiddleName }},
	{"suffix", func(c *models.Contact) *string { return &c.Suffix }},
	{"gender", func(c *models.Contact) *string { return &c.Gender }},
	{"birthday", func(c *models.Contact) *string { return &c.Birthday }},
	{"anniversary", func(c *models.Contact) *string { return &c.Anniversary }},
	{"deceased_date", func(c *models.Contact) *string { return &c.DeceasedDate }},
	{"organization", func(c *models.Contact) *string { return &c.Organization }},
	{"department", func(c *models.Contact) *string { return &c.Department }},
	{"job_title", func(c *models.Contact) *string { return &c.JobTitle }},
	{"role", func(c *models.Contact) *string { return &c.Role }},
	{"work_information", func(c *models.Contact) *string { return &c.WorkInformation }},
	{"how_we_met", func(c *models.Contact) *string { return &c.HowWeMet }},
	{"food_preference", func(c *models.Contact) *string { return &c.FoodPreference }},
	{"contact_information", func(c *models.Contact) *string { return &c.ContactInformation }},
}

// FindDuplicateContacts returns pairs of the user's contacts that share a full
// name, an email address or a phone number, minus pairs the user dismissed.
// Pairs with more shared signals come first.
func FindDuplicateContacts(db *gorm.DB, userID uint) ([]models.DuplicatePair, error) {
	var contacts []models.Contact
	// Thumbnails are base64 data URLs; they are loaded below for paired contacts only.
	if err := db.Select("id", "firstname", "lastname", "nickname", "email", "phone", "emails", "phones", "organization", "archived").
		Where("user_id = ?", userID).Order("id").Find(&contacts).Error; err != nil {
		return nil, err
	}

	var dismissals []models.ContactDuplicateDismissal
	if err := db.Where("user_id = ?", userID).Find(&dismissals).Error; err != nil {
		return nil, err
	}
	dismissed := make(map[[2]uint]bool, len(dismissals))
	for _, d := range dismissals {
		dismissed[orderedPair(d.ContactAID, d.ContactBID)] = true
	}

	buckets := map[string]map[string][]int{
		DuplicateReasonName:  {},
		DuplicateReasonEmail: {},
		DuplicateReasonPhone: {},
	}
	add := func(reason, key string, idx int) {
		if key == "" {
			return
		}
		// A contact listing the same email twice must not pair with itself.
		if members := buckets[reason][key]; len(members) > 0 && members[len(members)-1] == idx {
			return
		}
		buckets[reason][key] = append(buckets[reason][key], idx)
	}
	for i := range contacts {
		c := &contacts[i]
		add(DuplicateReasonName, duplicateNameKey(c.Firstname, c.Lastname), i)
		for _, e := range contactEmails(c) {
			add(DuplicateReasonEmail, emailKey(e.Value), i)
		}
		for _, p := range contactPhones(c) {
			add(DuplicateReasonPhone, phoneKey(p.Value), i)
		}
	}

	reasonsByPair := map[[2]int][]string{}
	for _, reason := range []string{DuplicateReasonName, DuplicateReasonEmail, DuplicateReasonPhone} {
		for _, members := range buckets[reason] {
			if len(members) < 2 || len(members) > maxDuplicateBucketSize {
				continue
			}
			for a := 0; a < len(members); a++ {
				for b := a + 1; b < len(members); b++ {
					pair := [2]int{members[a], members[b]}
					if !slices.Contains(reasonsByPair[pair], reason) {
						reasonsByPair[pair] = append(reasonsByPair[pair], reason)
					}
				}
			}
		}
	}

	pairs := make([]models.DuplicatePair, 0, len(reasonsByPair))
	for idx, reasons := range reasonsByPair {
		a, b := &contacts[idx[0]], &contacts[idx[1]]
		if dismissed[orderedPair(a.ID, b.ID)] {
			continue
		}
		pairs = append(pairs, models.DuplicatePair{
			Contacts: [2]models.DuplicateContactSummary{duplicateSummary(a), duplicateSummary(b)},
			Reasons:  reasons,
		})
	}
	if err := fillDuplicateThumbnails(db, userID, pairs); err != nil {
		return nil, err
	}
	sort.Slice(pairs, func(i, j int) bool {
		if len(pairs[i].Reasons) != len(pairs[j].Reasons) {
			return len(pairs[i].Reasons) > len(pairs[j].Reasons)
		}
		if pairs[i].Contacts[0].ID != pairs[j].Contacts[0].ID {
			return pairs[i].Contacts[0].ID < pairs[j].Contacts[0].ID
		}
		return pairs[i].Contacts[1].ID < pairs[j].Contacts[1].ID
	})
	return pairs, nil
}

func fillDuplicateThumbnails(db *gorm.DB, userID uint, pairs []models.DuplicatePair) error {
	ids := map[uint]bool{}
	for _, p := range pairs {
		ids[p.Contacts[0].ID], ids[p.Contacts[1].ID] = true, true
	}
	if len(ids) == 0 {
		return nil
	}
	var rows []models.Contact
	if err := db.Select("id", "photo_thumbnail").Where("user_id = ? AND id IN ? AND photo_thumbnail <> ''", userID, slices.Collect(maps.Keys(ids))).
		Find(&rows).Error; err != nil {
		return err
	}
	thumbnails := make(map[uint]string, len(rows))
	for _, r := range rows {
		thumbnails[r.ID] = r.PhotoThumbnail
	}
	for i := range pairs {
		for j := range pairs[i].Contacts {
			pairs[i].Contacts[j].PhotoThumbnail = thumbnails[pairs[i].Contacts[j].ID]
		}
	}
	return nil
}

// DismissDuplicatePair records that two contacts are not the same person, and
// drops dismissals of contacts that no longer exist. Dismissing an
// already-dismissed pair is a no-op.
func DismissDuplicatePair(db *gorm.DB, userID, contactAID, contactBID uint) error {
	var count int64
	if err := db.Model(&models.Contact{}).Where("user_id = ? AND id IN ?", userID, []uint{contactAID, contactBID}).Count(&count).Error; err != nil {
		return err
	}
	if count != 2 {
		return gorm.ErrRecordNotFound
	}
	pair := orderedPair(contactAID, contactBID)
	dismissal := models.ContactDuplicateDismissal{UserID: userID, ContactAID: pair[0], ContactBID: pair[1]}
	if err := db.Where(dismissal).FirstOrCreate(&dismissal).Error; err != nil {
		return err
	}

	// Contacts are soft-deleted on every path (DeleteContact, merges, CardDAV,
	// sync), so no foreign key can cascade to the dismissals. Prune the ones
	// whose contacts are gone on this write, keeping the duplicates GET read-only.
	live := db.Model(&models.Contact{}).Select("id").Where("user_id = ?", userID)
	return db.Where("user_id = ? AND (contact_a_id NOT IN (?) OR contact_b_id NOT IN (?))", userID, live, live).
		Delete(&models.ContactDuplicateDismissal{}).Error
}

// PreviewContactMerge loads both contacts and reports the fields that conflict
// and how much related data the merge would move from source to target.
func PreviewContactMerge(db *gorm.DB, userID, targetID, sourceID uint) (*models.MergePreview, error) {
	if targetID == sourceID {
		return nil, ErrMergeSameContact
	}
	target, source, err := loadMergePair(db, userID, targetID, sourceID)
	if err != nil {
		return nil, err
	}

	preview := &models.MergePreview{
		Target:    models.ContactResponse{Contact: *target, PhotoThumbnail: target.PhotoThumbnail},
		Source:    models.ContactResponse{Contact: *source, PhotoThumbnail: source.PhotoThumbnail},
		Conflicts: MergeConflicts(target, source),
	}
	counts := []struct {
		dst   *int64
		query *gorm.DB
	}{
		{&preview.Moved.Notes, db.Model(&models.Note{}).Where("contact_id = ? AND user_id = ?", sourceID, userID)},
		{&preview.Moved.Reminders, db.Model(&models.Reminder{}).Where("contact_id = ? AND user_id = ?", sourceID, userID)},
		{&preview.Moved.Relationships, db.Model(&models.Relationship{}).Where("contact_id = ? AND user_id = ?", sourceID, userID)},
		{&preview.Moved.Activities, db.Table("activity_contacts").Where("contact_id = ? AND activity_id IN (SELECT id FROM activities WHERE user_id = ? AND deleted_at IS NULL)", sourceID, userID)},
	}
	for _, c := range counts {
		if err := c.query.Count(c.dst).Error; err != nil {
			return nil, err
		}
	}
	return preview, nil
}

// MergeConflicts lists the fields both contacts fill with different values.
// List-valued fields (emails, phones, circles, …) never conflict: they are combined.
func MergeConflicts(target, source *models.Contact) []models.MergeConflict {
	conflicts := []models.MergeConflict{}
	for _, f := range mergeScalarFields {
		tv, sv := strings.TrimSpace(*f.get(target)), strings.TrimSpace(*f.get(source))
		if tv != "" && sv != "" && tv != sv {
			conflicts = append(conflicts, models.MergeConflict{Field: f.key, TargetValue: tv, SourceValue: sv})
		}
	}
	if hasPhoto(target) && hasPhoto(source) && (target.Photo != source.Photo || target.PhotoThumbnail != source.PhotoThumbnail) {
		conflicts = append(conflicts, models.MergeConflict{Field: mergeFieldPhoto, TargetValue: target.Photo, SourceValue: source.Photo})
	}
	if target.IsDeceased != source.IsDeceased {
		conflicts = append(conflicts, boolConflict(mergeFieldDeceased, target.IsDeceased, source.IsDeceased))
	}
	if target.Archived != source.Archived {
		conflicts = append(conflicts, boolConflict(mergeFieldArchived, target.Archived, source.Archived))
	}
	for _, key := range slices.Sorted(maps.Keys(source.CustomFields)) {
		tv, sv := strings.TrimSpace(target.CustomFields[key]), strings.TrimSpace(source.CustomFields[key])
		if tv != "" && sv != "" && tv != sv {
			conflicts = append(conflicts, models.MergeConflict{Field: mergeCustomPrefix + key, TargetValue: tv, SourceValue: sv})
		}
	}
	return conflicts
}

func boolConflict(field string, target, source bool) models.MergeConflict {
	return models.MergeConflict{Field: field, TargetValue: strconv.FormatBool(target), SourceValue: strconv.FormatBool(source)}
}

// MergeResult is the outcome of MergeContacts. DiscardedPhoto carries the
// photo the merged contact no longer uses, for the caller to delete from disk
// once the transaction has committed.
type MergeResult struct {
	Contact        models.Contact
	SourceID       uint
	DiscardedPhoto models.Contact
}

// MergeContacts folds the source contact into the target and deletes the
// source, in one transaction. Fields only one side fills are kept; conflicting
// fields take the side named in resolutions (default: target); list fields are
// combined without duplicates. Notes, reminders, activities and relationships
// move to the target.
func MergeContacts(db *gorm.DB, userID, targetID, sourceID uint, resolutions map[string]string) (*MergeResult, error) {
	if targetID == sourceID {
		return nil, ErrMergeSameContact
	}
	result := &MergeResult{SourceID: sourceID}
	err := db.Transaction(func(tx *gorm.DB) error {
		target, source, err := loadMergePair(tx, userID, targetID, sourceID)
		if err != nil {
			return err
		}
		result.DiscardedPhoto = applyMergedFields(target, source, resolutions)
		if err := tx.Save(target).Error; err != nil {
			return err
		}
		if err := moveContactData(tx, userID, targetID, sourceID); err != nil {
			return err
		}
		// An archived contact has no reminders (see ArchiveContact), including
		// the ones just moved over from the source.
		if target.Archived {
			if err := tx.Where("contact_id = ? AND user_id = ?", targetID, userID).Delete(&models.Reminder{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Delete(source).Error; err != nil {
			return err
		}
		result.Contact = *target
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func loadMergePair(db *gorm.DB, userID, targetID, sourceID uint) (*models.Contact, *models.Contact, error) {
	var target, source models.Contact
	if err := db.Where("user_id = ?", userID).First(&target, targetID).Error; err != nil {
		return nil, nil, err
	}
	if err := db.Where("user_id = ?", userID).First(&source, sourceID).Error; err != nil {
		return nil, nil, err
	}
	return &target, &source, nil
}

// applyMergedFields writes the merged field values into target and returns a
// contact holding only the photo that lost out (zero value if none did).
func applyMergedFields(target, source *models.Contact, resolutions map[string]string) models.Contact {
	pickSource := func(key string) bool { return resolutions[key] == MergeResolutionSource }

	for _, f := range mergeScalarFields {
		tv, sv := f.get(target), f.get(source)
		if strings.TrimSpace(*sv) == "" {
			continue
		}
		if strings.TrimSpace(*tv) == "" || pickSource(f.key) {
			*tv = *sv
		}
	}

	if len(source.CustomFields) > 0 {
		if target.CustomFields == nil {
			target.CustomFields = map[string]string{}
		}
		for key, sv := range source.CustomFields {
			if strings.TrimSpace(sv) == "" {
				continue
			}
			if strings.TrimSpace(target.CustomFields[key]) == "" || pickSource(mergeCustomPrefix+key) {
				target.CustomFields[key] = sv
			}
		}
	}

	target.Emails = unionBy(contactEmails(target), contactEmails(source), func(e models.ContactEmail) string { return emailKey(e.Value) })
	target.Phones = unionBy(contactPhones(target), contactPhones(source), func(p models.ContactPhone) string { return normalizePhoneForComparison(p.Value) })
	target.Addresses = unionBy(contactAddresses(target), contactAddresses(source), func(a models.ContactAddress) string {
		return strings.ToLower(models.FormatAddress(a))
	})
	target.URLs = unionBy(target.URLs, source.URLs, func(u models.ContactURL) string { return strings.ToLower(strings.TrimSpace(u.Value)) })
	target.IMPPs = unionBy(target.IMPPs, source.IMPPs, func(i models.ContactIMPP) string {
		return strings.ToLower(strings.TrimSpace(i.Type) + ":" + strings.TrimSpace(i.Value))
	})
	target.Circles = unionBy(target.Circles, source.Circles, func(c string) string { return c })

	if pickSource(mergeFieldDeceased) {
		target.IsDeceased = source.IsDeceased
	}
	if !target.IsDeceased {
		target.DeceasedDate = ""
	}
	if pickSource(mergeFieldArchived) {
		target.Archived = source.Archived
	}

	target.VCardExtra = mergeVCardExtra(target.VCardExtra, source.VCardExtra)

	var discarded models.Contact
	switch {
	case !hasPhoto(source):
	case !hasPhoto(target) || pickSource(mergeFieldPhoto):
		discarded.Photo, discarded.PhotoThumbnail = target.Photo, target.PhotoThumbnail
		target.Photo, target.PhotoThumbnail = source.Photo, source.PhotoThumbnail
	default:
		discarded.Photo, discarded.PhotoThumbnail = source.Photo, source.PhotoThumbnail
	}
	// Never hand back a file the merged contact still points at.
	if discarded.Photo == target.Photo {
		discarded.Photo = ""
	}
	if discarded.PhotoThumbnail == target.PhotoThumbnail {
		discarded.PhotoThumbnail = ""
	}
	return discarded
}

// mergeVCardExtra combines the unmapped vCard properties of both contacts, so
// properties only the source carried are still written out on CardDAV. If
// either side is not valid JSON the target's value is kept as is.
func mergeVCardExtra(target, source string) string {
	if source == "" {
		return target
	}
	if target == "" {
		return source
	}
	var t, s carddav.VCardExtra
	if json.Unmarshal([]byte(target), &t) != nil || json.Unmarshal([]byte(source), &s) != nil {
		return target
	}
	if t.Properties == nil {
		t.Properties = map[string][]vcard.Field{}
	}
	for name, fields := range s.Properties {
		t.Properties[name] = unionBy(t.Properties[name], fields, func(f vcard.Field) string {
			b, _ := json.Marshal(f)
			return string(b)
		})
	}
	merged, err := json.Marshal(t)
	if err != nil {
		return target
	}
	return string(merged)
}

// moveContactData re-points everything attached to source at target.
func moveContactData(tx *gorm.DB, userID, targetID, sourceID uint) error {
	for _, model := range []any{&models.Note{}, &models.Reminder{}, &models.ReminderCompletion{}} {
		if err := tx.Model(model).Where("contact_id = ? AND user_id = ?", sourceID, userID).
			Update("contact_id", targetID).Error; err != nil {
			return err
		}
	}

	// activity_contacts has (activity_id, contact_id) as primary key, so an
	// activity both contacts attended keeps a single row.
	if err := tx.Exec(`INSERT OR IGNORE INTO activity_contacts (activity_id, contact_id)
		SELECT activity_id, ? FROM activity_contacts
		WHERE contact_id = ? AND activity_id IN (SELECT id FROM activities WHERE user_id = ?)`,
		targetID, sourceID, userID).Error; err != nil {
		return err
	}
	if err := tx.Exec("DELETE FROM activity_contacts WHERE contact_id = ? AND activity_id IN (SELECT id FROM activities WHERE user_id = ?)",
		sourceID, userID).Error; err != nil {
		return err
	}

	return moveRelationships(tx, userID, targetID, sourceID)
}

// mergedRelationshipKey identifies a relationship once both of its ends are
// expressed in merged terms. RelatedID is 0 for a person without a contact of
// their own, who is then told apart by name.
type mergedRelationshipKey struct {
	contactID uint
	relType   string
	relatedID uint
	name      string
}

// moveRelationships re-points the source's relationships at the target. A
// relationship between the two contacts would become a self-relationship and
// is dropped, as is a source relationship the target already has (same type,
// same person); the copy that stays picks up any birthday or gender only the
// dropped one had. Relationships not involving the source are never touched.
func moveRelationships(tx *gorm.DB, userID, targetID, sourceID uint) error {
	var rels []models.Relationship
	if err := tx.Where("user_id = ? AND (contact_id IN ? OR related_contact_id IN ?)", userID,
		[]uint{targetID, sourceID}, []uint{targetID, sourceID}).Order("id").Find(&rels).Error; err != nil {
		return err
	}

	involvesSource := func(r models.Relationship) bool {
		return r.ContactID == sourceID || (r.RelatedContactID != nil && *r.RelatedContactID == sourceID)
	}
	remap := func(id uint) uint {
		if id == sourceID {
			return targetID
		}
		return id
	}
	keyOf := func(r models.Relationship) mergedRelationshipKey {
		k := mergedRelationshipKey{contactID: remap(r.ContactID), relType: strings.ToLower(strings.TrimSpace(r.Type))}
		if r.RelatedContactID != nil {
			k.relatedID = remap(*r.RelatedContactID)
		} else {
			k.name = strings.ToLower(strings.TrimSpace(r.Name))
		}
		return k
	}

	// Relationships untouched by the merge claim their identity first, so a
	// duplicate is always resolved by dropping the source's copy.
	sort.SliceStable(rels, func(i, j int) bool {
		return !involvesSource(rels[i]) && involvesSource(rels[j])
	})

	kept := map[mergedRelationshipKey]*models.Relationship{}
	updates := map[uint]map[string]any{}
	setUpdate := func(id uint, column string, value any) {
		if updates[id] == nil {
			updates[id] = map[string]any{}
		}
		updates[id][column] = value
	}
	var drop []uint
	for i := range rels {
		r := &rels[i]
		k := keyOf(*r)
		if !involvesSource(*r) {
			if _, exists := kept[k]; !exists {
				kept[k] = r
			}
			continue
		}
		if k.relatedID == k.contactID {
			drop = append(drop, r.ID)
			continue
		}
		if existing := kept[k]; existing != nil {
			if strings.TrimSpace(existing.Birthday) == "" && strings.TrimSpace(r.Birthday) != "" {
				existing.Birthday = r.Birthday
				setUpdate(existing.ID, "birthday", r.Birthday)
			}
			if existing.Gender == "" && r.Gender != "" {
				existing.Gender = r.Gender
				setUpdate(existing.ID, "gender", r.Gender)
			}
			drop = append(drop, r.ID)
			continue
		}
		kept[k] = r
		if k.contactID != r.ContactID {
			setUpdate(r.ID, "contact_id", k.contactID)
		}
		if r.RelatedContactID != nil && k.relatedID != *r.RelatedContactID {
			setUpdate(r.ID, "related_contact_id", k.relatedID)
		}
	}

	for _, id := range slices.Sorted(maps.Keys(updates)) {
		if err := tx.Model(&models.Relationship{}).Where("id = ?", id).Updates(updates[id]).Error; err != nil {
			return err
		}
	}
	if len(drop) > 0 {
		if err := tx.Where("id IN ?", drop).Delete(&models.Relationship{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func duplicateSummary(c *models.Contact) models.DuplicateContactSummary {
	return models.DuplicateContactSummary{
		ID:             c.ID,
		Firstname:      c.Firstname,
		Lastname:       c.Lastname,
		Nickname:       c.Nickname,
		Email:          c.Email,
		Phone:          c.Phone,
		Organization:   c.Organization,
		PhotoThumbnail: c.PhotoThumbnail,
		Archived:       c.Archived,
	}
}

// contactEmails, contactPhones and contactAddresses fall back to the legacy
// scalar for contacts created through the API without the list fields.
func contactEmails(c *models.Contact) []models.ContactEmail {
	if len(c.Emails) == 0 && strings.TrimSpace(c.Email) != "" {
		return []models.ContactEmail{{Value: c.Email}}
	}
	return c.Emails
}

func contactPhones(c *models.Contact) []models.ContactPhone {
	if len(c.Phones) == 0 && strings.TrimSpace(c.Phone) != "" {
		return []models.ContactPhone{{Value: c.Phone}}
	}
	return c.Phones
}

func contactAddresses(c *models.Contact) []models.ContactAddress {
	if len(c.Addresses) == 0 && strings.TrimSpace(c.Address) != "" {
		return []models.ContactAddress{{Street: c.Address}}
	}
	return c.Addresses
}

func hasPhoto(c *models.Contact) bool {
	return c.Photo != "" || c.PhotoThumbnail != ""
}

// unionBy appends the items of b whose key is not already in a, keeping order.
func unionBy[T any](a, b []T, key func(T) string) []T {
	seen := make(map[string]bool, len(a)+len(b))
	out := make([]T, 0, len(a)+len(b))
	for _, item := range slices.Concat(a, b) {
		k := key(item)
		if k != "" && seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, item)
	}
	return out
}

// duplicateNameKey is the whitespace- and case-folded full name; contacts
// without a first name have no key.
func duplicateNameKey(firstname, lastname string) string {
	full := strings.Join(strings.Fields(strings.ToLower(firstname+" "+lastname)), " ")
	if strings.TrimSpace(firstname) == "" {
		return ""
	}
	return full
}

func emailKey(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func phoneKey(phone string) string {
	digits := normalizePhoneForComparison(phone)
	if len(digits) < minPhoneMatchDigits {
		return ""
	}
	if len(digits) > phoneMatchDigits {
		digits = digits[len(digits)-phoneMatchDigits:]
	}
	return digits
}

func orderedPair(a, b uint) [2]uint {
	if a > b {
		a, b = b, a
	}
	return [2]uint{a, b}
}
