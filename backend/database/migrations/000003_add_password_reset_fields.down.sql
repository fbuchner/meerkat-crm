-- Remove password reset metadata from users table
DROP INDEX IF EXISTS idx_users_password_reset_token_hash;
ALTER TABLE users DROP COLUMN password_reset_token_hash;
ALTER TABLE users DROP COLUMN password_reset_expires_at;
ALTER TABLE users DROP COLUMN password_reset_requested_at;
