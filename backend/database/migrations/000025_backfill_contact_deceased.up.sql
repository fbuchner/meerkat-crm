-- Backfill is_deceased for any contact with a "Deceased" custom field
-- (set by the old Monica-import path). Value is either "yes" (date
-- unknown) or a Meerkat-format date string ("YYYY-MM-DD" or "--MM-DD",
-- the latter of which cannot be represented in deceased_date and is
-- dropped, leaving is_deceased true with no date).
UPDATE contacts
SET is_deceased = 1,
    deceased_date = CASE
        WHEN json_extract(custom_fields, '$.Deceased') GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'
            THEN json_extract(custom_fields, '$.Deceased')
        ELSE ''
    END
WHERE json_valid(custom_fields)
  AND json_extract(custom_fields, '$.Deceased') IS NOT NULL;

-- Remove the now-redundant custom field key.
UPDATE contacts
SET custom_fields = json_remove(custom_fields, '$.Deceased')
WHERE json_valid(custom_fields)
  AND json_extract(custom_fields, '$.Deceased') IS NOT NULL;
