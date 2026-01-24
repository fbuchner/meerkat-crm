-- SQLite doesn't support DROP COLUMN directly in older versions
-- For SQLite 3.35.0+ (2021-03-12), DROP COLUMN is supported
ALTER TABLE contacts DROP COLUMN custom_fields;
ALTER TABLE users DROP COLUMN custom_field_names;
