-- Add a token version to users so credential changes can invalidate issued JWTs.
-- JWTs are stateless and carry no server-side session, so without this a
-- password change or reset left every previously-issued token valid until it
-- expired on its own (up to JWT_EXPIRY_HOURS, 96 by default).
ALTER TABLE users ADD COLUMN token_version INTEGER NOT NULL DEFAULT 0;
