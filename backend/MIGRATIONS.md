# Database Migrations

This project uses [golang-migrate](https://github.com/golang-migrate/migrate) for database schema version control and migrations.

## Overview

Database migrations provide:
- **Version Control**: Track schema changes over time
- **Reproducibility**: Apply same changes across environments
- **Rollback Support**: Revert problematic changes
- **Team Collaboration**: Share schema changes via code
- **CI/CD Integration**: Automate migrations in deployment pipelines

## Migration Files

Migration files are stored in `backend/migrations/` and follow the naming pattern:
```
{version}_{description}.up.sql    # Apply migration
{version}_{description}.down.sql  # Rollback migration
```

Example:
```
000001_initial_schema.up.sql
000001_initial_schema.down.sql
000002_add_user_preferences.up.sql
000002_add_user_preferences.down.sql
```

## Quick Start

### Using Makefile (Recommended)

```bash
# View all available commands
make help

# Run all pending migrations
make migrate-up

# Check current migration version
make migrate-status

# Rollback last migration
make migrate-down

# Create a new migration
make migrate-create NAME=add_profile_photos

# View available migration files
make migrate-version
```

### Using migrate CLI directly

If you have `migrate` CLI installed:

```bash
# Install migrate CLI (macOS)
brew install golang-migrate

# Run migrations
migrate -path ./migrations -database "sqlite3://perema.db" up

# Check version
migrate -path ./migrations -database "sqlite3://perema.db" version

# Rollback
migrate -path ./migrations -database "sqlite3://perema.db" down 1
```

## Application Startup

The application automatically runs pending migrations on startup via `database.InitDB()`:

```go
// main.go
db, err := database.InitDB(cfg.DBPath)
if err != nil {
    logger.Fatal().Err(err).Msg("Failed to initialize database")
}
```

This ensures the database schema is always up-to-date when the application starts.

## Creating New Migrations

### Step 1: Create Migration Files

```bash
make migrate-create NAME=add_user_settings
```

This creates two files:
- `migrations/000002_add_user_settings.up.sql`
- `migrations/000002_add_user_settings.down.sql`

### Step 2: Write SQL for Up Migration

Edit `000002_add_user_settings.up.sql`:

```sql
-- Add user_settings table
CREATE TABLE user_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    theme VARCHAR(50) DEFAULT 'light',
    language VARCHAR(10) DEFAULT 'en',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Add index on user_id
CREATE INDEX idx_user_settings_user_id ON user_settings(user_id);
```

### Step 3: Write SQL for Down Migration

Edit `000002_add_user_settings.down.sql`:

```sql
-- Remove user_settings table
DROP TABLE IF EXISTS user_settings;
```

### Step 4: Test Migration

```bash
# Apply migration
make migrate-up

# Verify it worked
sqlite3 perema.db ".schema user_settings"

# Test rollback
make migrate-down

# Verify table is gone
sqlite3 perema.db ".schema user_settings"

# Re-apply for production
make migrate-up
```

## Best Practices

### 1. **Always Create Both Up and Down**
Every `.up.sql` file must have a corresponding `.down.sql` that reverses the changes.

### 2. **Keep Migrations Small**
One logical change per migration makes debugging easier:
- ✅ Good: One migration adds `user_settings` table
- ❌ Bad: One migration adds 5 tables, 3 indexes, and modifies 2 columns

### 3. **Test Down Migrations**
Always test that your down migration successfully reverses the up migration:
```bash
make migrate-up && make migrate-down && make migrate-up
```

### 4. **Never Modify Existing Migrations**
Once a migration is committed and applied in any environment:
- ❌ Don't edit the SQL
- ✅ Create a new migration to fix issues

### 5. **Use Transactions Carefully**
SQLite doesn't support DDL transactions well. Avoid complex multi-statement migrations.

### 6. **Add Descriptive Names**
```bash
# Good names
make migrate-create NAME=add_photo_storage
make migrate-create NAME=add_email_verification
make migrate-create NAME=rename_circles_to_groups

# Poor names
make migrate-create NAME=update_db
make migrate-create NAME=fix_stuff
```

### 7. **Document Complex Migrations**
Add comments explaining non-obvious changes:
```sql
-- Migration: Add soft delete support to contacts
-- Reason: Support archiving contacts without losing relationship history
-- Related: Issue #123

ALTER TABLE contacts ADD COLUMN deleted_at DATETIME;
CREATE INDEX idx_contacts_deleted_at ON contacts(deleted_at);
```

## Migration Patterns

### Adding a Column
```sql
-- up
ALTER TABLE contacts ADD COLUMN nickname VARCHAR(100);

-- down
ALTER TABLE contacts DROP COLUMN nickname;
```

### Adding an Index
```sql
-- up
CREATE INDEX idx_contacts_email ON contacts(email);

-- down
DROP INDEX IF EXISTS idx_contacts_email;
```

### Creating a Table
```sql
-- up
CREATE TABLE tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(50) NOT NULL UNIQUE,
    color VARCHAR(7),
    created_at DATETIME NOT NULL
);

-- down
DROP TABLE IF EXISTS tags;
```

### Renaming a Column (SQLite)
SQLite doesn't support `ALTER TABLE ... RENAME COLUMN` in older versions:
```sql
-- up
-- Create new table with new column name
CREATE TABLE contacts_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    full_name VARCHAR(255),  -- renamed from firstname
    -- ... other columns
);

-- Copy data
INSERT INTO contacts_new SELECT id, firstname, ... FROM contacts;

-- Replace old table
DROP TABLE contacts;
ALTER TABLE contacts_new RENAME TO contacts;

-- down (reverse the process)
```

### Adding Foreign Key (SQLite)
SQLite doesn't support adding foreign keys to existing tables. Need table recreation:
```sql
-- See "Renaming a Column" pattern above - same approach
```

## Troubleshooting

### Dirty State
If a migration fails midway, the database enters a "dirty" state:

```bash
# Check status
make migrate-status
# Output: 1/d (dirty)

# Force clean state (use cautiously)
make migrate-force VERSION=1

# Then investigate and fix the failed migration
```

### Reset Database
In development, you can reset to a clean state:
```bash
# Delete database
rm perema.db

# Recreate and run all migrations
./perema  # Application auto-runs migrations
```

### Migration Out of Sync
If your database version doesn't match migration files:
```bash
# Check current version
make migrate-status

# Check available migrations
make migrate-version

# Force to correct version if needed
make migrate-force VERSION=2
```

## Production Deployment

### Option 1: Auto-Migrate on Startup (Current)
Application runs migrations automatically when it starts. Simple but requires careful testing.

### Option 2: Manual Migration in CI/CD
Run migrations separately before deploying new code:

```bash
# In deployment pipeline
migrate -path ./migrations -database "sqlite3://prod.db" up

# Then deploy application
./perema
```

### Option 3: Kubernetes Init Container
Use an init container to run migrations before app starts:
```yaml
initContainers:
- name: migrate
  image: migrate/migrate
  command: ["-path", "/migrations", "-database", "sqlite3://prod.db", "up"]
```

## Environment-Specific Migrations

For different databases in different environments:

```bash
# Development
DB_PATH=dev.db make migrate-up

# Staging
DB_PATH=staging.db make migrate-up

# Production
DB_PATH=prod.db make migrate-up
```

## Programmatic Usage

You can also use migrations programmatically in Go code:

```go
import "perema/database"

// Run migrations
db, err := database.InitDB("path/to/db.db")

// Rollback last migration
err := database.MigrateDown("path/to/db.db")
```

## Resources

- [golang-migrate Documentation](https://github.com/golang-migrate/migrate)
- [SQLite SQL Syntax](https://www.sqlite.org/lang.html)
- [Database Migration Best Practices](https://www.oreilly.com/library/view/refactoring-databases/0321293533/)

## FAQ

**Q: Can I skip a migration?**  
A: No, migrations must be applied in order. If you need to skip one, you'll need to delete/modify migration files before running.

**Q: What happens if a migration fails?**  
A: The database enters a "dirty" state and stops applying further migrations. You must fix the issue and force the version.

**Q: Can I run migrations in parallel?**  
A: No, migrations are sequential by design to ensure consistency.

**Q: Should I commit migration files?**  
A: Yes! Migration files are code and should be version-controlled with your application.

**Q: How do I handle data migrations?**  
A: Create a migration with `INSERT`, `UPDATE`, or `DELETE` statements. Test thoroughly!

**Q: What about GORM AutoMigrate?**  
A: We replaced GORM's AutoMigrate with golang-migrate for better control and version management.
