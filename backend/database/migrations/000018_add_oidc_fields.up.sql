ALTER TABLE users ADD COLUMN oidc_subject TEXT;
ALTER TABLE users ADD COLUMN oidc_provider TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_oidc_subject ON users(oidc_subject, oidc_provider)
    WHERE oidc_subject IS NOT NULL;
