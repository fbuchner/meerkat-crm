-- Add completed field to reminders table
ALTER TABLE reminders ADD COLUMN completed BOOLEAN DEFAULT false NOT NULL;

-- Update recurrence field validation (note: SQLite doesn't support ENUM, validation is done in application layer)
-- No schema change needed for recurrence field, validation is handled by Go validation tags
