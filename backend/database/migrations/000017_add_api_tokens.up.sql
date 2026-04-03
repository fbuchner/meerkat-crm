CREATE TABLE IF NOT EXISTS api_tokens (
    id           INTEGER  PRIMARY KEY AUTOINCREMENT,
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at   DATETIME,
    user_id      INTEGER  NOT NULL,
    name         TEXT     NOT NULL,
    token_hash   TEXT     NOT NULL UNIQUE,
    last_used_at DATETIME,
    revoked_at   DATETIME,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX idx_api_tokens_user_id    ON api_tokens(user_id);
CREATE INDEX idx_api_tokens_token_hash ON api_tokens(token_hash);
