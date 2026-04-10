DROP INDEX IF EXISTS idx_users_oidc_subject;
ALTER TABLE users DROP COLUMN oidc_subject;
ALTER TABLE users DROP COLUMN oidc_provider;
