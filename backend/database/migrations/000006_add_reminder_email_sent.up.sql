-- Add email_sent column to track if reminder email was sent for current occurrence
ALTER TABLE reminders ADD COLUMN email_sent BOOLEAN DEFAULT FALSE NOT NULL;
