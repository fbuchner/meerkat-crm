package database

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMigrator(t *testing.T, db *sql.DB) *migrate.Migrate {
	t.Helper()
	driver, err := withInstance(db, &sqliteConfig{})
	require.NoError(t, err)
	source, err := iofs.New(migrationsFS, "migrations")
	require.NoError(t, err)
	m, err := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	require.NoError(t, err)
	return m
}

func TestMigration31_AllowsMultipleEmptyEmailsAndKeepsData(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)
	m := newMigrator(t, db)
	require.NoError(t, m.Migrate(30))

	_, err = db.Exec(`INSERT INTO users (id, username, password, email, is_admin, carddav_circles) VALUES
		(1, 'admin', 'hash', 'admin@example.com', 1, '["a"]'),
		(2, 'oidc', '', '', 0, NULL),
		(9, 'gone', 'hash', 'gone@example.com', 0, NULL)`)
	require.NoError(t, err)
	_, err = db.Exec(`DELETE FROM users WHERE id = 9`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO webhooks (user_id, name, url, events, secret) VALUES (1, 'kept', 'https://example.com', '[]', 's')`)
	require.NoError(t, err)

	require.NoError(t, m.Migrate(31))

	var username, email, circles string
	var isAdmin int
	require.NoError(t, db.QueryRow(`SELECT username, email, is_admin, carddav_circles FROM users WHERE id = 1`).Scan(&username, &email, &isAdmin, &circles))
	assert.Equal(t, "admin", username)
	assert.Equal(t, "admin@example.com", email)
	assert.Equal(t, 1, isAdmin)
	assert.Equal(t, `["a"]`, circles)

	var webhooks int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM webhooks WHERE user_id = 1`).Scan(&webhooks))
	assert.Equal(t, 1, webhooks, "rows referencing users must survive the rebuild")

	_, err = db.Exec(`INSERT INTO users (username, password, email) VALUES ('oidc2', '', '')`)
	require.NoError(t, err, "a second account without an email")
	var id int
	require.NoError(t, db.QueryRow(`SELECT id FROM users WHERE username = 'oidc2'`).Scan(&id))
	assert.Equal(t, 10, id, "the id of a deleted user must not be reused")

	_, err = db.Exec(`INSERT INTO users (username, password, email) VALUES ('dupe', '', 'admin@example.com')`)
	assert.Error(t, err, "a real email stays unique")
	_, err = db.Exec(`INSERT INTO users (username, password, email) VALUES ('admin', '', 'other@example.com')`)
	assert.Error(t, err, "a username stays unique")

	require.NoError(t, m.Migrate(30))

	_, err = db.Exec(`INSERT INTO users (username, password, email) VALUES ('oidc3', '', '')`)
	assert.Error(t, err, "rolling back restores unique emails")
	var remaining int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&remaining))
	assert.Equal(t, 3, remaining)
}

func TestMigration32_RevivesRecurringRemindersTheBugEnded(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)
	m := newMigrator(t, db)
	require.NoError(t, m.Migrate(31))

	exec := func(query string) {
		t.Helper()
		_, err := db.Exec(query)
		require.NoError(t, err)
	}
	exec(`INSERT INTO users (id, username, password, email) VALUES (1, 'a', 'x', 'a@example.com')`)
	exec(`INSERT INTO contacts (id, user_id, firstname, lastname) VALUES (1, 1, 'A', ''), (2, 1, 'B', ''), (3, 1, 'C', '')`)
	exec(`UPDATE contacts SET deleted_at = '2026-01-01 00:00:00+00:00' WHERE id = 3`)
	exec(`INSERT INTO contacts (id, user_id, firstname, lastname, is_deceased) VALUES (4, 1, 'D', '', 1)`)

	insert := func(id, contact int, message, recurrence, remindAt string, completed int, deletedAt any) {
		t.Helper()
		_, err := db.Exec(`INSERT INTO reminders (id, user_id, contact_id, message, recurrence, remind_at, completed, email_sent, by_mail, deleted_at)
			VALUES (?, 1, ?, ?, ?, ?, ?, 1, 1, ?)`, id, contact, message, recurrence, remindAt, completed, deletedAt)
		require.NoError(t, err)
	}
	const old = "2020-01-31 00:00:00+00:00"
	insert(1, 1, "weekly", "weekly", old, 1, nil)
	insert(2, 1, "monthly", "monthly", old, 1, nil)
	insert(3, 1, "quarterly", "quarterly", old, 1, nil)
	insert(4, 1, "half", "six-months", old, 1, nil)
	insert(5, 1, "birthday", "yearly", "2020-03-15 00:00:00+00:00", 1, nil)
	insert(6, 2, "replaced", "monthly", old, 1, nil)
	insert(7, 2, "replaced", "monthly", "2030-01-01 00:00:00+00:00", 0, nil)
	insert(8, 2, "Birthday of B", "yearly", "2020-05-05 00:00:00+00:00", 1, nil)
	insert(9, 2, "Geburtstag von B", "yearly", "2031-05-05 00:00:00+00:00", 0, nil)
	insert(10, 2, "twice", "weekly", old, 1, nil)
	insert(11, 2, "twice", "weekly", old, 1, nil)
	insert(12, 3, "deleted contact", "weekly", old, 1, nil)
	insert(13, 1, "deleted reminder", "weekly", old, 1, "2026-01-01 00:00:00+00:00")
	insert(14, 1, "active and overdue", "weekly", old, 0, nil)
	insert(15, 4, "deceased contact", "yearly", old, 1, nil)
	insert(16, 1, "older than the step limit", "weekly", "0001-01-01 00:00:00+00:00", 1, nil)

	require.NoError(t, m.Migrate(32))

	read := func(id int) (time.Time, bool) {
		t.Helper()
		var raw string
		var completed bool
		require.NoError(t, db.QueryRow(`SELECT CAST(remind_at AS TEXT), completed FROM reminders WHERE id = ?`, id).Scan(&raw, &completed))
		at, err := time.Parse("2006-01-02 15:04:05-07:00", raw)
		require.NoError(t, err, "the format GORM writes, which date comparisons rely on")
		return at, completed
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)

	for _, id := range []int{1, 2, 3, 4, 5, 11} {
		at, completed := read(id)
		assert.False(t, completed, "reminder %d", id)
		assert.True(t, at.After(today), "reminder %d must not be due at once, got %s", id, at)
	}
	var mailPending int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM reminders WHERE id = 1 AND email_sent = 0`).Scan(&mailPending))
	assert.Equal(t, 1, mailPending, "the next occurrence must send its mail")

	weekly, _ := read(1)
	assert.Equal(t, time.Friday, weekly.Weekday(), "31 January 2020 was a Friday")
	assert.False(t, weekly.After(today.AddDate(0, 0, 7)), "the first occurrence after today, got %s", weekly)
	monthly, _ := read(2)
	assert.False(t, monthly.After(today.AddDate(0, 1, 3)), "the first occurrence after today, got %s", monthly)
	assert.True(t, monthly.Day() == 31 || monthly.AddDate(0, 0, 1).Day() == 1, "stays on the last day of the month, got %s", monthly)
	quarterly, _ := read(3)
	assert.Equal(t, 1, int(quarterly.Month())%3, "keeps the January, April, July, October cycle, got %s", quarterly)
	birthday, _ := read(5)
	assert.Equal(t, "03-15", birthday.Format("01-02"))

	for id, why := range map[int]string{
		6:  "replaced by an active reminder with the same message",
		8:  "replaced by an active yearly reminder on the same date",
		10: "an identical newer row is revived instead",
		12: "the contact is deleted",
		13: "the reminder is deleted",
		15: "the contact is deceased",
		16: "too old to step forward, and must not fail the migration",
	} {
		at, completed := read(id)
		assert.True(t, completed, why)
		assert.True(t, at.Before(today.AddDate(-1, 0, 0)), why)
	}
	active, completed := read(14)
	assert.False(t, completed)
	assert.Equal(t, 2020, active.Year(), "active reminders keep their date")

	require.NoError(t, m.Migrate(31), "rolling back is a no-op, not a failure")
}
