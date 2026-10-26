-- Per-user setting: on the contact detail page, hide fields that have no value
-- until the user asks to see them all. Defaults to showing every field.
ALTER TABLE users ADD COLUMN hide_empty_contact_fields BOOLEAN NOT NULL DEFAULT 0;
