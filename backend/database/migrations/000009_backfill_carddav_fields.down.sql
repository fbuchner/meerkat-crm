-- Down migration: clear backfilled values
-- Note: This will break CardDAV sync for existing contacts if rolled back
UPDATE contacts SET etag = NULL, vcard_uid = NULL;
