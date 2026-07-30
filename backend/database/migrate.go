package database

import (
	"database/sql"
	"embed"
	"fmt"
	"mycorrhizal/logger"

	"github.com/glebarez/sqlite"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// openDSN appends this app's standard connection pragmas to a plain file
// path:
//   - journal_mode(WAL): what makes docs/deployment.md's backup instructions
//     ("copy the database file while the app is running") actually safe --
//     the default rollback journal can leave a plain `cp` with a torn,
//     inconsistent file mid-write. Persisted in the database file itself
//     once set, so this only needs to run against a real file, never
//     ":memory:".
//   - foreign_keys(1): turns on real FK enforcement (Tier 3c item 8) --
//     unlike journal_mode this is a per-connection setting, not persisted in
//     the file, so it must be supplied via the DSN (applied by the driver on
//     every new physical connection it opens) rather than a one-time
//     PRAGMA statement, which would only affect whichever single connection
//     ran it.
func openDSN(dbPath string) string {
	return dbPath + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
}

// InitDB initializes the database connection and runs migrations
func InitDB(dbPath string) (*gorm.DB, error) {
	// Open database connection for migrations
	sqlDB, err := sql.Open("sqlite", openDSN(dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Run migrations
	if err := RunMigrations(sqlDB); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Open GORM connection
	db, err := gorm.Open(sqlite.Open(openDSN(dbPath)), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect with GORM: %w", err)
	}

	return db, nil
}

// RunMigrations runs all pending database migrations
func RunMigrations(db *sql.DB) error {
	driver, err := withInstance(db, &sqliteConfig{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite", driver)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	// Get current version
	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	if dirty {
		logger.Warn().Uint("version", version).Msg("Database is in dirty state, forcing version")
		if err := m.Force(int(version)); err != nil {
			return fmt.Errorf("failed to force version: %w", err)
		}
	}

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	// Get final version
	version, _, err = m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get final version: %w", err)
	}

	if err == migrate.ErrNilVersion {
		logger.Info().Msg("No migrations applied (database is empty)")
	} else {
		logger.Info().Uint("version", version).Msg("Migrations applied successfully")
	}

	return nil
}

// MigrateDown rolls back the last migration (use with caution)
func MigrateDown(dbPath string) error {
	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer sqlDB.Close()

	driver, err := withInstance(sqlDB, &sqliteConfig{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite", driver)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	if err := m.Steps(-1); err != nil {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	logger.Info().Msg("Migration rolled back successfully")
	return nil
}
