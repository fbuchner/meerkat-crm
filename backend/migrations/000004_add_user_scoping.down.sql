DROP INDEX IF EXISTS idx_contacts_user_id;
DROP INDEX IF EXISTS idx_notes_user_id;
DROP INDEX IF EXISTS idx_reminders_user_id;
DROP INDEX IF EXISTS idx_relationships_user_id;
DROP INDEX IF EXISTS idx_activities_user_id;

ALTER TABLE contacts DROP COLUMN user_id;
ALTER TABLE notes DROP COLUMN user_id;
ALTER TABLE reminders DROP COLUMN user_id;
ALTER TABLE relationships DROP COLUMN user_id;
ALTER TABLE activities DROP COLUMN user_id;
