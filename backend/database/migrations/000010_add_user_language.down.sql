-- Remove language preference column from users table
-- Note: SQLite doesn't support DROP COLUMN in older versions, but modern SQLite 3.35+ does
ALTER TABLE users DROP COLUMN language;
