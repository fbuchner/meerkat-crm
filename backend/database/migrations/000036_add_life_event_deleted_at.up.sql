-- Add soft-delete support to life_events, making it consistent with Note,
-- Activity, and Reminder (first-class user-authored content). Hard-deleting
-- a life event leaves no tombstone for change feeds or offline replicas.
ALTER TABLE life_events ADD COLUMN deleted_at DATETIME;
CREATE INDEX idx_life_events_deleted_at ON life_events(deleted_at);
