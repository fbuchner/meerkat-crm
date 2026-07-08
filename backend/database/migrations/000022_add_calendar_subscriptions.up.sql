CREATE TABLE IF NOT EXISTS calendar_subscriptions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME, updated_at DATETIME, deleted_at DATETIME,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    username TEXT NOT NULL DEFAULT '',
    password_encrypted TEXT NOT NULL DEFAULT '',
    sync_enabled INTEGER NOT NULL DEFAULT 1,
    past_days INTEGER NOT NULL DEFAULT 5,
    future_days INTEGER NOT NULL DEFAULT 10,
    last_synced_at DATETIME,
    last_sync_status TEXT NOT NULL DEFAULT '',
    last_sync_error TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX idx_calendar_subscriptions_user_id ON calendar_subscriptions(user_id);
CREATE INDEX idx_calendar_subscriptions_deleted_at ON calendar_subscriptions(deleted_at);

CREATE TABLE IF NOT EXISTS calendar_event_links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME, updated_at DATETIME,
    subscription_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    uid TEXT NOT NULL,
    activity_id INTEGER NOT NULL,
    content_hash TEXT NOT NULL,
    FOREIGN KEY (subscription_id) REFERENCES calendar_subscriptions(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX idx_calendar_event_links_sub_uid ON calendar_event_links(subscription_id, uid);
CREATE INDEX idx_calendar_event_links_user_id ON calendar_event_links(user_id);
CREATE INDEX idx_calendar_event_links_activity_id ON calendar_event_links(activity_id);
