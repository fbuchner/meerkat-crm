-- Restore the custom field key for any contact currently marked deceased
-- (best-effort: the original "yes" vs. date distinction for
-- year-unknown Monica dates cannot be recovered, so any deceased contact
-- without a full date is restored as "yes").
UPDATE contacts
SET custom_fields = json_set(
    COALESCE(NULLIF(custom_fields, ''), '{}'),
    '$.Deceased',
    CASE WHEN deceased_date != '' THEN deceased_date ELSE 'yes' END
)
WHERE is_deceased = 1;

UPDATE contacts SET is_deceased = 0, deceased_date = '' WHERE is_deceased = 1;
