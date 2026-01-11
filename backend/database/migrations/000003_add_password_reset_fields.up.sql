-- Add password reset metadata to users table
ALTER TABLE users ADD COLUMN password_reset_token_hash TEXT;
ALTER TABLE users ADD COLUMN password_reset_expires_at DATETIME;
ALTER TABLE users ADD COLUMN password_reset_requested_at DATETIME;

CREATE INDEX IF NOT EXISTS idx_users_password_reset_token_hash ON users(password_reset_token_hash);
