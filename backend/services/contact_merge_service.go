package services

import (
	"errors"
	"maps"
	"meerkat/models"
	"slices"
	"sort"
	"strconv"
	"strings"

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
	if err := db.Select("id", "firstname", "lastname", "nickname", "email", "phone", "emails", "phones", "organization", "photo_thumbnail", "archived").
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

// DismissDuplicatePair records that two contacts are not the same person.
// Dismissing an already-dismissed pair is a no-op.
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
	return db.Where(dismissal).FirstOrCreate(&dismissal).Error
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
	if hasPhoto(target) && hasPhoto(source) && target.Photo != source.Photo {
		conflicts = append(conflicts, models.MergeConflict{Field: mergeFieldPhoto, TargetValue: target.Photo, SourceValue: source.Photo})
	}
	for _, key := range slices.Sorted(maps.Keys(source.CustomFields)) {
		tv, sv := strings.TrimSpace(target.CustomFields[key]), strings.TrimSpace(source.CustomFields[key])
		if tv != "" && sv != "" && tv != sv {
			conflicts = append(conflicts, models.MergeConflict{Field: mergeCustomPrefix + key, TargetValue: tv, SourceValue: sv})
		}
	}
	return conflicts
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
		if err := tx.Where("user_id = ? AND (contact_a_id = ? OR contact_b_id = ?)", userID, sourceID, sourceID).
			Delete(&models.ContactDuplicateDismissal{}).Error; err != nil {
			return err
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

	target.IsDeceased = target.IsDeceased || source.IsDeceased
	// Archiving deletes reminders, so keep the result active if either side is.
	target.Archived = target.Archived && source.Archived

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

// moveRelationships re-points the source's relationships at the target. A
// relationship between the two contacts would become a self-relationship and
// is dropped, as is one the target already has (same type, same person).
func moveRelationships(tx *gorm.DB, userID, targetID, sourceID uint) error {
	var rels []models.Relationship
	if err := tx.Where("user_id = ? AND (contact_id IN ? OR related_contact_id IN ?)", userID,
		[]uint{targetID, sourceID}, []uint{targetID, sourceID}).Find(&rels).Error; err != nil {
		return err
	}

	// Identity of a relationship once both ends are expressed in merged terms.
	key := func(r models.Relationship, contactID uint, relatedID *uint) string {
		other := "name:" + strings.ToLower(strings.TrimSpace(r.Name))
		if relatedID != nil {
			other = "id:" + uintString(*relatedID)
		}
		return uintString(contactID) + "|" + strings.ToLower(strings.TrimSpace(r.Type)) + "|" + other
	}
	remap := func(id uint) uint {
		if id == sourceID {
			return targetID
		}
		return id
	}

	seen := map[string]bool{}
	var drop []uint
	// Existing target relationships claim their identity first so that the
	// source's copies are the ones dropped.
	sort.SliceStable(rels, func(i, j int) bool {
		return rels[i].ContactID != sourceID && rels[j].ContactID == sourceID
	})
	for _, r := range rels {
		contactID := remap(r.ContactID)
		var relatedID *uint
		if r.RelatedContactID != nil {
			id := remap(*r.RelatedContactID)
			relatedID = &id
		}
		if relatedID != nil && *relatedID == contactID {
			drop = append(drop, r.ID)
			continue
		}
		k := key(r, contactID, relatedID)
		if seen[k] {
			drop = append(drop, r.ID)
			continue
		}
		seen[k] = true

		updates := map[string]any{}
		if contactID != r.ContactID {
			updates["contact_id"] = contactID
		}
		if relatedID != nil && *relatedID != *r.RelatedContactID {
			updates["related_contact_id"] = *relatedID
		}
		if len(updates) > 0 {
			if err := tx.Model(&models.Relationship{}).Where("id = ?", r.ID).Updates(updates).Error; err != nil {
				return err
			}
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

func uintString(v uint) string {
	return strconv.FormatUint(uint64(v), 10)
}
