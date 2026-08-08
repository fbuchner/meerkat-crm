package services

import (
	"testing"
	"time"

	"meerkat/models"
	"meerkat/monica"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newMonicaImportDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(
		&models.User{}, &models.Contact{}, &models.Activity{},
		&models.Note{}, &models.Reminder{}, &models.Relationship{},
	))
	return db
}

func monicaActivity(summary, happenedAt, description string, attendeeIDs ...int) monica.Activity {
	ma := monica.Activity{Summary: summary, HappenedAt: happenedAt, Description: description}
	for _, id := range attendeeIDs {
		ma.Attendees.Contacts = append(ma.Attendees.Contacts, monica.ContactRef{ID: id})
	}
	return ma
}

// Finding #3: (user_id, title, date) collapsed genuinely distinct activities.
func TestImportActivitiesDedupeConsidersAttendeesAndDescription(t *testing.T) {
	db := newMonicaImportDB(t)
	alice := models.Contact{UserID: 1, Firstname: "Alice", Lastname: "A"}
	bob := models.Contact{UserID: 1, Firstname: "Bob", Lastname: "B"}
	require.NoError(t, db.Create(&alice).Error)
	require.NoError(t, db.Create(&bob).Error)

	snapshot := &monica.Snapshot{Activities: []monica.Activity{
		// Same title and date, different attendees — two real activities.
		monicaActivity("Lunch", "2026-03-01", "", 10),
		monicaActivity("Lunch", "2026-03-01", "", 20),
		// Same title, date and attendee, different description.
		monicaActivity("Call", "2026-03-02", "about the move", 10),
		monicaActivity("Call", "2026-03-02", "about the party", 10),
		// A genuine duplicate: identical in every dimension.
		monicaActivity("Call", "2026-03-02", "about the move", 10),
	}}
	idMap := map[int]uint{10: alice.ID, 20: bob.ID}

	m := NewMonicaImportManager()
	result := models.MonicaImportResult{}
	require.NoError(t, m.importActivities(db, 1, "en", snapshot, idMap, &result))

	assert.Equal(t, 4, result.ActivitiesCreated, "only the exact duplicate should collapse")
	assert.Empty(t, result.Errors)

	var stored int64
	require.NoError(t, db.Model(&models.Activity{}).Where("user_id = ?", 1).Count(&stored).Error)
	assert.Equal(t, int64(4), stored)
}

// Finding #3: an unrelated pre-existing activity must not block the import.
func TestImportActivitiesIgnoresUnrelatedExistingActivity(t *testing.T) {
	db := newMonicaImportDB(t)
	alice := models.Contact{UserID: 1, Firstname: "Alice", Lastname: "A"}
	require.NoError(t, db.Create(&alice).Error)

	// Same title and date as the Monica activity, but from another source:
	// no attendees, different description.
	existing := models.Activity{
		UserID:      1,
		Title:       "Meeting",
		Description: "from calendar sync",
		Date:        time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	}
	require.NoError(t, db.Create(&existing).Error)

	snapshot := &monica.Snapshot{Activities: []monica.Activity{
		monicaActivity("Meeting", "2026-03-01", "", 10),
	}}

	m := NewMonicaImportManager()
	result := models.MonicaImportResult{}
	require.NoError(t, m.importActivities(db, 1, "en", snapshot, map[int]uint{10: alice.ID}, &result))

	assert.Equal(t, 1, result.ActivitiesCreated)
}

// Re-running an import must not duplicate what it already created.
func TestImportActivitiesIsIdempotent(t *testing.T) {
	db := newMonicaImportDB(t)
	alice := models.Contact{UserID: 1, Firstname: "Alice", Lastname: "A"}
	require.NoError(t, db.Create(&alice).Error)

	snapshot := &monica.Snapshot{Activities: []monica.Activity{
		monicaActivity("Lunch", "2026-03-01", "at the usual place", 10),
	}}
	idMap := map[int]uint{10: alice.ID}
	m := NewMonicaImportManager()

	first := models.MonicaImportResult{}
	require.NoError(t, m.importActivities(db, 1, "en", snapshot, idMap, &first))
	assert.Equal(t, 1, first.ActivitiesCreated)

	second := models.MonicaImportResult{}
	require.NoError(t, m.importActivities(db, 1, "en", snapshot, idMap, &second))
	assert.Equal(t, 0, second.ActivitiesCreated, "second run should dedupe against the first")

	var stored int64
	require.NoError(t, db.Model(&models.Activity{}).Where("user_id = ?", 1).Count(&stored).Error)
	assert.Equal(t, int64(1), stored)
}

// Finding #8: DuplicateIndex replaced per-row DetectDuplicate calls and must
// return exactly the same answers.
func TestDuplicateIndexMatchesDetectDuplicate(t *testing.T) {
	db := newMonicaImportDB(t)
	seed := []models.Contact{
		{UserID: 1, Firstname: "Ada", Lastname: "Lovelace", Email: "ada@example.com", Phone: "+44 20 1234 5678"},
		{UserID: 1, Firstname: "Grace", Lastname: "Hopper", Email: "grace@example.com"},
		{UserID: 1, Firstname: "Alan", Lastname: "Turing", Phone: "+44 161 999 0000"},
		{UserID: 2, Firstname: "Other", Lastname: "User", Email: "other@example.com"},
	}
	for i := range seed {
		require.NoError(t, db.Create(&seed[i]).Error)
	}

	idx, err := NewDuplicateIndex(db, 1)
	require.NoError(t, err)

	cases := []struct {
		name                              string
		firstname, lastname, email, phone string
	}{
		{"email match", "Augusta", "Byron", "ada@example.com", ""},
		{"email match is case insensitive", "", "", "ADA@EXAMPLE.COM", ""},
		{"name match", "Grace", "Hopper", "", ""},
		{"phone match ignores formatting", "", "", "", "+441619990000"},
		{"email wins over name", "Alan", "Turing", "grace@example.com", ""},
		{"no match", "Nobody", "Here", "nobody@example.com", "+1 555 0000"},
		{"other user's contact is invisible", "", "", "other@example.com", ""},
		{"phone too short to match", "", "", "", "12"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := DetectDuplicate(db, 1, tc.firstname, tc.lastname, tc.email, tc.phone)
			got := idx.Detect(tc.firstname, tc.lastname, tc.email, tc.phone)
			assert.Equal(t, want, got)
		})
	}
}

// Finding #9: dedupe keys are preloaded, so a second pass over the same data
// must not re-create notes or reminders either.
func TestImportNotesAndRemindersDedupeAgainstExistingRows(t *testing.T) {
	db := newMonicaImportDB(t)
	alice := models.Contact{UserID: 1, Firstname: "Alice", Lastname: "A"}
	require.NoError(t, db.Create(&alice).Error)

	date := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	existingNote := models.Note{UserID: 1, ContactID: &alice.ID, Content: "already here", Date: date}
	require.NoError(t, db.Create(&existingNote).Error)

	keys, err := loadNoteKeys(db, 1)
	require.NoError(t, err)
	assert.False(t, keys.addIfNew(noteKey(alice.ID, "already here", date)), "existing note should be known")
	assert.True(t, keys.addIfNew(noteKey(alice.ID, "something new", date)))
	assert.False(t, keys.addIfNew(noteKey(alice.ID, "something new", date)), "keys added in-batch stick")

	remindAt := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	existingReminder := models.Reminder{
		UserID: 1, ContactID: &alice.ID, Message: "call back",
		RemindAt: remindAt, Recurrence: "once",
	}
	require.NoError(t, db.Create(&existingReminder).Error)

	reminderKeys, err := loadReminderKeys(db, 1)
	require.NoError(t, err)
	assert.False(t, reminderKeys.addIfNew(reminderKey(alice.ID, "call back", remindAt)))
	assert.True(t, reminderKeys.addIfNew(reminderKey(alice.ID, "call back", remindAt.Add(time.Hour))))
}

// Finding #6: the cap must bound the whole snapshot, not just contacts.
func TestEntityCountsTotal(t *testing.T) {
	counts := monica.EntityCounts{
		Contacts: 200, Activities: 400000, Notes: 5, Reminders: 5,
		Calls: 1, Tasks: 2, Gifts: 3, Debts: 4,
	}
	assert.Equal(t, 400220, counts.Total())
	assert.Greater(t, counts.Total(), MaxMonicaEntities,
		"an account this size must be rejected even though it is under MaxMonicaContacts")
	assert.LessOrEqual(t, counts.Contacts, MaxMonicaContacts)
}

// monicaRelEdge builds one directed Monica relationship edge: "other is the
// <relType> of the contact whose relationships were fetched".
func monicaRelEdge(otherID int, otherName, relType string) monica.Relationship {
	rel := monica.Relationship{OfContact: &monica.ContactRef{ID: otherID, CompleteName: otherName}}
	rel.RelationshipType.Name = relType
	return rel
}

func importedRelationships(t *testing.T, db *gorm.DB) []models.Relationship {
	t.Helper()
	var rels []models.Relationship
	require.NoError(t, db.Where("user_id = ?", 1).Order("contact_id").Find(&rels).Error)
	return rels
}

// Monica stores every relationship on both contacts; Meerkat's is directed
// and shows on the other contact as incoming, so importing both halves lists
// each relationship twice.
func TestImportRelationshipsCollapsesReciprocalPairs(t *testing.T) {
	newSnapshot := func() *monica.Snapshot {
		return &monica.Snapshot{
			Contacts: []monica.Contact{{ID: 10, FirstName: "Alice"}, {ID: 20, FirstName: "Bob"}},
			Relationships: map[int][]monica.Relationship{
				10: {monicaRelEdge(20, "Bob B", "brother")},
				20: {monicaRelEdge(10, "Alice A", "sister")},
			},
		}
	}

	t.Run("collapsed keeps only the lower-id side", func(t *testing.T) {
		db := newMonicaImportDB(t)
		alice := models.Contact{UserID: 1, Firstname: "Alice", Lastname: "A"}
		bob := models.Contact{UserID: 1, Firstname: "Bob", Lastname: "B"}
		require.NoError(t, db.Create(&alice).Error)
		require.NoError(t, db.Create(&bob).Error)

		m := NewMonicaImportManager()
		result := models.MonicaImportResult{}
		require.NoError(t, m.importRelationships(db, 1, newSnapshot(), map[int]uint{10: alice.ID, 20: bob.ID}, true, &result))

		assert.Equal(t, 1, result.RelationshipsCreated)
		assert.Equal(t, 1, result.RelationshipsCollapsed)

		rels := importedRelationships(t, db)
		require.Len(t, rels, 1)
		assert.Equal(t, alice.ID, rels[0].ContactID, "Monica id 10 is the lower of the pair")
		assert.Equal(t, "brother", rels[0].Type)
		require.NotNil(t, rels[0].RelatedContactID, "the link is what renders the incoming side on Bob")
		assert.Equal(t, bob.ID, *rels[0].RelatedContactID)
	})

	t.Run("not collapsed keeps both halves", func(t *testing.T) {
		db := newMonicaImportDB(t)
		alice := models.Contact{UserID: 1, Firstname: "Alice", Lastname: "A"}
		bob := models.Contact{UserID: 1, Firstname: "Bob", Lastname: "B"}
		require.NoError(t, db.Create(&alice).Error)
		require.NoError(t, db.Create(&bob).Error)

		m := NewMonicaImportManager()
		result := models.MonicaImportResult{}
		require.NoError(t, m.importRelationships(db, 1, newSnapshot(), map[int]uint{10: alice.ID, 20: bob.ID}, false, &result))

		assert.Equal(t, 2, result.RelationshipsCreated)
		assert.Equal(t, 0, result.RelationshipsCollapsed)
		assert.Len(t, importedRelationships(t, db), 2)
	})
}

// A relationship Monica only recorded in one direction has no other half to
// fall back on, so it must survive collapsing whichever way it points.
func TestImportRelationshipsKeepsOneSidedEdges(t *testing.T) {
	db := newMonicaImportDB(t)
	alice := models.Contact{UserID: 1, Firstname: "Alice", Lastname: "A"}
	bob := models.Contact{UserID: 1, Firstname: "Bob", Lastname: "B"}
	require.NoError(t, db.Create(&alice).Error)
	require.NoError(t, db.Create(&bob).Error)

	// Only the higher-id contact records the edge — the direction that the
	// lower-id-wins rule would otherwise discard.
	snapshot := &monica.Snapshot{
		Contacts:      []monica.Contact{{ID: 10, FirstName: "Alice"}, {ID: 20, FirstName: "Bob"}},
		Relationships: map[int][]monica.Relationship{20: {monicaRelEdge(10, "Alice A", "sister")}},
	}

	m := NewMonicaImportManager()
	result := models.MonicaImportResult{}
	require.NoError(t, m.importRelationships(db, 1, snapshot, map[int]uint{10: alice.ID, 20: bob.ID}, true, &result))

	assert.Equal(t, 1, result.RelationshipsCreated)
	assert.Equal(t, 0, result.RelationshipsCollapsed)
	rels := importedRelationships(t, db)
	require.Len(t, rels, 1)
	assert.Equal(t, bob.ID, rels[0].ContactID)
}

// Without the other contact there is no incoming side to show the collapsed
// half, so the edge has to be kept even when its reciprocal exists.
func TestImportRelationshipsKeepsEdgeWhenOtherContactNotImported(t *testing.T) {
	db := newMonicaImportDB(t)
	bob := models.Contact{UserID: 1, Firstname: "Bob", Lastname: "B"}
	require.NoError(t, db.Create(&bob).Error)

	snapshot := &monica.Snapshot{
		Contacts: []monica.Contact{{ID: 10, FirstName: "Alice"}, {ID: 20, FirstName: "Bob"}},
		Relationships: map[int][]monica.Relationship{
			10: {monicaRelEdge(20, "Bob B", "brother")},
			20: {monicaRelEdge(10, "Alice A", "sister")},
		},
	}

	m := NewMonicaImportManager()
	result := models.MonicaImportResult{}
	// Alice (Monica id 10) was skipped in the review step.
	require.NoError(t, m.importRelationships(db, 1, snapshot, map[int]uint{20: bob.ID}, true, &result))

	assert.Equal(t, 1, result.RelationshipsCreated)
	assert.Equal(t, 0, result.RelationshipsCollapsed)
	rels := importedRelationships(t, db)
	require.Len(t, rels, 1)
	assert.Equal(t, bob.ID, rels[0].ContactID)
	assert.Equal(t, "Alice A", rels[0].Name)
	assert.Nil(t, rels[0].RelatedContactID, "Alice has no Meerkat contact to link to")
}

// snapshot.Relationships is a map, so the surviving half must be chosen by a
// rule that does not depend on iteration order.
func TestImportRelationshipsCollapseIsDeterministic(t *testing.T) {
	for run := 0; run < 20; run++ {
		db := newMonicaImportDB(t)
		alice := models.Contact{UserID: 1, Firstname: "Alice", Lastname: "A"}
		bob := models.Contact{UserID: 1, Firstname: "Bob", Lastname: "B"}
		require.NoError(t, db.Create(&alice).Error)
		require.NoError(t, db.Create(&bob).Error)

		snapshot := &monica.Snapshot{
			Contacts: []monica.Contact{{ID: 10, FirstName: "Alice"}, {ID: 20, FirstName: "Bob"}},
			Relationships: map[int][]monica.Relationship{
				10: {monicaRelEdge(20, "Bob B", "brother")},
				20: {monicaRelEdge(10, "Alice A", "sister")},
			},
		}
		m := NewMonicaImportManager()
		result := models.MonicaImportResult{}
		require.NoError(t, m.importRelationships(db, 1, snapshot, map[int]uint{10: alice.ID, 20: bob.ID}, true, &result))

		rels := importedRelationships(t, db)
		require.Len(t, rels, 1)
		require.Equal(t, "brother", rels[0].Type, "run %d picked the other direction", run)
	}
}

// End-to-end through importReminders: the birthday lives on the contact in
// the snapshot, the reminder on a separate endpoint, and only the join of
// the two identifies a birthday reminder.
func TestImportRemindersPinsBirthdaysToTheCalendar(t *testing.T) {
	db := newMonicaImportDB(t)
	alice := models.Contact{UserID: 1, Firstname: "Alice", Lastname: "A"}
	require.NoError(t, db.Create(&alice).Error)

	birthdate := "1815-12-10T00:00:00Z"
	contact := monica.Contact{ID: 10, FirstName: "Alice"}
	contact.Information.Dates.Birthdate = monica.SpecialDate{Date: &birthdate}

	birthdayDate := "2026-12-10"
	otherDate := "2026-07-04"
	snapshot := &monica.Snapshot{
		Contacts: []monica.Contact{contact},
		Reminders: []monica.Reminder{
			{
				Title: "Wish happy birthday", FrequencyType: "year", FrequencyNumber: 1,
				NextExpectedDate: &birthdayDate, Contact: &monica.ContactRef{ID: 10},
			},
			{
				Title: "Yearly check-in", FrequencyType: "year", FrequencyNumber: 1,
				NextExpectedDate: &otherDate, Contact: &monica.ContactRef{ID: 10},
			},
		},
	}

	m := NewMonicaImportManager()
	result := models.MonicaImportResult{}
	require.NoError(t, m.importReminders(db, 1, snapshot, map[int]uint{10: alice.ID}, &result))
	assert.Equal(t, 2, result.RemindersCreated)

	var reminders []models.Reminder
	require.NoError(t, db.Where("user_id = ?", 1).Order("message").Find(&reminders).Error)
	require.Len(t, reminders, 2)

	byMessage := map[string]models.Reminder{}
	for _, r := range reminders {
		byMessage[r.Message] = r
	}

	birthday := byMessage["Wish happy birthday"]
	require.NotNil(t, birthday.ReoccurFromCompletion)
	assert.False(t, *birthday.ReoccurFromCompletion, "a birthday must not drift with the completion date")

	checkIn := byMessage["Yearly check-in"]
	require.NotNil(t, checkIn.ReoccurFromCompletion)
	assert.True(t, *checkIn.ReoccurFromCompletion, "ordinary reminders keep the flexible default")
}
