package services

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"meerkat/i18n"
	"meerkat/models"
	"meerkat/monica"
)

// This file maps Monica API entities to Meerkat models. All functions are
// pure (no DB access); linking via contact IDs happens in the session layer.

// Monica avatar sources worth downloading. Photos live behind the Monica API
// (see monica.Client.FetchContactPhoto); gravatars are plain public URLs.
const (
	avatarSourcePhoto    = "photo"
	avatarSourceGravatar = "gravatar"
)

// MonicaMappedContact pairs a mapped contact with its Monica identity.
type MonicaMappedContact struct {
	MonicaID     int
	Contact      models.Contact
	AvatarURL    string
	AvatarSource string // avatarSourcePhoto or avatarSourceGravatar; empty when there is nothing to fetch
}

// MapMonicaContact converts a Monica contact. The avatar URL is returned
// separately because photos are downloaded after the import transaction.
func MapMonicaContact(mc monica.Contact) MonicaMappedContact {
	contact := models.Contact{
		Firstname: strings.TrimSpace(mc.FirstName),
		Lastname:  strings.TrimSpace(mc.LastName),
		Nickname:  strings.TrimSpace(mc.Nickname),
	}
	// Monica allows contacts with only a nickname or last name; Meerkat
	// requires a first name.
	if contact.Firstname == "" {
		if contact.Nickname != "" {
			contact.Firstname = contact.Nickname
		} else if contact.Lastname != "" {
			contact.Firstname = contact.Lastname
			contact.Lastname = ""
		}
	}

	switch mc.GenderType {
	case "M":
		contact.Gender = "male"
	case "F":
		contact.Gender = "female"
	case "O":
		contact.Gender = "other"
	default:
		contact.Gender = NormalizeGender(mc.Gender)
	}

	contact.CustomFields = map[string]string{}
	contact.Birthday = mapMonicaSpecialDate(mc.Information.Dates.Birthdate)
	if mc.Information.Dates.Birthdate.IsAgeBased {
		if age, ok := monicaAgeFromDate(mc.Information.Dates.Birthdate, time.Now()); ok {
			contact.CustomFields[monicaFieldApproximateAge] = fmt.Sprintf("%d", age)
		}
	}
	contact.IsDeceased = mc.IsDead
	if mc.IsDead {
		if d := mapMonicaSpecialDate(mc.Information.Dates.DeceasedDate); d != "" && !strings.HasPrefix(d, "--") {
			contact.DeceasedDate = d
		}
	}
	if mc.IsStarred {
		contact.CustomFields[monicaFieldStarred] = "yes"
	}

	if mc.Information.Career.Job != nil {
		contact.JobTitle = truncateRunes(strings.TrimSpace(*mc.Information.Career.Job), 200)
	}
	if mc.Information.Career.Company != nil {
		contact.Organization = truncateRunes(strings.TrimSpace(*mc.Information.Career.Company), 200)
	}
	contact.FoodPreference = truncateRunes(strings.TrimSpace(mc.FoodPreferences), 500)
	if mc.Information.HowYouMet.GeneralInformation != nil {
		contact.HowWeMet = truncateRunes(strings.TrimSpace(*mc.Information.HowYouMet.GeneralInformation), 1000)
	}
	contact.ContactInformation = truncateRunes(strings.TrimSpace(mc.Description), 1000)

	for _, tag := range mc.Tags {
		if name := strings.TrimSpace(tag.Name); name != "" {
			contact.Circles = append(contact.Circles, name)
		}
	}

	for _, addr := range mc.Addresses {
		mapped := models.ContactAddress{
			Type:   strings.ToLower(strings.TrimSpace(addr.Name)),
			Street: strings.TrimSpace(addr.Street),
			City:   strings.TrimSpace(addr.City),
			Region: strings.TrimSpace(addr.Province),
			Postal: strings.TrimSpace(addr.PostalCode),
		}
		if addr.Country != nil {
			mapped.Country = strings.TrimSpace(addr.Country.Name)
		}
		if mapped.Street == "" && mapped.City == "" && mapped.Region == "" && mapped.Postal == "" && mapped.Country == "" {
			continue
		}
		contact.Addresses = append(contact.Addresses, mapped)
	}

	for _, field := range mc.ContactFields {
		mapMonicaContactField(&contact, field)
	}

	// Mirror primaries so duplicate detection and list views work, matching
	// what Contact.BeforeSave does on regular saves.
	if len(contact.Emails) > 0 {
		contact.Email = contact.Emails[0].Value
	}
	if len(contact.Phones) > 0 {
		contact.Phone = contact.Phones[0].Value
	}
	if len(contact.Addresses) > 0 {
		contact.Address = models.FormatAddress(contact.Addresses[0])
	}

	avatarURL, avatarSource := "", ""
	if mc.Information.Avatar.URL != nil && mc.Information.Avatar.Source != nil {
		// "default" and "adorable" are generated placeholder avatars.
		if src := *mc.Information.Avatar.Source; src == avatarSourcePhoto || src == avatarSourceGravatar {
			avatarURL = *mc.Information.Avatar.URL
			avatarSource = src
		}
	}

	return MonicaMappedContact{MonicaID: mc.ID, Contact: contact, AvatarURL: avatarURL, AvatarSource: avatarSource}
}

// Custom field names created by the Monica import (added to the user's
// CustomFieldNames on confirm).
const (
	monicaFieldStarred        = "Starred"
	monicaFieldApproximateAge = "Approximate age"
)

// mapMonicaContactField routes a denormalized Monica contact field into the
// matching multi-value list (email, phone, URL, or instant messaging/social).
func mapMonicaContactField(contact *models.Contact, field monica.ContactField) {
	content := strings.TrimSpace(field.Content)
	if content == "" {
		return
	}
	fieldType := field.ContactFieldType
	protocol := ""
	if fieldType.Protocol != nil {
		protocol = *fieldType.Protocol
	}
	kind := ""
	if fieldType.Type != nil {
		kind = *fieldType.Type
	}
	label := strings.ToLower(strings.TrimSpace(fieldType.Name))

	switch {
	case protocol == "mailto:" || kind == "email":
		contact.Emails = append(contact.Emails, models.ContactEmail{Type: "home", Value: content})
	case protocol == "tel:" || kind == "phone":
		contact.Phones = append(contact.Phones, models.ContactPhone{Type: "cell", Value: content})
	case isMonicaHTTPURL(content):
		contact.URLs = append(contact.URLs, models.ContactURL{Type: label, Value: content})
	default:
		contact.IMPPs = append(contact.IMPPs, models.ContactIMPP{Type: label, Value: content})
	}
}

func isMonicaHTTPURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// mapMonicaSpecialDate converts Monica's flexible date to Meerkat's birthday
// format: YYYY-MM-DD, --MM-DD when the year is unknown, or "" when the date
// is absent or age-based (age is only an estimate, not a date).
func mapMonicaSpecialDate(d monica.SpecialDate) string {
	if d.Date == nil || d.IsAgeBased {
		return ""
	}
	date := *d.Date
	if len(date) < 10 {
		return ""
	}
	date = date[:10]
	if !IsValidBirthdayFormat(date) {
		return ""
	}
	if d.IsYearUnknown {
		return "--" + date[5:]
	}
	return date
}

func monicaAgeFromDate(d monica.SpecialDate, now time.Time) (int, bool) {
	if d.Date == nil {
		return 0, false
	}
	parsed, ok := monicaParseTime(*d.Date)
	if !ok {
		return 0, false
	}
	age := now.Year() - parsed.Year()
	if age < 0 || age > 150 {
		return 0, false
	}
	return age, true
}

// monicaParseTime parses the timestamp formats Monica uses across endpoints.
func monicaParseTime(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// truncateRunes shortens s to at most n runes (multi-byte safe).
func truncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

// MapMonicaActivity converts an activity; attendee linking happens later via
// the Monica→Meerkat contact ID map. ok is false when the date is unusable.
func MapMonicaActivity(ma monica.Activity, lang string) (activity models.Activity, attendeeIDs []int, ok bool) {
	date, parsed := monicaParseTime(ma.HappenedAt)
	if !parsed {
		return models.Activity{}, nil, false
	}
	title := strings.TrimSpace(ma.Summary)
	if title == "" && ma.ActivityType != nil {
		title = strings.TrimSpace(ma.ActivityType.Name)
	}
	if title == "" {
		title = i18n.T(lang, "monicaImport.untitledActivity")
	}
	description := strings.TrimSpace(ma.Description)
	if ma.ActivityType != nil && strings.TrimSpace(ma.ActivityType.Name) != "" && strings.TrimSpace(ma.Summary) != "" {
		// Preserve the activity type (Meerkat has no such field).
		typeName := strings.TrimSpace(ma.ActivityType.Name)
		if description == "" {
			description = typeName
		} else {
			description = typeName + "\n" + description
		}
	}
	activity = models.Activity{
		Title:       truncateRunes(title, 200),
		Description: truncateRunes(description, 2000),
		Date:        date,
	}
	for _, attendee := range ma.Attendees.Contacts {
		attendeeIDs = append(attendeeIDs, attendee.ID)
	}
	return activity, attendeeIDs, true
}

// MapMonicaNote converts a note. ok is false for empty or orphaned notes.
func MapMonicaNote(mn monica.Note) (note models.Note, monicaContactID int, ok bool) {
	body := strings.TrimSpace(mn.Body)
	if body == "" || mn.Contact == nil {
		return models.Note{}, 0, false
	}
	date, parsed := monicaParseTime(mn.CreatedAt)
	if !parsed {
		date = time.Now()
	}
	return models.Note{Content: truncateRunes(body, 5000), Date: date}, mn.Contact.ID, true
}

// MapMonicaReminder converts a reminder. Monica's frequency model
// (one_time/week/month/year + frequency_number) is folded into Meerkat's
// fixed recurrence enum. Expired one-time reminders are skipped.
//
// birthdayMonthDay is the "MM-DD" of the contact's birthday, or "" when it
// has none; it identifies Monica's auto-created birthday reminders — see
// isBirthdayReminder.
func MapMonicaReminder(mr monica.Reminder, now time.Time, birthdayMonthDay string) (reminder models.Reminder, monicaContactID int, ok bool) {
	if mr.Contact == nil {
		return models.Reminder{}, 0, false
	}
	dateStr := ""
	if mr.NextExpectedDate != nil && *mr.NextExpectedDate != "" {
		dateStr = *mr.NextExpectedDate
	} else if mr.InitialDate != nil {
		dateStr = *mr.InitialDate
	}
	remindAt, parsed := monicaParseTime(dateStr)
	if !parsed {
		return models.Reminder{}, 0, false
	}

	recurrence := ""
	switch mr.FrequencyType {
	case "one_time":
		recurrence = "once"
	case "week":
		recurrence = "weekly"
	case "month":
		switch mr.FrequencyNumber {
		case 3:
			recurrence = "quarterly"
		case 6:
			recurrence = "six-months"
		default:
			recurrence = "monthly"
		}
	case "year":
		recurrence = "yearly"
	default:
		return models.Reminder{}, 0, false
	}
	if recurrence == "once" && remindAt.Before(now) {
		return models.Reminder{}, 0, false
	}

	message := strings.TrimSpace(mr.Title)
	if desc := strings.TrimSpace(mr.Description); desc != "" {
		if message == "" {
			message = desc
		} else {
			message = message + ": " + desc
		}
	}
	if message == "" {
		return models.Reminder{}, 0, false
	}
	byMail := false
	// Meerkat's default is to reschedule a completed reminder from the
	// completion date. That is wrong for a birthday: marking it done a week
	// late would walk the date forward every year. Birthdays stay pinned to
	// the calendar, matching how Meerkat creates them for a new contact.
	reoccurFromCompletion := !isBirthdayReminder(recurrence, remindAt, birthdayMonthDay)
	reminder = models.Reminder{
		Message:               truncateRunes(message, 500),
		RemindAt:              nextRecurrenceOnOrAfter(remindAt, recurrence, now),
		Recurrence:            recurrence,
		ByMail:                &byMail,
		ReoccurFromCompletion: &reoccurFromCompletion,
	}
	return reminder, mr.Contact.ID, true
}

// nextRecurrenceOnOrAfter moves a recurring reminder to its first occurrence on or after today.
// Monica dates a birthday reminder from the birth date.
func nextRecurrenceOnOrAfter(remindAt time.Time, recurrence string, now time.Time) time.Time {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, remindAt.Location())
	if !remindAt.Before(today) {
		return remindAt
	}
	var after func(periods int) time.Time
	switch recurrence {
	case "weekly":
		after = func(p int) time.Time { return remindAt.AddDate(0, 0, 7*p) }
	case "monthly":
		after = func(p int) time.Time { return addMonths(remindAt, p) }
	case "quarterly":
		after = func(p int) time.Time { return addMonths(remindAt, 3*p) }
	case "six-months":
		after = func(p int) time.Time { return addMonths(remindAt, 6*p) }
	case "yearly":
		after = func(p int) time.Time { return addYears(remindAt, p) }
	default:
		// "once" keeps its date: a past one-time reminder is dropped earlier,
		// and inventing a future date for it would be wrong.
		return remindAt
	}
	next := remindAt
	for p := 1; next.Before(today); p++ {
		next = after(p)
	}
	return next
}

// isBirthdayReminder recognizes Monica's auto-created birthday reminders by
// their shape: a yearly reminder falling on the contact's birthday.
//
// Monica's reminders endpoint exposes no "this is a birthday" flag, and the
// generated title is written in the Monica user's own language, so matching
// the date is the signal that holds regardless of locale.
func isBirthdayReminder(recurrence string, remindAt time.Time, birthdayMonthDay string) bool {
	if recurrence != "yearly" || birthdayMonthDay == "" {
		return false
	}
	return remindAt.Format("01-02") == birthdayMonthDay
}

// birthdayMonthDayByMonicaID indexes the "MM-DD" of every contact's birthday.
// Monica stores year-unknown birthdays as "--MM-DD" and full ones as
// "YYYY-MM-DD"; both end in the month and day.
func birthdayMonthDayByMonicaID(snapshot *monica.Snapshot) map[int]string {
	out := make(map[int]string, len(snapshot.Contacts))
	for _, mc := range snapshot.Contacts {
		if birthday := mapMonicaSpecialDate(mc.Information.Dates.Birthdate); len(birthday) >= 5 {
			out[mc.ID] = birthday[len(birthday)-5:]
		}
	}
	return out
}

// MapMonicaRelationship converts one directed relationship edge of the given
// Monica contact. related is the full Monica contact of the other side when
// it is part of the snapshot (used for gender/birthday enrichment).
func MapMonicaRelationship(rel monica.Relationship, related *monica.Contact) (relationship models.Relationship, otherMonicaID int, ok bool) {
	if rel.OfContact == nil {
		return models.Relationship{}, 0, false
	}
	name := strings.TrimSpace(rel.OfContact.CompleteName)
	if name == "" {
		name = strings.TrimSpace(strings.TrimSpace(rel.OfContact.FirstName) + " " + strings.TrimSpace(rel.OfContact.LastName))
	}
	relType := strings.TrimSpace(rel.RelationshipType.Name)
	if name == "" || relType == "" {
		return models.Relationship{}, 0, false
	}
	relationship = models.Relationship{
		Name: truncateRunes(name, 100),
		Type: truncateRunes(relType, 50),
	}
	if related != nil {
		switch related.GenderType {
		case "M":
			relationship.Gender = "male"
		case "F":
			relationship.Gender = "female"
		case "O":
			relationship.Gender = "other"
		}
		relationship.Birthday = mapMonicaSpecialDate(related.Information.Dates.Birthdate)
	}
	return relationship, rel.OfContact.ID, true
}

// The extra-entity mappers below turn Monica records without a Meerkat
// equivalent into dated notes, rendered in the importing user's language.

// MapMonicaCall converts a logged call into a note; calls without content
// carry no information and are skipped.
func MapMonicaCall(mc monica.Call, lang string) (note models.Note, monicaContactID int, ok bool) {
	content := strings.TrimSpace(mc.Content)
	if content == "" || mc.Contact == nil {
		return models.Note{}, 0, false
	}
	date, parsed := monicaParseTime(mc.CalledAt)
	if !parsed {
		date = time.Now()
	}
	body := i18n.T(lang, "monicaImport.note.call", map[string]string{
		"date":    date.Format("2006-01-02"),
		"content": content,
	})
	return models.Note{Content: truncateRunes(body, 5000), Date: date}, mc.Contact.ID, true
}

// MapMonicaTask converts a task into a note.
func MapMonicaTask(mt monica.Task, lang string) (note models.Note, monicaContactID int, ok bool) {
	title := strings.TrimSpace(mt.Title)
	if title == "" || mt.Contact == nil {
		return models.Note{}, 0, false
	}
	key := "monicaImport.note.task"
	if mt.Completed {
		key = "monicaImport.note.taskCompleted"
	}
	body := i18n.T(lang, key, map[string]string{"title": title})
	if desc := strings.TrimSpace(mt.Description); desc != "" {
		body += "\n" + desc
	}
	date, parsed := monicaParseTime(mt.CreatedAt)
	if !parsed {
		date = time.Now()
	}
	if mt.CompletedAt != nil {
		if completed, okDate := monicaParseTime(*mt.CompletedAt); okDate {
			date = completed
		}
	}
	return models.Note{Content: truncateRunes(body, 5000), Date: date}, mt.Contact.ID, true
}

// MapMonicaGift converts a gift (idea/offered/received) into a note.
func MapMonicaGift(mg monica.Gift, lang string) (note models.Note, monicaContactID int, ok bool) {
	name := strings.TrimSpace(mg.Name)
	if name == "" || mg.Contact == nil {
		return models.Note{}, 0, false
	}
	status := mg.Status
	if status == "" {
		switch {
		case mg.HasBeenReceived != nil && *mg.HasBeenReceived:
			status = "received"
		case mg.HasBeenOffered != nil && *mg.HasBeenOffered:
			status = "offered"
		default:
			status = "idea"
		}
	}
	statusLabel := i18n.T(lang, "monicaImport.gift."+status)
	body := i18n.T(lang, "monicaImport.note.gift", map[string]string{
		"status": statusLabel,
		"name":   name,
	})
	if comment := strings.TrimSpace(mg.Comment); comment != "" {
		body += "\n" + comment
	}
	if giftURL := strings.TrimSpace(mg.URL); giftURL != "" {
		body += "\n" + giftURL
	}
	date := time.Now()
	if mg.Date != nil {
		if d, okDate := monicaParseTime(*mg.Date); okDate {
			date = d
		}
	} else if d, okDate := monicaParseTime(mg.CreatedAt); okDate {
		date = d
	}
	return models.Note{Content: truncateRunes(body, 5000), Date: date}, mg.Contact.ID, true
}

// MapMonicaDebt converts a debt into a note.
func MapMonicaDebt(md monica.Debt, lang string) (note models.Note, monicaContactID int, ok bool) {
	if md.Contact == nil {
		return models.Note{}, 0, false
	}
	amount := strings.TrimSpace(md.AmountWithCurrency)
	if amount == "" {
		amount = fmt.Sprintf("%.2f", md.Amount)
	}
	key := "monicaImport.note.debtTheyOwe"
	if md.InDebt == "yes" {
		key = "monicaImport.note.debtYouOwe"
	}
	body := i18n.T(lang, key, map[string]string{"amount": amount})
	if reason := strings.TrimSpace(md.Reason); reason != "" {
		body += "\n" + reason
	}
	date, parsed := monicaParseTime(md.CreatedAt)
	if !parsed {
		date = time.Now()
	}
	return models.Note{Content: truncateRunes(body, 5000), Date: date}, md.Contact.ID, true
}
