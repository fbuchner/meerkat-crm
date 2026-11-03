-- OIDC providers need not supply an email, so several accounts can have an empty one.
-- The UNIQUE constraint on email is inline, and SQLite can only drop that by rebuilding
-- the table. The sequence is carried over so ids of deleted users are never reused.
CREATE TABLE users_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    username TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL,
    email TEXT NOT NULL,
    password_reset_token_hash TEXT,
    password_reset_expires_at DATETIME,
    password_reset_requested_at DATETIME,
    language TEXT DEFAULT 'en',
    custom_field_names TEXT DEFAULT '[]',
    is_admin INTEGER DEFAULT 0,
    date_format TEXT DEFAULT 'eu',
    oidc_subject TEXT,
    oidc_provider TEXT,
    enabled_contact_fields TEXT DEFAULT NULL,
    hide_empty_contact_fields BOOLEAN NOT NULL DEFAULT 0,
    carddav_circle_mode TEXT DEFAULT 'all' NOT NULL,
    carddav_circles TEXT,
    password_changed_at DATETIME
);

INSERT INTO users_new (
    id, created_at, updated_at, deleted_at, username, password, email,
    password_reset_token_hash, password_reset_expires_at, password_reset_requested_at,
    language, custom_field_names, is_admin, date_format, oidc_subject, oidc_provider,
    enabled_contact_fields, hide_empty_contact_fields, carddav_circle_mode, carddav_circles,
    password_changed_at
)
SELECT
    id, created_at, updated_at, deleted_at, username, password, email,
    password_reset_token_hash, password_reset_expires_at, password_reset_requested_at,
    language, custom_field_names, is_admin, date_format, oidc_subject, oidc_provider,
    enabled_contact_fields, hide_empty_contact_fields, carddav_circle_mode, carddav_circles,
    password_changed_at
FROM users;

DELETE FROM sqlite_sequence WHERE name = 'users_new';
INSERT INTO sqlite_sequence (name, seq) SELECT 'users_new', seq FROM sqlite_sequence WHERE name = 'users';

DROP TABLE users;
ALTER TABLE users_new RENAME TO users;

CREATE INDEX idx_users_deleted_at ON users(deleted_at);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_password_reset_token_hash ON users(password_reset_token_hash);
CREATE UNIQUE INDEX idx_users_oidc_subject ON users(oidc_subject, oidc_provider)
    WHERE oidc_subject IS NOT NULL;
CREATE UNIQUE INDEX idx_users_email ON users(email) WHERE email <> '';
