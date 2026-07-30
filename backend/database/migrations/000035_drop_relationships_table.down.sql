-- Recreates the cumulative schema from 000001_initial_schema.up.sql +
-- 000004_add_user_scoping.up.sql (the only two migrations that ever touched
-- this table). Reversing this migration restores an empty table, not the
-- data that was in it at drop time -- there is no way to recover dropped
-- rows from a down migration.

CREATE TABLE IF NOT EXISTS relationships (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    gender TEXT,
    birthday TEXT,
    contact_id INTEGER NOT NULL,
    related_contact_id INTEGER,
    user_id INTEGER,
    FOREIGN KEY (contact_id) REFERENCES contacts(id),
    FOREIGN KEY (related_contact_id) REFERENCES contacts(id)
);

CREATE INDEX IF NOT EXISTS idx_relationships_deleted_at ON relationships(deleted_at);
CREATE INDEX IF NOT EXISTS idx_relationships_contact_id ON relationships(contact_id);
CREATE INDEX IF NOT EXISTS idx_relationships_related_contact_id ON relationships(related_contact_id);
CREATE INDEX IF NOT EXISTS idx_relationships_user_id ON relationships(user_id);
