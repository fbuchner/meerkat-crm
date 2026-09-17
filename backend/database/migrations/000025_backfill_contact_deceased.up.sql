-- Backfill is_deceased for any contact with a "Deceased" custom field
-- (set by the old Monica-import path). Value is either "yes" (date
-- unknown) or a Meerkat-format date string ("YYYY-MM-DD" or "--MM-DD",
-- the latter of which cannot be represented in deceased_date and is
-- dropped, leaving is_deceased true with no date).
--
-- Custom fields are free-text and user-editable (not exclusively
-- Monica-import output), so we only match values within the documented
-- "Deceased" value domain: "yes", a full YYYY-MM-DD date, or a
-- year-unknown --MM-DD date. Values like "no" or "" must NOT mark a
-- contact deceased.
UPDATE contacts
SET is_deceased = 1,
    deceased_date = CASE
        WHEN json_extract(custom_fields, '$.Deceased') GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'
            THEN json_extract(custom_fields, '$.Deceased')
        ELSE ''
    END
WHERE json_valid(custom_fields)
  AND (json_extract(custom_fields, '$.Deceased') = 'yes'
       OR json_extract(custom_fields, '$.Deceased') GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'
       OR json_extract(custom_fields, '$.Deceased') GLOB '--[0-9][0-9]-[0-9][0-9]');

-- Remove the now-redundant custom field key.
UPDATE contacts
SET custom_fields = json_remove(custom_fields, '$.Deceased')
WHERE json_valid(custom_fields)
  AND (json_extract(custom_fields, '$.Deceased') = 'yes'
       OR json_extract(custom_fields, '$.Deceased') GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'
       OR json_extract(custom_fields, '$.Deceased') GLOB '--[0-9][0-9]-[0-9][0-9]');

-- Remove the stale "Deceased" entry from each user's custom_field_names
-- list so the contact-detail UI stops offering a free-text "Deceased"
-- field that duplicates/contradicts the new structured is_deceased field.
UPDATE users
SET custom_field_names = (
    SELECT json_group_array(value) FROM json_each(users.custom_field_names) WHERE value <> 'Deceased'
)
WHERE json_valid(custom_field_names)
  AND EXISTS (SELECT 1 FROM json_each(users.custom_field_names) WHERE value = 'Deceased');
