-- Link reminders back to the life event they were generated from (T5b),
-- so create/update/delete can keep the materialised reminder row in sync.
ALTER TABLE reminders ADD COLUMN life_event_id TEXT;
CREATE INDEX idx_reminders_life_event_id ON reminders(life_event_id);
