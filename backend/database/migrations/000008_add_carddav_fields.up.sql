-- Add CardDAV fields to contacts table
ALTER TABLE contacts ADD COLUMN vcard_uid TEXT;
ALTER TABLE contacts ADD COLUMN vcard_extra TEXT;
ALTER TABLE contacts ADD COLUMN etag TEXT;

-- Create unique index for vcard_uid per user (SQLite partial index syntax)
CREATE UNIQUE INDEX idx_contacts_vcard_uid_user ON contacts(user_id, vcard_uid) WHERE vcard_uid IS NOT NULL;

-- Create sync token tracking table
CREATE TABLE IF NOT EXISTS carddav_sync (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL UNIQUE,
    sync_token TEXT NOT NULL,
    last_modified DATETIME NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_carddav_sync_user ON carddav_sync(user_id);
