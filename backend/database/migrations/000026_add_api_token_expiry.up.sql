-- Add an expiry to API tokens. Previously the only way a token stopped working
-- was explicit revocation, so a leaked token stayed valid indefinitely.
-- NULL means "no expiry" and is retained for rows created before this column
-- existed; new tokens always get a concrete expiry.
ALTER TABLE api_tokens ADD COLUMN expires_at DATETIME;

CREATE INDEX IF NOT EXISTS idx_api_tokens_expires_at ON api_tokens(expires_at);
