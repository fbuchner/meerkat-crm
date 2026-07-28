CREATE TABLE IF NOT EXISTS contact_subscriptions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME, updated_at DATETIME, deleted_at DATETIME,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    username TEXT NOT NULL DEFAULT '',
    password_encrypted TEXT NOT NULL DEFAULT '',
    sync_enabled INTEGER NOT NULL DEFAULT 1,
    sync_token TEXT NOT NULL DEFAULT '',
    last_synced_at DATETIME,
    last_sync_status TEXT NOT NULL DEFAULT '',
    last_sync_error TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_contact_subscriptions_user_id ON contact_subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_contact_subscriptions_deleted_at ON contact_subscriptions(deleted_at);

CREATE TABLE IF NOT EXISTS contact_sync_links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME, updated_at DATETIME,
    subscription_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    href TEXT NOT NULL,
    contact_id INTEGER NOT NULL,
    etag TEXT NOT NULL DEFAULT '',
    content_hash TEXT NOT NULL,
    FOREIGN KEY (subscription_id) REFERENCES contact_subscriptions(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_contact_sync_links_sub_href ON contact_sync_links(subscription_id, href);
CREATE INDEX IF NOT EXISTS idx_contact_sync_links_user_id ON contact_sync_links(user_id);
CREATE INDEX IF NOT EXISTS idx_contact_sync_links_contact_id ON contact_sync_links(contact_id);
