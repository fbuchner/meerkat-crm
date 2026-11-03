-- A full unique index enforces what the inline constraint did. All but the oldest account
-- without an email get an undeliverable placeholder, since they would otherwise collide.
UPDATE users
SET email = 'no-email-' || id || '@no-email.invalid'
WHERE email = '' AND id <> (SELECT MIN(id) FROM users WHERE email = '');

DROP INDEX idx_users_email;
CREATE UNIQUE INDEX idx_users_email ON users(email);
