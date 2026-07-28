-- Remove the JWT token version from users
ALTER TABLE users DROP COLUMN token_version;
