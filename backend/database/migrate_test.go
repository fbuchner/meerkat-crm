package database

import (
	"database/sql"
	"path/filepath"
	"testing"

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
