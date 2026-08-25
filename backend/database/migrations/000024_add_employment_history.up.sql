CREATE TABLE IF NOT EXISTS employment_histories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    user_id INTEGER NOT NULL,
    contact_id INTEGER NOT NULL,
    organization TEXT NOT NULL DEFAULT '',
    department TEXT NOT NULL DEFAULT '',
    job_title TEXT NOT NULL DEFAULT '',
    role TEXT NOT NULL DEFAULT '',
    start_date TEXT NOT NULL DEFAULT '',
    end_date TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE CASCADE
);
CREATE INDEX idx_employment_histories_contact_id ON employment_histories(contact_id);
CREATE INDEX idx_employment_histories_user_id ON employment_histories(user_id);

-- Backfill: give every contact that already has organizational data one
-- open-ended (current) history entry, so nothing is lost when the
-- Organization/Department/JobTitle/Role fields on contacts become read-only.
INSERT INTO employment_histories (created_at, updated_at, user_id, contact_id, organization, department, job_title, role, start_date, end_date, notes)
SELECT CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, user_id, id, organization, department, job_title, role, '', '', ''
FROM contacts
WHERE deleted_at IS NULL
  AND (COALESCE(organization, '') != '' OR COALESCE(department, '') != '' OR COALESCE(job_title, '') != '' OR COALESCE(role, '') != '');
