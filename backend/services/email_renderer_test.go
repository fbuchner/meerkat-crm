package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertWellFormedHTMLDocument does light structural sanity checks that the
// rendered output is a complete, balanced HTML document (a full parser would
// pull in a new dependency for a package that only needs to confirm the
// template didn't produce truncated/mismatched markup).
func assertWellFormedHTMLDocument(t *testing.T, html string) {
	t.Helper()
	assert.True(t, strings.HasPrefix(html, "<!DOCTYPE html>"), "should start with doctype")
	assert.True(t, strings.HasSuffix(strings.TrimSpace(html), "</html>"), "should end with closing html tag")
	assert.Equal(t, 1, strings.Count(html, "<html"))
	assert.Equal(t, 1, strings.Count(html, "</html>"))
	assert.Equal(t, 1, strings.Count(html, "<body"))
	assert.Equal(t, 1, strings.Count(html, "</body>"))
	assert.Equal(t, strings.Count(html, "<table"), strings.Count(html, "</table>"), "table open/close tags should balance")
	assert.Equal(t, strings.Count(html, "<tr>")+strings.Count(html, "<tr "), strings.Count(html, "</tr>"), "tr open/close tags should balance")
}

func TestRenderReminderEmail(t *testing.T) {
	t.Run("renders a well-formed document with both sections empty", func(t *testing.T) {
		html, err := renderReminderEmail(ReminderEmailData{
			RemindersTitle: "Upcoming Reminders",
			BirthdaysTitle: "Upcoming Birthdays",
			ContactLabel:   "Contact",
			Footer:         "You are receiving this because you have an account.",
		})
		require.NoError(t, err)
		assertWellFormedHTMLDocument(t, html)

		// Neither section's {{if}} block should have rendered.
		assert.NotContains(t, html, "Upcoming Reminders")
		assert.NotContains(t, html, "Upcoming Birthdays")
		assert.Contains(t, html, "You are receiving this because you have an account.")
	})

	t.Run("renders the reminders section only", func(t *testing.T) {
		html, err := renderReminderEmail(ReminderEmailData{
			RemindersTitle: "Upcoming Reminders",
			BirthdaysTitle: "Upcoming Birthdays",
			ContactLabel:   "Contact",
			Footer:         "Footer text",
			Reminders: []ReminderItem{
				{Date: "Jan 5", Message: "Follow up on proposal", ContactName: "Ada Lovelace"},
				{Date: "Jan 6", Message: "Send invoice", ContactName: "Grace Hopper"},
			},
		})
		require.NoError(t, err)
		assertWellFormedHTMLDocument(t, html)

		assert.Contains(t, html, "Upcoming Reminders")
		assert.NotContains(t, html, "Upcoming Birthdays")
		assert.Contains(t, html, "Jan 5")
		assert.Contains(t, html, "Follow up on proposal")
		assert.Contains(t, html, "Ada Lovelace")
		assert.Contains(t, html, "Jan 6")
		assert.Contains(t, html, "Send invoice")
		assert.Contains(t, html, "Grace Hopper")
		assert.Contains(t, html, "Contact: Ada Lovelace")

		// With no birthdays, the reminders section should use the
		// no-birthdays bottom padding.
		assert.Contains(t, html, "padding-bottom:0")
	})

	t.Run("renders the birthdays section only, with all badge types", func(t *testing.T) {
		html, err := renderReminderEmail(ReminderEmailData{
			RemindersTitle: "Upcoming Reminders",
			BirthdaysTitle: "Upcoming Birthdays",
			ContactLabel:   "Contact",
			Footer:         "Footer text",
			Birthdays: []BirthdayItem{
				{FormattedDate: "Feb 1", Name: "Alan Turing", DaysText: "Today", BadgeType: "today"},
				{FormattedDate: "Feb 2", Name: "Barbara Liskov", DaysText: "Tomorrow", BadgeType: "tomorrow"},
				{FormattedDate: "Feb 10", Name: "Katherine Johnson", DaysText: "In 12 days", BadgeType: "future"},
			},
		})
		require.NoError(t, err)
		assertWellFormedHTMLDocument(t, html)

		assert.NotContains(t, html, "Upcoming Reminders")
		assert.Contains(t, html, "Upcoming Birthdays")

		assert.Contains(t, html, "Alan Turing")
		assert.Contains(t, html, `class="badge-today"`)
		assert.Contains(t, html, "background-color:#DCFCE7;color:#16A34A;")

		assert.Contains(t, html, "Barbara Liskov")
		assert.Contains(t, html, `class="badge-tomorrow"`)
		assert.Contains(t, html, "background-color:#FEF3C7;color:#D97706;")

		assert.Contains(t, html, "Katherine Johnson")
		assert.Contains(t, html, `class="badge-future"`)
		assert.Contains(t, html, "background-color:#DBEAFE;color:#2563EB;")
	})

	t.Run("falls back to the future badge style for an unrecognized badge type", func(t *testing.T) {
		html, err := renderReminderEmail(ReminderEmailData{
			Birthdays: []BirthdayItem{
				{FormattedDate: "Mar 3", Name: "Edge Case", DaysText: "??", BadgeType: "not-a-real-type"},
			},
		})
		require.NoError(t, err)
		// Class attribute reflects the raw value passed in...
		assert.Contains(t, html, `class="badge-not-a-real-type"`)
		// ...but the inline style falls through the if/else-if chain to the
		// "future" (else) branch since it matches neither "today" nor
		// "tomorrow".
		assert.Contains(t, html, "background-color:#DBEAFE;color:#2563EB;")
	})

	t.Run("renders both sections together with the two-section spacing", func(t *testing.T) {
		html, err := renderReminderEmail(ReminderEmailData{
			RemindersTitle: "Reminders",
			BirthdaysTitle: "Birthdays",
			ContactLabel:   "Contact",
			Footer:         "Footer",
			Reminders: []ReminderItem{
				{Date: "Jan 1", Message: "Msg", ContactName: "Name"},
			},
			Birthdays: []BirthdayItem{
				{FormattedDate: "Jan 2", Name: "Bday Person", DaysText: "Today", BadgeType: "today"},
			},
		})
		require.NoError(t, err)
		assertWellFormedHTMLDocument(t, html)

		assert.Contains(t, html, "Reminders")
		assert.Contains(t, html, "Birthdays")
		// When both sections are present, the reminders section uses the
		// wider bottom padding to separate it from the birthdays section.
		assert.Contains(t, html, "padding-bottom:28px")
	})

	t.Run("HTML-escapes untrusted interpolated values", func(t *testing.T) {
		html, err := renderReminderEmail(ReminderEmailData{
			RemindersTitle: "Reminders",
			ContactLabel:   "Contact",
			Reminders: []ReminderItem{
				{Date: "Jan 1", Message: `<script>alert("xss")</script>`, ContactName: "Me & You"},
			},
		})
		require.NoError(t, err)

		assert.NotContains(t, html, "<script>alert")
		assert.Contains(t, html, "&lt;script&gt;")
		assert.Contains(t, html, "Me &amp; You")
	})
}

func TestRenderPasswordResetEmail(t *testing.T) {
	t.Run("renders a well-formed document with all fields interpolated", func(t *testing.T) {
		html, err := renderPasswordResetEmail(PasswordResetEmailData{
			Intro:       "Hi there,",
			Instruction: "Use the code below to reset your password.",
			Token:       "ABC123XYZ",
			Ignore:      "If you didn't request this, ignore this email.",
			Footer:      "Mycorrhizal CRM",
		})
		require.NoError(t, err)
		assertWellFormedHTMLDocument(t, html)

		assert.Contains(t, html, "Hi there,")
		assert.Contains(t, html, "Use the code below to reset your password.")
		assert.Contains(t, html, "ABC123XYZ")
		// html/template auto-escapes the apostrophe as &#39;.
		assert.Contains(t, html, "If you didn&#39;t request this, ignore this email.")
		assert.Contains(t, html, "Mycorrhizal CRM")
	})

	t.Run("renders without error when all fields are empty", func(t *testing.T) {
		html, err := renderPasswordResetEmail(PasswordResetEmailData{})
		require.NoError(t, err)
		assertWellFormedHTMLDocument(t, html)
	})

	t.Run("HTML-escapes a token containing special characters", func(t *testing.T) {
		html, err := renderPasswordResetEmail(PasswordResetEmailData{
			Token: `<b>&"'</b>`,
		})
		require.NoError(t, err)

		assert.NotContains(t, html, `<b>&"'</b>`)
		assert.Contains(t, html, "&lt;b&gt;")
	})
}
