package database

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// columnExists reports whether table has the named column, via SQLite's
// table_info pragma.
func columnExists(t *testing.T, dbPath, table, column string) bool {
	t.Helper()

	db, err := InitDB(dbPath)
	require.NoError(t, err)

	var count int64
	err = db.Raw(
		"SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?", table, column,
	).Scan(&count).Error
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	return count == 1
}

// Applies the full migration chain to an empty database. Guards against a new
// migration that only works against a schema that already has the column, and
// against SQL that parses but fails on this SQLite build.
func TestMigrationsApplyToEmptyDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "migrate.db")

	db, err := InitDB(dbPath)
	require.NoError(t, err, "migrations must apply cleanly to an empty database")

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
}

// TestMigrationDropsLegacyRelationshipsTable is the regression test for §3d
// WP5 (docs/fork-plan/95-backlog-and-priorities.md): migration 000035 must
// actually remove the legacy `relationships` table against the real
// migration chain, not just leave a Go model with no backing table.
func TestMigrationDropsLegacyRelationshipsTable(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "drop-relationships.db")
	db, err := InitDB(dbPath)
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Raw(
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'relationships'",
	).Scan(&count).Error)
	assert.Zero(t, count, "relationships table should be dropped by migration 000035")

	// relationship_edges (its replacement, WP-80) must still be present.
	require.NoError(t, db.Raw(
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'relationship_edges'",
	).Scan(&count).Error)
	assert.Equal(t, int64(1), count, "relationship_edges table should still exist")
}

func TestMigrationsAddCredentialLifecycleColumns(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "columns.db")

	assert.True(t, columnExists(t, dbPath, "users", "token_version"),
		"users.token_version is required to invalidate JWTs on password change")
	assert.True(t, columnExists(t, dbPath, "api_tokens", "expires_at"),
		"api_tokens.expires_at is required to bound API token lifetime")
}

// TestForeignKeysEnforced is the regression test for Tier 3c item 8
// (docs/fork-plan/95-backlog-and-priorities.md): foreign_keys is a
// per-connection SQLite setting, not persisted in the database file, so it
// must be supplied via the DSN on every InitDB call (openDSN) rather than a
// one-time PRAGMA statement.
func TestForeignKeysEnforced(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "fk.db")
	db, err := InitDB(dbPath)
	require.NoError(t, err)

	var enabled int
	require.NoError(t, db.Raw("PRAGMA foreign_keys").Scan(&enabled).Error)
	assert.Equal(t, 1, enabled)
}

// TestForeignKeyCascadeDeletesOrphanedChildRows proves foreign_keys is not
// just set but actually enforced: deleting a circle now auto-cascades its
// circle_members rows at the SQLite level, closing a real orphan-row gap
// DeleteCircle's own code doesn't handle explicitly (household_members/
// contact_tags/field_values have the identical shape and rely on the same
// declared ON DELETE CASCADE).
func TestForeignKeyCascadeDeletesOrphanedChildRows(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "fk-cascade.db")
	db, err := InitDB(dbPath)
	require.NoError(t, err)

	require.NoError(t, db.Exec(
		"INSERT INTO users (created_at, updated_at, username, email, password) VALUES (datetime('now'), datetime('now'), 'u', 'u@example.com', 'x')",
	).Error)
	var userID int64
	require.NoError(t, db.Raw("SELECT id FROM users WHERE username = 'u'").Scan(&userID).Error)

	require.NoError(t, db.Exec(
		"INSERT INTO circles (id, created_at, updated_at, user_id, name) VALUES ('circle-1', datetime('now'), datetime('now'), ?, 'c')",
		userID,
	).Error)
	require.NoError(t, db.Exec(
		"INSERT INTO circle_members (created_at, updated_at, circle_id, user_id, member_vcard_uid) VALUES (datetime('now'), datetime('now'), 'circle-1', ?, 'vcard-1')",
		userID,
	).Error)

	require.NoError(t, db.Exec("DELETE FROM circles WHERE id = 'circle-1'").Error)

	var remaining int64
	require.NoError(t, db.Table("circle_members").Where("circle_id = 'circle-1'").Count(&remaining).Error)
	assert.Zero(t, remaining, "circle_members should be auto-cascaded when its parent circle is deleted")
}
