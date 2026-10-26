package services

import (
	"meerkat/models"
	"testing"
	"time"

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

	anna1 := createContact(t, db, models.Contact{UserID: userID, Firstname: "Anna", Lastname: "Schmidt",
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
		{UserID: userID, Name: "max", Type: "child", ContactID: source.ID},                                 // same child: dropped
		{UserID: userID, Name: "Lena", Type: "Child", ContactID: source.ID},                                // moved
		{UserID: userID, Name: "Anna", Type: "Friend", ContactID: friend.ID, RelatedContactID: &source.ID}, // incoming: re-pointed
	}
	require.NoError(t, db.Create(&rels).Error)

	preview, err := PreviewContactMerge(db, userID, target.ID, source.ID)
	require.NoError(t, err)
	assert.Equal(t, models.MergeMovedCounts{Notes: 1, Activities: 2, Reminders: 1, Relationships: 2}, preview.Moved)
	assert.Len(t, preview.Conflicts, 4) // lastname, job_title, photo, custom Shoe size

	result, err := MergeContacts(db, userID, target.ID, source.ID, map[string]string{
		"lastname":                "source",
		"photo":                   "source",
		"custom_fields.Shoe size": "target",
		// job_title left unresolved: target wins
	})
	require.NoError(t, err)

	var merged models.Contact
	require.NoError(t, db.First(&merged, target.ID).Error)
	assert.Equal(t, "Schmidt-Weber", merged.Lastname)
	assert.Equal(t, "Annie", merged.Nickname, "a field only the source fills is kept")
	assert.Equal(t, "Engineer", merged.JobTitle)
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
	require.Len(t, remaining, 3)
	assert.Equal(t, "Max", remaining[0].Name)
	assert.Equal(t, target.ID, remaining[0].ContactID)
	assert.Equal(t, "Lena", remaining[1].Name)
	assert.Equal(t, target.ID, remaining[1].ContactID)
	assert.Equal(t, friend.ID, remaining[2].ContactID)
	require.NotNil(t, remaining[2].RelatedContactID)
	assert.Equal(t, target.ID, *remaining[2].RelatedContactID)
}

func TestMergeContactsKeepsActiveAndDeceased(t *testing.T) {
	db, userID := setupMergeDB(t)
	target := createContact(t, db, models.Contact{UserID: userID, Firstname: "Old", Archived: true})
	source := createContact(t, db, models.Contact{UserID: userID, Firstname: "Old", IsDeceased: true, DeceasedDate: "2020-05-01",
		Photo: "only-source.jpg"})

	result, err := MergeContacts(db, userID, target.ID, source.ID, nil)
	require.NoError(t, err)
	assert.False(t, result.Contact.Archived)
	assert.True(t, result.Contact.IsDeceased)
	assert.Equal(t, "2020-05-01", result.Contact.DeceasedDate)
	assert.Equal(t, "only-source.jpg", result.Contact.Photo)
	assert.Equal(t, models.Contact{}, result.DiscardedPhoto)
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
