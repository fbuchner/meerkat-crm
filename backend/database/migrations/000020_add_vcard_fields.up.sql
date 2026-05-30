-- Multi-valued vCard fields on contacts (JSON arrays of typed objects)
ALTER TABLE contacts ADD COLUMN emails TEXT DEFAULT '[]';
ALTER TABLE contacts ADD COLUMN phones TEXT DEFAULT '[]';
ALTER TABLE contacts ADD COLUMN addresses TEXT DEFAULT '[]';
ALTER TABLE contacts ADD COLUMN urls TEXT DEFAULT '[]';
ALTER TABLE contacts ADD COLUMN impps TEXT DEFAULT '[]';

-- Structured name parts (vCard N)
ALTER TABLE contacts ADD COLUMN prefix TEXT DEFAULT '';
ALTER TABLE contacts ADD COLUMN middle_name TEXT DEFAULT '';
ALTER TABLE contacts ADD COLUMN suffix TEXT DEFAULT '';

-- Organizational fields (vCard ORG / TITLE / ROLE)
ALTER TABLE contacts ADD COLUMN organization TEXT DEFAULT '';
ALTER TABLE contacts ADD COLUMN department TEXT DEFAULT '';
ALTER TABLE contacts ADD COLUMN job_title TEXT DEFAULT '';
ALTER TABLE contacts ADD COLUMN role TEXT DEFAULT '';

-- Anniversary date (vCard X-ANNIVERSARY)
ALTER TABLE contacts ADD COLUMN anniversary TEXT DEFAULT '';

-- Backfill the JSON arrays from existing single-valued columns
UPDATE contacts SET emails = json_array(json_object('type', 'home', 'value', email))
  WHERE email IS NOT NULL AND email != '';
UPDATE contacts SET phones = json_array(json_object('type', 'cell', 'value', phone))
  WHERE phone IS NOT NULL AND phone != '';
UPDATE contacts SET addresses = json_array(json_object(
    'type', 'home', 'street', address, 'city', '', 'region', '', 'postal', '', 'country', ''))
  WHERE address IS NOT NULL AND address != '';

-- Per-user setting: which extended contact fields are visible in the UI.
-- NULL means the client applies its default set.
ALTER TABLE users ADD COLUMN enabled_contact_fields TEXT DEFAULT NULL;
