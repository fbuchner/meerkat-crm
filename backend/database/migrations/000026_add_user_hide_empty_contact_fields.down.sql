-- NOTE: ALTER TABLE ... DROP COLUMN requires SQLite >= 3.35.0 (2021-03-12).
ALTER TABLE users DROP COLUMN hide_empty_contact_fields;
