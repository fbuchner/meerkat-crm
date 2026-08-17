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

-- Best-effort restoration of the "Deceased" custom_field_names entry for
-- any user who has at least one deceased contact. The original name-list
-- membership prior to the up-migration cannot be perfectly reconstructed.
-- Must run before is_deceased is reset below, since it relies on that flag.
UPDATE users
SET custom_field_names = json_insert(
    COALESCE(NULLIF(custom_field_names, ''), '[]'),
    '$[#]',
    'Deceased'
)
WHERE EXISTS (
    SELECT 1 FROM contacts WHERE contacts.user_id = users.id AND contacts.is_deceased = 1
)
AND NOT EXISTS (
    SELECT 1 FROM json_each(COALESCE(NULLIF(users.custom_field_names, ''), '[]')) WHERE value = 'Deceased'
);

UPDATE contacts SET is_deceased = 0, deceased_date = '' WHERE is_deceased = 1;
