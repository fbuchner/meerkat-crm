package services

import (
	"testing"
	"time"

	"meerkat/i18n"
	"meerkat/monica"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

func monicaTestContact() monica.Contact {
	mc := monica.Contact{
		ID:              42,
		FirstName:       "Ada",
		LastName:        "Lovelace",
		Nickname:        "The Countess",
		GenderType:      "F",
		Gender:          "Woman",
		IsStarred:       true,
		Description:     "Mathematician",
		FoodPreferences: "Vegetarian",
	}
	birthdate := "1815-12-10T00:00:00Z"
	mc.Information.Dates.Birthdate = monica.SpecialDate{Date: &birthdate}
	mc.Information.Career.Job = strPtr("Analyst")
	mc.Information.Career.Company = strPtr("Analytical Engines Ltd")
	mc.Information.HowYouMet.GeneralInformation = strPtr("Met at a lecture")
	mc.Information.Avatar.URL = strPtr("https://monica.example.com/storage/avatar.jpg")
	mc.Information.Avatar.Source = strPtr("photo")
	mc.Tags = []monica.Tag{{Name: "friends"}, {Name: "work"}}
	mc.Addresses = []monica.Address{{
		Name: "Home", Street: "12 St James Square", City: "London", PostalCode: "SW1",
		Country: &struct {
			Name string `json:"name"`
		}{Name: "United Kingdom"},
	}}
	mc.ContactFields = []monica.ContactField{
		{Content: "ada@example.com", ContactFieldType: contactFieldType("Email", "mailto:", "email")},
		{Content: "+44 20 1234 5678", ContactFieldType: contactFieldType("Phone", "tel:", "phone")},
		{Content: "https://ada.example.com", ContactFieldType: contactFieldType("Website", "", "")},
		{Content: "@ada", ContactFieldType: contactFieldType("Twitter", "", "")},
	}
	return mc
}

func contactFieldType(name, protocol, kind string) struct {
	Name     string  `json:"name"`
	Protocol *string `json:"protocol"`
	Type     *string `json:"type"`
} {
	ft := struct {
		Name     string  `json:"name"`
		Protocol *string `json:"protocol"`
		Type     *string `json:"type"`
	}{Name: name}
	if protocol != "" {
		ft.Protocol = &protocol
	}
	if kind != "" {
		ft.Type = &kind
	}
	return ft
}

func TestMapMonicaContactFullMapping(t *testing.T) {
	mapped := MapMonicaContact(monicaTestContact())

	assert.Equal(t, 42, mapped.MonicaID)
	c := mapped.Contact
	assert.Equal(t, "Ada", c.Firstname)
	assert.Equal(t, "Lovelace", c.Lastname)
	assert.Equal(t, "The Countess", c.Nickname)
	assert.Equal(t, "female", c.Gender)
	assert.Equal(t, "1815-12-10", c.Birthday)
	assert.Equal(t, "Analyst", c.JobTitle)
	assert.Equal(t, "Analytical Engines Ltd", c.Organization)
	assert.Equal(t, "Vegetarian", c.FoodPreference)
	assert.Equal(t, "Met at a lecture", c.HowWeMet)
	assert.Equal(t, "Mathematician", c.ContactInformation)
	assert.Equal(t, []string{"friends", "work"}, c.Circles)
	assert.Equal(t, "yes", c.CustomFields["Starred"])

	assert.Len(t, c.Emails, 1)
	assert.Equal(t, "ada@example.com", c.Emails[0].Value)
	assert.Equal(t, "ada@example.com", c.Email) // primary mirrored
	assert.Len(t, c.Phones, 1)
	assert.Equal(t, "+44 20 1234 5678", c.Phone)
	assert.Len(t, c.URLs, 1)
	assert.Equal(t, "https://ada.example.com", c.URLs[0].Value)
	assert.Equal(t, "website", c.URLs[0].Type)
	assert.Len(t, c.IMPPs, 1)
	assert.Equal(t, "twitter", c.IMPPs[0].Type)
	assert.Equal(t, "@ada", c.IMPPs[0].Value)

	assert.Len(t, c.Addresses, 1)
	assert.Equal(t, "home", c.Addresses[0].Type)
	assert.Equal(t, "United Kingdom", c.Addresses[0].Country)
	assert.NotEmpty(t, c.Address)

	assert.Equal(t, "https://monica.example.com/storage/avatar.jpg", mapped.AvatarURL)
}

func TestMapMonicaContactNameFallbacks(t *testing.T) {
	mapped := MapMonicaContact(monica.Contact{Nickname: "Smithy"})
	assert.Equal(t, "Smithy", mapped.Contact.Firstname)

	mapped = MapMonicaContact(monica.Contact{LastName: "Smith"})
	assert.Equal(t, "Smith", mapped.Contact.Firstname)
	assert.Empty(t, mapped.Contact.Lastname)
}

func TestMapMonicaContactGenderFallback(t *testing.T) {
	assert.Equal(t, "male", MapMonicaContact(monica.Contact{GenderType: "M"}).Contact.Gender)
	assert.Equal(t, "other", MapMonicaContact(monica.Contact{GenderType: "O"}).Contact.Gender)
	assert.Equal(t, "male", MapMonicaContact(monica.Contact{Gender: "Male"}).Contact.Gender)
	assert.Equal(t, "", MapMonicaContact(monica.Contact{Gender: "Genderqueer"}).Contact.Gender)
}

func TestMapMonicaContactSpecialDates(t *testing.T) {
	// Year unknown → --MM-DD
	mc := monica.Contact{FirstName: "A"}
	date := "1900-04-20T00:00:00Z"
	mc.Information.Dates.Birthdate = monica.SpecialDate{Date: &date, IsYearUnknown: true}
	assert.Equal(t, "--04-20", MapMonicaContact(mc).Contact.Birthday)

	// Age-based → no birthday, approximate age custom field
	born := time.Now().AddDate(-30, 0, 0).Format(time.RFC3339)
	mc.Information.Dates.Birthdate = monica.SpecialDate{Date: &born, IsAgeBased: true}
	mapped := MapMonicaContact(mc)
	assert.Empty(t, mapped.Contact.Birthday)
	assert.Equal(t, "30", mapped.Contact.CustomFields["Approximate age"])

	// No date at all
	mc.Information.Dates.Birthdate = monica.SpecialDate{}
	assert.Empty(t, MapMonicaContact(mc).Contact.Birthday)
}

func TestMapMonicaContactDeceased(t *testing.T) {
	mc := monica.Contact{FirstName: "A", IsDead: true}
	deceased := "2020-01-15T00:00:00Z"
	mc.Information.Dates.DeceasedDate = monica.SpecialDate{Date: &deceased}
	assert.Equal(t, "2020-01-15", MapMonicaContact(mc).Contact.CustomFields["Deceased"])

	mc.Information.Dates.DeceasedDate = monica.SpecialDate{}
	assert.Equal(t, "yes", MapMonicaContact(mc).Contact.CustomFields["Deceased"])
}

func TestMapMonicaContactSkipsGeneratedAvatars(t *testing.T) {
	mc := monica.Contact{FirstName: "A"}
	mc.Information.Avatar.URL = strPtr("https://monica.example.com/avatar.png")
	mc.Information.Avatar.Source = strPtr("default")
	assert.Empty(t, MapMonicaContact(mc).AvatarURL)

	mc.Information.Avatar.Source = strPtr("adorable")
	assert.Empty(t, MapMonicaContact(mc).AvatarURL)

	mc.Information.Avatar.Source = strPtr("gravatar")
	assert.Equal(t, "https://monica.example.com/avatar.png", MapMonicaContact(mc).AvatarURL)
}

func TestMapMonicaActivity(t *testing.T) {
	assert.NoError(t, i18n.Init())

	ma := monica.Activity{Summary: "Lunch", Description: "At the pub", HappenedAt: "2023-05-01"}
	ma.ActivityType = &struct {
		Name string `json:"name"`
	}{Name: "food"}
	ma.Attendees.Contacts = []monica.ContactRef{{ID: 1}, {ID: 2}}

	activity, attendees, ok := MapMonicaActivity(ma, "en")
	assert.True(t, ok)
	assert.Equal(t, "Lunch", activity.Title)
	assert.Equal(t, "food\nAt the pub", activity.Description)
	assert.Equal(t, 2023, activity.Date.Year())
	assert.Equal(t, []int{1, 2}, attendees)

	// Missing summary falls back to the activity type name
	ma.Summary = ""
	activity, _, ok = MapMonicaActivity(ma, "en")
	assert.True(t, ok)
	assert.Equal(t, "food", activity.Title)

	// No title at all falls back to the i18n placeholder
	ma.ActivityType = nil
	activity, _, ok = MapMonicaActivity(ma, "en")
	assert.True(t, ok)
	assert.Equal(t, "Activity", activity.Title)

	// Unparseable date is skipped
	ma.HappenedAt = "not-a-date"
	_, _, ok = MapMonicaActivity(ma, "en")
	assert.False(t, ok)
}

func TestMapMonicaNote(t *testing.T) {
	note, contactID, ok := MapMonicaNote(monica.Note{
		Body:      "Remember the milk",
		CreatedAt: "2023-01-02T10:00:00Z",
		Contact:   &monica.ContactRef{ID: 5},
	})
	assert.True(t, ok)
	assert.Equal(t, "Remember the milk", note.Content)
	assert.Equal(t, 5, contactID)
	assert.Equal(t, 2023, note.Date.Year())

	_, _, ok = MapMonicaNote(monica.Note{Body: "", Contact: &monica.ContactRef{ID: 5}})
	assert.False(t, ok)
	_, _, ok = MapMonicaNote(monica.Note{Body: "orphan"})
	assert.False(t, ok)
}

func TestMapMonicaReminderFrequencies(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	future := "2026-06-01"
	cases := []struct {
		freqType string
		freqNum  int
		want     string
	}{
		{"one_time", 1, "once"},
		{"week", 1, "weekly"},
		{"month", 1, "monthly"},
		{"month", 3, "quarterly"},
		{"month", 6, "six-months"},
		{"year", 1, "yearly"},
	}
	for _, tc := range cases {
		reminder, contactID, ok := MapMonicaReminder(monica.Reminder{
			Title: "Check in", FrequencyType: tc.freqType, FrequencyNumber: tc.freqNum,
			InitialDate: &future, Contact: &monica.ContactRef{ID: 3},
		}, now, "")
		assert.True(t, ok, tc.freqType)
		assert.Equal(t, tc.want, reminder.Recurrence, "%s/%d", tc.freqType, tc.freqNum)
		assert.Equal(t, 3, contactID)
		assert.Equal(t, "Check in", reminder.Message)
	}

	// Expired one-time reminder is skipped
	past := "2020-01-01"
	_, _, ok := MapMonicaReminder(monica.Reminder{
		Title: "Old", FrequencyType: "one_time", InitialDate: &past, Contact: &monica.ContactRef{ID: 3},
	}, now, "")
	assert.False(t, ok)

	// next_expected_date preferred over initial_date
	next := "2026-09-01"
	reminder, _, ok := MapMonicaReminder(monica.Reminder{
		Title: "Soon", FrequencyType: "month", InitialDate: &past, NextExpectedDate: &next,
		Contact: &monica.ContactRef{ID: 3},
	}, now, "")
	assert.True(t, ok)
	assert.Equal(t, time.September, reminder.RemindAt.Month())
}

func TestMapMonicaRelationship(t *testing.T) {
	rel := monica.Relationship{OfContact: &monica.ContactRef{ID: 9, FirstName: "Jane", LastName: "Doe", CompleteName: "Jane Doe"}}
	rel.RelationshipType.Name = "daughter"

	related := monica.Contact{ID: 9, GenderType: "F"}
	birthdate := "2010-03-05T00:00:00Z"
	related.Information.Dates.Birthdate = monica.SpecialDate{Date: &birthdate}

	relationship, otherID, ok := MapMonicaRelationship(rel, &related)
	assert.True(t, ok)
	assert.Equal(t, "Jane Doe", relationship.Name)
	assert.Equal(t, "daughter", relationship.Type)
	assert.Equal(t, "female", relationship.Gender)
	assert.Equal(t, "2010-03-05", relationship.Birthday)
	assert.Equal(t, 9, otherID)

	_, _, ok = MapMonicaRelationship(monica.Relationship{}, nil)
	assert.False(t, ok)
}

func TestExtraNoteRendering(t *testing.T) {
	assert.NoError(t, i18n.Init())

	note, contactID, ok := MapMonicaCall(monica.Call{
		Content: "Talked about the project", CalledAt: "2023-04-01 12:00:00",
		Contact: &monica.ContactRef{ID: 7},
	}, "en")
	assert.True(t, ok)
	assert.Equal(t, 7, contactID)
	assert.Contains(t, note.Content, "Call on 2023-04-01 (imported from Monica):")
	assert.Contains(t, note.Content, "Talked about the project")

	// German rendering
	note, _, _ = MapMonicaCall(monica.Call{
		Content: "Projekt besprochen", CalledAt: "2023-04-01 12:00:00",
		Contact: &monica.ContactRef{ID: 7},
	}, "de")
	assert.Contains(t, note.Content, "Anruf am 2023-04-01 (aus Monica importiert):")

	// Content-less calls are skipped
	_, _, ok = MapMonicaCall(monica.Call{Contact: &monica.ContactRef{ID: 7}}, "en")
	assert.False(t, ok)

	note, _, ok = MapMonicaTask(monica.Task{
		Title: "Buy gift", Description: "Something nice", Completed: true,
		CompletedAt: strPtr("2023-02-01 09:00:00"), CreatedAt: "2023-01-01 09:00:00",
		Contact: &monica.ContactRef{ID: 7},
	}, "en")
	assert.True(t, ok)
	assert.Contains(t, note.Content, "Completed task (imported from Monica): Buy gift")
	assert.Contains(t, note.Content, "Something nice")
	assert.Equal(t, time.February, note.Date.Month())

	note, _, ok = MapMonicaGift(monica.Gift{
		Name: "Book", Comment: "Loves sci-fi", Status: "idea",
		CreatedAt: "2023-01-01 09:00:00", Contact: &monica.ContactRef{ID: 7},
	}, "en")
	assert.True(t, ok)
	assert.Contains(t, note.Content, "Gift (idea, imported from Monica): Book")
	assert.Contains(t, note.Content, "Loves sci-fi")

	// Legacy boolean gift status
	note, _, _ = MapMonicaGift(monica.Gift{
		Name: "Wine", HasBeenReceived: boolPtr(true),
		CreatedAt: "2023-01-01 09:00:00", Contact: &monica.ContactRef{ID: 7},
	}, "en")
	assert.Contains(t, note.Content, "Gift (received, imported from Monica): Wine")

	note, _, ok = MapMonicaDebt(monica.Debt{
		InDebt: "yes", AmountWithCurrency: "€50.00", Reason: "Dinner",
		CreatedAt: "2023-01-01 09:00:00", Contact: &monica.ContactRef{ID: 7},
	}, "en")
	assert.True(t, ok)
	assert.Contains(t, note.Content, "I owe €50.00")
	assert.Contains(t, note.Content, "Dinner")

	note, _, _ = MapMonicaDebt(monica.Debt{
		InDebt: "no", Amount: 25, CreatedAt: "2023-01-01 09:00:00",
		Contact: &monica.ContactRef{ID: 7},
	}, "en")
	assert.Contains(t, note.Content, "owes me 25.00")
}

func TestTruncateRunes(t *testing.T) {
	assert.Equal(t, "abc", truncateRunes("abc", 5))
	assert.Equal(t, "ab", truncateRunes("abcde", 2))
	assert.Equal(t, "äö", truncateRunes("äöü", 2))
}

// Meerkat reschedules a completed reminder from the completion date, which
// would walk a birthday forward every year it is marked done late. Monica's
// auto-created birthday reminders must come in pinned to the calendar date.
func TestMapMonicaReminderBirthdaysReoccurFromReminderDate(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	birthday := "2026-06-15"

	cases := []struct {
		name             string
		freqType         string
		date             string
		birthdayMonthDay string
		wantFromDate     bool
	}{
		{"birthday reminder", "year", birthday, "06-15", true},
		{"year-unknown birthday still matches", "year", birthday, "06-15", true},
		{"yearly reminder on another day", "year", birthday, "11-02", false},
		{"contact has no birthday", "year", birthday, "", false},
		{"monthly reminder on the birthday", "month", birthday, "06-15", false},
		{"weekly reminder on the birthday", "week", birthday, "06-15", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reminder, _, ok := MapMonicaReminder(monica.Reminder{
				Title: "Wish happy birthday", FrequencyType: tc.freqType, FrequencyNumber: 1,
				InitialDate: &tc.date, Contact: &monica.ContactRef{ID: 3},
			}, now, tc.birthdayMonthDay)
			assert.True(t, ok)
			require.NotNil(t, reminder.ReoccurFromCompletion,
				"the import must decide explicitly; nil falls back to the DB default of true")
			assert.Equal(t, !tc.wantFromDate, *reminder.ReoccurFromCompletion)
		})
	}
}

func TestBirthdayMonthDayByMonicaID(t *testing.T) {
	full := "1815-12-10T00:00:00Z"
	yearUnknown := "1900-03-05T00:00:00Z"
	ageBased := "1980-01-01T00:00:00Z"

	withBirthday := monica.Contact{ID: 1}
	withBirthday.Information.Dates.Birthdate = monica.SpecialDate{Date: &full}

	unknownYear := monica.Contact{ID: 2}
	unknownYear.Information.Dates.Birthdate = monica.SpecialDate{Date: &yearUnknown, IsYearUnknown: true}

	// Monica stores "about 40 years old" without a real date; there is no
	// calendar day to pin a reminder to.
	approximate := monica.Contact{ID: 3}
	approximate.Information.Dates.Birthdate = monica.SpecialDate{Date: &ageBased, IsAgeBased: true}

	none := monica.Contact{ID: 4}

	got := birthdayMonthDayByMonicaID(&monica.Snapshot{
		Contacts: []monica.Contact{withBirthday, unknownYear, approximate, none},
	})

	assert.Equal(t, map[int]string{1: "12-10", 2: "03-05"}, got)
}
