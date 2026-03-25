package services

import (
	"bytes"
	"embed"
	"html/template"
)

//go:embed templates/*.html
var emailTemplatesFS embed.FS

var (
	reminderTmpl      *template.Template
	passwordResetTmpl *template.Template
)

func init() {
	reminderTmpl = template.Must(template.ParseFS(emailTemplatesFS, "templates/reminder.html"))
	passwordResetTmpl = template.Must(template.ParseFS(emailTemplatesFS, "templates/password_reset.html"))
}

// ReminderItem is a single reminder row in the email template.
type ReminderItem struct {
	Date        string
	Message     string
	ContactName string
}

// BirthdayItem is a single birthday row in the email template.
type BirthdayItem struct {
	FormattedDate         string
	Name                  string
	DaysText              string
	BadgeType             string // "today", "tomorrow", "future"
	IsRelationship        bool
	AssociatedContactName string
	RelationshipType      string
}

// ReminderEmailData holds all data passed to the reminder email template.
type ReminderEmailData struct {
	RemindersTitle string
	BirthdaysTitle string
	ContactLabel   string
	Footer         string
	Reminders      []ReminderItem
	Birthdays      []BirthdayItem
}

// PasswordResetEmailData holds all data passed to the password reset email template.
type PasswordResetEmailData struct {
	Intro       string
	Instruction string
	Token       string
	Ignore      string
	Footer      string
}

func renderReminderEmail(data ReminderEmailData) (string, error) {
	var buf bytes.Buffer
	if err := reminderTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func renderPasswordResetEmail(data PasswordResetEmailData) (string, error) {
	var buf bytes.Buffer
	if err := passwordResetTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
