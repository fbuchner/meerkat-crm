package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	migrateSQLite "github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run cmd/migrate/main.go [up|down|version]")
	}

	// run's own defers (notably db.Close()) execute before main ever calls
	// log.Fatal, unlike inlining this logic directly in main: log.Fatal(f)
	// calls os.Exit, which skips every pending defer in the process.
	if err := run(os.Args[1]); err != nil {
		log.Fatal(err)
	}
}

func run(command string) error {
	dbPath := "mycorrhizal.db"
	migrationsPath := "file://database/migrations"

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	driver, err := migrateSQLite.WithInstance(db, &migrateSQLite.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(migrationsPath, "sqlite", driver)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	switch command {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("failed to run migrations: %w", err)
		}
		fmt.Println("Migrations applied successfully!")
	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("failed to rollback migrations: %w", err)
		}
		fmt.Println("Migrations rolled back successfully!")
	case "version":
		version, dirty, err := m.Version()
		if err != nil && err != migrate.ErrNilVersion {
			return fmt.Errorf("failed to get migration version: %w", err)
		}
		if err == migrate.ErrNilVersion {
			fmt.Println("No migrations applied yet")
		} else {
			fmt.Printf("Current version: %d (dirty: %v)\n", version, dirty)
		}
	default:
		return fmt.Errorf("unknown command: %s. Use 'up', 'down', or 'version'", command)
	}
	return nil
}
