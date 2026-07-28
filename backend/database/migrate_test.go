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

func TestMigrationsAddCredentialLifecycleColumns(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "columns.db")

	assert.True(t, columnExists(t, dbPath, "users", "token_version"),
		"users.token_version is required to invalidate JWTs on password change")
	assert.True(t, columnExists(t, dbPath, "api_tokens", "expires_at"),
		"api_tokens.expires_at is required to bound API token lifetime")
}
