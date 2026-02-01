CREATE TABLE IF NOT EXISTS reminder_completions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    user_id INTEGER NOT NULL,
    reminder_id INTEGER,
    contact_id INTEGER NOT NULL,
    message TEXT NOT NULL,
    completed_at DATETIME NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE CASCADE
);
CREATE INDEX idx_reminder_completions_contact_id ON reminder_completions(contact_id);
CREATE INDEX idx_reminder_completions_user_id ON reminder_completions(user_id);
