DROP INDEX IF EXISTS idx_activities_uuid;

-- Note: SQLite doesn't support DROP COLUMN in older versions (see 000008's
-- own down migration for the same limitation/precedent) -- uuid/type/
-- external_ref columns remain but are unused after downgrade.
