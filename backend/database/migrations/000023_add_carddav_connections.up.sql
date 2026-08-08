CREATE TABLE IF NOT EXISTS carddav_connections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME, updated_at DATETIME,
    user_id INTEGER NOT NULL,
    base_url TEXT NOT NULL,
    address_book_path TEXT NOT NULL,
    address_book_name TEXT NOT NULL DEFAULT '',
    username TEXT NOT NULL DEFAULT '',
    password_encrypted TEXT NOT NULL DEFAULT '',
    direction TEXT NOT NULL DEFAULT 'two_way',
    sync_enabled INTEGER NOT NULL DEFAULT 1,
    sync_token TEXT NOT NULL DEFAULT '',
    last_synced_at DATETIME,
    last_sync_status TEXT NOT NULL DEFAULT '',
    last_sync_error TEXT NOT NULL DEFAULT '',
    -- JSON ContactSyncStats of the last run. Syncs are started in the
    -- background and their outcome is polled, so the counts have to outlive the
    -- request that kicked the run off.
    last_sync_stats TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX idx_carddav_connections_user_id ON carddav_connections(user_id);

CREATE TABLE IF NOT EXISTS carddav_contact_links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME, updated_at DATETIME,
    connection_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    contact_id INTEGER NOT NULL,
    remote_uid TEXT NOT NULL,
    remote_path TEXT NOT NULL,
    remote_etag TEXT NOT NULL DEFAULT '',
    local_hash TEXT NOT NULL DEFAULT '',
    synced_at DATETIME,
    FOREIGN KEY (connection_id) REFERENCES carddav_connections(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX idx_carddav_links_conn_contact ON carddav_contact_links(connection_id, contact_id);
CREATE UNIQUE INDEX idx_carddav_links_conn_uid ON carddav_contact_links(connection_id, remote_uid);
CREATE INDEX idx_carddav_links_user_id ON carddav_contact_links(user_id);
CREATE INDEX idx_carddav_links_contact_id ON carddav_contact_links(contact_id);
