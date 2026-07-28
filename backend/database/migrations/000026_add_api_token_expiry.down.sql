-- Remove the API token expiry
DROP INDEX IF EXISTS idx_api_tokens_expires_at;
ALTER TABLE api_tokens DROP COLUMN expires_at;
