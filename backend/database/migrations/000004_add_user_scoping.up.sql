ALTER TABLE contacts ADD COLUMN user_id INTEGER;
ALTER TABLE notes ADD COLUMN user_id INTEGER;
ALTER TABLE reminders ADD COLUMN user_id INTEGER;
ALTER TABLE relationships ADD COLUMN user_id INTEGER;
ALTER TABLE activities ADD COLUMN user_id INTEGER;

-- Assign existing records to the first user (if any)
UPDATE contacts SET user_id = (
    SELECT id FROM users WHERE deleted_at IS NULL ORDER BY id LIMIT 1
) WHERE user_id IS NULL;
UPDATE notes SET user_id = (
    SELECT id FROM users WHERE deleted_at IS NULL ORDER BY id LIMIT 1
) WHERE user_id IS NULL;
UPDATE reminders SET user_id = (
    SELECT id FROM users WHERE deleted_at IS NULL ORDER BY id LIMIT 1
) WHERE user_id IS NULL;
UPDATE relationships SET user_id = (
    SELECT id FROM users WHERE deleted_at IS NULL ORDER BY id LIMIT 1
) WHERE user_id IS NULL;
UPDATE activities SET user_id = (
    SELECT id FROM users WHERE deleted_at IS NULL ORDER BY id LIMIT 1
) WHERE user_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_contacts_user_id ON contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_notes_user_id ON notes(user_id);
CREATE INDEX IF NOT EXISTS idx_reminders_user_id ON reminders(user_id);
CREATE INDEX IF NOT EXISTS idx_relationships_user_id ON relationships(user_id);
CREATE INDEX IF NOT EXISTS idx_activities_user_id ON activities(user_id);
