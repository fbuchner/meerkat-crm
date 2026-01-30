ALTER TABLE users ADD COLUMN is_admin INTEGER DEFAULT 0;
UPDATE users SET is_admin = 1 WHERE id = 1;
