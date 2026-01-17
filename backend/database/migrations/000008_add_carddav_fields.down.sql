-- Remove CardDAV sync table
DROP INDEX IF EXISTS idx_carddav_sync_user;
DROP TABLE IF EXISTS carddav_sync;

-- Remove CardDAV fields from contacts (SQLite requires table recreation)
-- For simplicity, we drop the index and leave the columns (SQLite limitation)
DROP INDEX IF EXISTS idx_contacts_vcard_uid_user;

-- Note: SQLite doesn't support DROP COLUMN in older versions
-- The columns will remain but be unused after downgrade
