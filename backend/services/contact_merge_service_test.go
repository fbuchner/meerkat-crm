package services

import (
	"encoding/json"
	"meerkat/carddav"
	"meerkat/models"
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupMergeDB(t *testing.T) (*gorm.DB, uint) {
	t.Helper()
	db, _ := setupRouter()
	require.NoError(t, db.AutoMigrate(&models.ReminderCompletion{}, &models.ContactDuplicateDismissal{}))
	user := models.User{Username: "merger", Password: "password123", Email: "merger@example.com"}
	require.NoError(t, db.Create(&user).Error)
	return db, user.ID
}

func createContact(t *testing.T, db *gorm.DB, c models.Contact) models.Contact {
	t.Helper()
	require.NoError(t, db.Create(&c).Error)
	return c
}

func pairIDs(p models.DuplicatePair) [2]uint {
	return [2]uint{p.Contacts[0].ID, p.Contacts[1].ID}
}

func TestFindDuplicateContacts(t *testing.T) {
	db, userID := setupMergeDB(t)

	anna1 := createContact(t, db, models.Contact{UserID: userID, Firstname: "Anna", Lastname: "Schmidt", PhotoThumbnail: "data:anna",
		Emails: []models.ContactEmail{{Value: "anna@example.com"}}, Phones: []models.ContactPhone{{Value: "+49 171 1234567"}}})
	anna2 := createContact(t, db, models.Contact{UserID: userID, Firstname: " anna ", Lastname: "SCHMIDT",
		Emails: []models.ContactEmail{{Value: "Anna@Example.com"}}, Phones: []models.ContactPhone{{Value: "0171 1234567"}}})
	byEmail := createContact(t, db, models.Contact{UserID: userID, Firstname: "Ben", Lastname: "Meyer",
		Emails: []models.ContactEmail{{Value: "ben@example.com"}}})
	byEmail2 := createContact(t, db, models.Contact{UserID: userID, Firstname: "Benjamin", Lastname: "Meyer", Email: "BEN@example.com"})
	createContact(t, db, models.Contact{UserID: userID, Firstname: "Unrelated", Lastname: "Person"})

	// Another user's identical contact must never be paired with ours.
	other := models.User{Username: "other", Password: "password123", Email: "other@example.com"}
	require.NoError(t, db.Create(&other).Error)
	createContact(t, db, models.Contact{UserID: other.ID, Firstname: "Anna", Lastname: "Schmidt"})

	pairs, err := FindDuplicateContacts(db, userID)
	require.NoError(t, err)
	require.Len(t, pairs, 2)

	assert.Equal(t, [2]uint{anna1.ID, anna2.ID}, pairIDs(pairs[0]))
	assert.ElementsMatch(t, []string{DuplicateReasonName, DuplicateReasonEmail, DuplicateReasonPhone}, pairs[0].Reasons)
	assert.Equal(t, "data:anna", pairs[0].Contacts[0].PhotoThumbnail)
	assert.Empty(t, pairs[0].Contacts[1].PhotoThumbnail)
	assert.Equal(t, [2]uint{byEmail.ID, byEmail2.ID}, pairIDs(pairs[1]))
	assert.Equal(t, []string{DuplicateReasonEmail}, pairs[1].Reasons)

	require.NoError(t, DismissDuplicatePair(db, userID, anna2.ID, anna1.ID))
	require.NoError(t, DismissDuplicatePair(db, userID, anna1.ID, anna2.ID), "dismissing twice is a no-op")
	pairs, err = FindDuplicateContacts(db, userID)
	require.NoError(t, err)
	require.Len(t, pairs, 1)
	assert.Equal(t, [2]uint{byEmail.ID, byEmail2.ID}, pairIDs(pairs[0]))

	assert.ErrorIs(t, DismissDuplicatePair(db, other.ID, anna1.ID, anna2.ID), gorm.ErrRecordNotFound)
}

func TestDismissDuplicatePairPrunesStaleDismissals(t *testing.T) {
	db, userID := setupMergeDB(t)
	a := createContact(t, db, models.Contact{UserID: userID, Firstname: "Sam"})
	b := createContact(t, db, models.Contact{UserID: userID, Firstname: "Sam"})
	c := createContact(t, db, models.Contact{UserID: userID, Firstname: "Sam"})
	d := createContact(t, db, models.Contact{UserID: userID, Firstname: "Kim"})
	e := createContact(t, db, models.Contact{UserID: userID, Firstname: "Kim"})
	require.NoError(t, DismissDuplicatePair(db, userID, a.ID, b.ID))
	require.NoError(t, DismissDuplicatePair(db, userID, a.ID, c.ID))

	// Removed outside DeleteContact, e.g. by CardDAV: a plain soft delete.
	require.NoError(t, db.Delete(&models.Contact{}, b.ID).Error)

	pairs, err := FindDuplicateContacts(db, userID)
	require.NoError(t, err)
	require.Len(t, pairs, 1, "only d/e is suggested; a/c stays dismissed")

	require.NoError(t, DismissDuplicatePair(db, userID, d.ID, e.ID))
	var remaining []models.ContactDuplicateDismissal
	require.NoError(t, db.Order("id").Find(&remaining).Error)
	require.Len(t, remaining, 2, "the dismissal of the deleted contact is pruned")
	assert.Equal(t, [2]uint{a.ID, c.ID}, [2]uint{remaining[0].ContactAID, remaining[0].ContactBID})
	assert.Equal(t, [2]uint{d.ID, e.ID}, [2]uint{remaining[1].ContactAID, remaining[1].ContactBID})
}

func TestFindDuplicateContactsSkipsOversizedBuckets(t *testing.T) {
	db, userID := setupMergeDB(t)
	for i := 0; i <= maxDuplicateBucketSize; i++ {
		createContact(t, db, models.Contact{UserID: userID, Firstname: "Colleague", Lastname: string(rune('A' + i)),
			Phones: []models.ContactPhone{{Value: "+49 30 9999999"}}})
	}
	pairs, err := FindDuplicateContacts(db, userID)
	require.NoError(t, err)
	assert.Empty(t, pairs)
}

func TestMergeConflicts(t *testing.T) {
	target := &models.Contact{Firstname: "Anna", Lastname: "Schmidt", Birthday: "1990-01-02", Nickname: "",
		Photo: "a.jpg", CustomFields: map[string]string{"Shoe size": "38", "Pet": "Cat"}}
	source := &models.Contact{Firstname: "Anna", Lastname: "Schmidt-Weber", Birthday: "1990-01-02", Nickname: "Annie",
		Photo: "b.jpg", CustomFields: map[string]string{"Shoe size": "39", "Team": "Blue"}}

	assert.Equal(t, []models.MergeConflict{
		{Field: "lastname", TargetValue: "Schmidt", SourceValue: "Schmidt-Weber"},
		{Field: "photo", TargetValue: "a.jpg", SourceValue: "b.jpg"},
		{Field: "custom_fields.Shoe size", TargetValue: "38", SourceValue: "39"},
	}, MergeConflicts(target, source))
}

func TestMergeContacts(t *testing.T) {
	db, userID := setupMergeDB(t)

	target := createContact(t, db, models.Contact{UserID: userID, Firstname: "Anna", Lastname: "Schmidt",
		Birthday: "1990-01-02", JobTitle: "Engineer", Photo: "target.jpg", PhotoThumbnail: "data:target",
		Circles:      []string{"Friends"},
		Emails:       []models.ContactEmail{{Type: "home", Value: "anna@example.com"}},
		CustomFields: map[string]string{"Shoe size": "38"}})
	source := createContact(t, db, models.Contact{UserID: userID, Firstname: "Anna", Lastname: "Schmidt-Weber",
		Nickname: "Annie", JobTitle: "Manager", Photo: "source.jpg", PhotoThumbnail: "data:source",
		Circles: []string{"Work", "Friends"},
		Emails: []models.ContactEmail{
			{Type: "work", Value: "ANNA@example.com"},
			{Type: "work", Value: "anna@work.example"},
		},
		Phones:       []models.ContactPhone{{Type: "cell", Value: "+49 171 1234567"}},
		CustomFields: map[string]string{"Shoe size": "39", "Team": "Blue"},
		IsDeceased:   false})
	friend := createContact(t, db, models.Contact{UserID: userID, Firstname: "Friend", Lastname: "One"})

	// Related data hanging off the source.
	note := models.Note{UserID: userID, Content: "met at conference", Date: time.Now(), ContactID: &source.ID}
	require.NoError(t, db.Create(&note).Error)
	reminder := models.Reminder{UserID: userID, Message: "call", RemindAt: time.Now(), Recurrence: "once", ContactID: &source.ID}
	require.NoError(t, db.Create(&reminder).Error)
	completion := models.ReminderCompletion{UserID: userID, ContactID: source.ID, Message: "called", CompletedAt: time.Now()}
	require.NoError(t, db.Create(&completion).Error)

	shared := models.Activity{UserID: userID, Title: "Dinner", Date: time.Now(), Contacts: []models.Contact{target, source}}
	require.NoError(t, db.Create(&shared).Error)
	onlySource := models.Activity{UserID: userID, Title: "Hike", Date: time.Now(), Contacts: []models.Contact{source}}
	require.NoError(t, db.Create(&onlySource).Error)

	rels := []models.Relationship{
		{UserID: userID, Name: "Anna", Type: "Spouse", ContactID: target.ID, RelatedContactID: &source.ID}, // between the two: dropped
		{UserID: userID, Name: "Max", Type: "Child", ContactID: target.ID},                                 // target's own
		{UserID: userID, Name: "max", Type: "child", ContactID: source.ID, Birthday: "2015-04-01"},         // same child: dropped, birthday kept
		{UserID: userID, Name: "Tom", Type: "Child", ContactID: target.ID},                                 // target's own...
		{UserID: userID, Name: "Tom", Type: "Child", ContactID: target.ID},                                 // ...pre-existing duplicate: left alone
		{UserID: userID, Name: "Lena", Type: "Child", ContactID: source.ID},                                // moved
		{UserID: userID, Name: "Anna", Type: "Friend", ContactID: friend.ID, RelatedContactID: &source.ID}, // incoming: re-pointed
	}
	require.NoError(t, db.Create(&rels).Error)

	preview, err := PreviewContactMerge(db, userID, target.ID, source.ID)
	require.NoError(t, err)
	assert.Equal(t, models.MergeMovedCounts{Notes: 1, Activities: 2, Reminders: 1, Relationships: 2, EmploymentHistories: 1}, preview.Moved)
	assert.Len(t, preview.Conflicts, 3) // lastname, photo, custom Shoe size

	result, err := MergeContacts(db, userID, target.ID, source.ID, map[string]string{
		"lastname":                "source",
		"photo":                   "source",
		"custom_fields.Shoe size": "target",
	})
	require.NoError(t, err)

	var merged models.Contact
	require.NoError(t, db.First(&merged, target.ID).Error)
	assert.Equal(t, "Schmidt-Weber", merged.Lastname)
	assert.Equal(t, "Annie", merged.Nickname, "a field only the source fills is kept")
	assert.Equal(t, "Engineer", merged.JobTitle)
	assert.Equal(t, "Engineer", result.Contact.JobTitle, "the response reflects the recomputed fields")
	assert.Equal(t, "1990-01-02", merged.Birthday)
	assert.Equal(t, "source.jpg", merged.Photo)
	assert.Equal(t, "data:source", merged.PhotoThumbnail)
	assert.Equal(t, models.Contact{Photo: "target.jpg", PhotoThumbnail: "data:target"}, result.DiscardedPhoto)
	assert.Equal(t, []string{"Friends", "Work"}, merged.Circles)
	assert.Equal(t, map[string]string{"Shoe size": "38", "Team": "Blue"}, merged.CustomFields)
	assert.Equal(t, []models.ContactEmail{
		{Type: "home", Value: "anna@example.com"},
		{Type: "work", Value: "anna@work.example"},
	}, merged.Emails)
	assert.Equal(t, "anna@example.com", merged.Email)
	assert.Equal(t, "+49 171 1234567", merged.Phone)

	// Source is gone.
	assert.ErrorIs(t, db.First(&models.Contact{}, source.ID).Error, gorm.ErrRecordNotFound)

	// Related data now belongs to the target.
	var count int64
	db.Model(&models.EmploymentHistory{}).Where("contact_id = ?", target.ID).Count(&count)
	assert.EqualValues(t, 2, count)
	db.Model(&models.Note{}).Where("contact_id = ?", target.ID).Count(&count)
	assert.EqualValues(t, 1, count)
	db.Model(&models.Reminder{}).Where("contact_id = ?", target.ID).Count(&count)
	assert.EqualValues(t, 1, count)
	db.Model(&models.ReminderCompletion{}).Where("contact_id = ?", target.ID).Count(&count)
	assert.EqualValues(t, 1, count)
	db.Table("activity_contacts").Where("contact_id = ?", target.ID).Count(&count)
	assert.EqualValues(t, 2, count)
	db.Table("activity_contacts").Where("contact_id = ?", source.ID).Count(&count)
	assert.EqualValues(t, 0, count)

	var remaining []models.Relationship
	require.NoError(t, db.Order("id").Find(&remaining).Error)
	require.Len(t, remaining, 5)
	assert.Equal(t, "Max", remaining[0].Name)
	assert.Equal(t, target.ID, remaining[0].ContactID)
	assert.Equal(t, "2015-04-01", remaining[0].Birthday, "the dropped duplicate's birthday is kept")
	assert.Equal(t, "Tom", remaining[1].Name)
	assert.Equal(t, "Tom", remaining[2].Name)
	assert.Equal(t, "Lena", remaining[3].Name)
	assert.Equal(t, target.ID, remaining[3].ContactID)
	assert.Equal(t, friend.ID, remaining[4].ContactID)
	require.NotNil(t, remaining[4].RelatedContactID)
	assert.Equal(t, target.ID, *remaining[4].RelatedContactID)
}

func TestMergeContactsDeceasedAndArchivedFollowResolutions(t *testing.T) {
	db, userID := setupMergeDB(t)
	newPair := func() (models.Contact, models.Contact) {
		target := createContact(t, db, models.Contact{UserID: userID, Firstname: "Old", Archived: true})
		source := createContact(t, db, models.Contact{UserID: userID, Firstname: "Old", IsDeceased: true, DeceasedDate: "2020-05-01",
			Photo: "only-source.jpg"})
		reminder := models.Reminder{UserID: userID, Message: "visit", RemindAt: time.Now(), Recurrence: "once", ContactID: &source.ID}
		require.NoError(t, db.Create(&reminder).Error)
		return target, source
	}

	target, source := newPair()
	assert.ElementsMatch(t, []models.MergeConflict{
		{Field: "is_deceased", TargetValue: "false", SourceValue: "true"},
		{Field: "archived", TargetValue: "true", SourceValue: "false"},
	}, MergeConflicts(&target, &source))

	// Unresolved: the target's state wins, so the result stays archived (and,
	// like any archived contact, loses its reminders) and is not deceased.
	result, err := MergeContacts(db, userID, target.ID, source.ID, nil)
	require.NoError(t, err)
	assert.True(t, result.Contact.Archived)
	assert.False(t, result.Contact.IsDeceased)
	assert.Empty(t, result.Contact.DeceasedDate, "a death date without the deceased flag is dropped")
	assert.Equal(t, "only-source.jpg", result.Contact.Photo)
	assert.Equal(t, models.Contact{}, result.DiscardedPhoto)
	var reminders int64
	db.Model(&models.Reminder{}).Where("contact_id = ?", target.ID).Count(&reminders)
	assert.Zero(t, reminders)

	target, source = newPair()
	result, err = MergeContacts(db, userID, target.ID, source.ID, map[string]string{"is_deceased": "source", "archived": "source"})
	require.NoError(t, err)
	assert.False(t, result.Contact.Archived)
	assert.True(t, result.Contact.IsDeceased)
	assert.Equal(t, "2020-05-01", result.Contact.DeceasedDate)
	db.Model(&models.Reminder{}).Where("contact_id = ?", target.ID).Count(&reminders)
	assert.EqualValues(t, 1, reminders)
}

func TestMergeConflictsThumbnailOnlyPhotos(t *testing.T) {
	target := &models.Contact{Firstname: "A", PhotoThumbnail: "data:a"}
	source := &models.Contact{Firstname: "A", PhotoThumbnail: "data:b"}
	conflicts := MergeConflicts(target, source)
	require.Len(t, conflicts, 1)
	assert.Equal(t, "photo", conflicts[0].Field)
}

func TestMergeVCardExtra(t *testing.T) {
	target := `{"properties":{"X-SOCIAL":[{"Value":"@anna"}]}}`
	source := `{"properties":{"X-SOCIAL":[{"Value":"@anna"},{"Value":"@annie"}],"X-PET":[{"Value":"Cat"}]}}`

	var merged carddav.VCardExtra
	require.NoError(t, json.Unmarshal([]byte(mergeVCardExtra(target, source)), &merged))
	assert.Equal(t, []string{"@anna", "@annie"}, fieldValues(merged.Properties["X-SOCIAL"]))
	assert.Equal(t, []string{"Cat"}, fieldValues(merged.Properties["X-PET"]))

	assert.Equal(t, source, mergeVCardExtra("", source))
	assert.Equal(t, target, mergeVCardExtra(target, ""))
	assert.Equal(t, "not json", mergeVCardExtra("not json", source), "an unreadable target is left untouched")
}

func fieldValues(fields []vcard.Field) []string {
	values := make([]string, len(fields))
	for i, f := range fields {
		values[i] = f.Value
	}
	return values
}

func TestMergeContactsRejectsInvalidPairs(t *testing.T) {
	db, userID := setupMergeDB(t)
	a := createContact(t, db, models.Contact{UserID: userID, Firstname: "A"})

	other := models.User{Username: "other", Password: "password123", Email: "other@example.com"}
	require.NoError(t, db.Create(&other).Error)
	foreign := createContact(t, db, models.Contact{UserID: other.ID, Firstname: "B"})

	_, err := MergeContacts(db, userID, a.ID, a.ID, nil)
	assert.ErrorIs(t, err, ErrMergeSameContact)

	_, err = MergeContacts(db, userID, a.ID, foreign.ID, nil)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	assert.NoError(t, db.First(&models.Contact{}, foreign.ID).Error, "another user's contact is untouched")
}

func TestMergeContactsResultReflectsMovedEmployment(t *testing.T) {
	db, userID := setupMergeDB(t)
	target := createContact(t, db, models.Contact{UserID: userID, Firstname: "Anna"})
	source := createContact(t, db, models.Contact{UserID: userID, Firstname: "Anna", Organization: "Acme"})

	result, err := MergeContacts(db, userID, target.ID, source.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, "Acme", result.Contact.Organization)
}
