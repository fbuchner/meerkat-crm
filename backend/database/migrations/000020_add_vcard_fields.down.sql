-- NOTE: ALTER TABLE ... DROP COLUMN requires SQLite >= 3.35.0 (2021-03-12).
-- CAUTION: if 000021 has already run, it moved URL/IMPP/TITLE/ROLE/ANNIVERSARY
-- values out of vcard_extra into these columns and stripped them from vcard_extra.
-- Dropping these columns here permanently discards those values (000021's own down
-- migration is a no-op). Only roll back if that data loss is acceptable.
ALTER TABLE users DROP COLUMN enabled_contact_fields;

ALTER TABLE contacts DROP COLUMN anniversary;
ALTER TABLE contacts DROP COLUMN role;
ALTER TABLE contacts DROP COLUMN job_title;
ALTER TABLE contacts DROP COLUMN department;
ALTER TABLE contacts DROP COLUMN organization;
ALTER TABLE contacts DROP COLUMN suffix;
ALTER TABLE contacts DROP COLUMN middle_name;
ALTER TABLE contacts DROP COLUMN prefix;
ALTER TABLE contacts DROP COLUMN impps;
ALTER TABLE contacts DROP COLUMN urls;
ALTER TABLE contacts DROP COLUMN addresses;
ALTER TABLE contacts DROP COLUMN phones;
ALTER TABLE contacts DROP COLUMN emails;
