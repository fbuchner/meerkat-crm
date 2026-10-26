-- Pairs of contacts the user marked as "not a duplicate", so duplicate
-- detection stops suggesting them. contact_a_id is always the smaller id.
CREATE TABLE IF NOT EXISTS contact_duplicate_dismissals (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    user_id INTEGER NOT NULL,
    contact_a_id INTEGER NOT NULL,
    contact_b_id INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX idx_contact_duplicate_dismissals_pair ON contact_duplicate_dismissals(user_id, contact_a_id, contact_b_id);
