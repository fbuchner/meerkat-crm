-- vCard properties URL, IMPP, TITLE, ROLE and ANNIVERSARY used to be unmapped and
-- were therefore stored in the hidden vcard_extra blob. They are now first-class
-- columns. Move any existing values out of vcard_extra into the new columns and
-- strip them from vcard_extra so they are not emitted twice on the next export.
--
-- Only rows whose vcard_extra holds valid, non-empty JSON are touched. Columns are
-- only filled when still empty, so this never clobbers values set after the upgrade.
-- Item-grouped variants (e.g. "item1.URL") and other X-* properties are left in
-- vcard_extra untouched, preserving round-trip fidelity.

-- TITLE -> job_title
UPDATE contacts
SET job_title = json_extract(vcard_extra, '$.properties.TITLE[0].Value')
WHERE job_title = ''
  AND vcard_extra IS NOT NULL AND vcard_extra != '' AND json_valid(vcard_extra)
  AND json_extract(vcard_extra, '$.properties.TITLE[0].Value') IS NOT NULL;

-- ROLE -> role
UPDATE contacts
SET role = json_extract(vcard_extra, '$.properties.ROLE[0].Value')
WHERE role = ''
  AND vcard_extra IS NOT NULL AND vcard_extra != '' AND json_valid(vcard_extra)
  AND json_extract(vcard_extra, '$.properties.ROLE[0].Value') IS NOT NULL;

-- ANNIVERSARY -> anniversary
UPDATE contacts
SET anniversary = json_extract(vcard_extra, '$.properties.ANNIVERSARY[0].Value')
WHERE anniversary = ''
  AND vcard_extra IS NOT NULL AND vcard_extra != '' AND json_valid(vcard_extra)
  AND json_extract(vcard_extra, '$.properties.ANNIVERSARY[0].Value') IS NOT NULL;

-- URL -> urls (array of {type,value}); the original TYPE is not preserved, default to "home"
UPDATE contacts
SET urls = (
  SELECT json_group_array(json_object('type', 'home', 'value', json_extract(value, '$.Value')))
  FROM json_each(vcard_extra, '$.properties.URL')
  WHERE json_extract(value, '$.Value') IS NOT NULL AND json_extract(value, '$.Value') != ''
)
WHERE urls = '[]'
  AND vcard_extra IS NOT NULL AND vcard_extra != '' AND json_valid(vcard_extra)
  AND json_type(vcard_extra, '$.properties.URL') = 'array';

-- IMPP -> impps (service taken from the X-SERVICE-TYPE parameter when present)
UPDATE contacts
SET impps = (
  SELECT json_group_array(json_object(
    'type', COALESCE(json_extract(value, '$.Params."X-SERVICE-TYPE"[0]'), ''),
    'value', json_extract(value, '$.Value')))
  FROM json_each(vcard_extra, '$.properties.IMPP')
  WHERE json_extract(value, '$.Value') IS NOT NULL AND json_extract(value, '$.Value') != ''
)
WHERE impps = '[]'
  AND vcard_extra IS NOT NULL AND vcard_extra != '' AND json_valid(vcard_extra)
  AND json_type(vcard_extra, '$.properties.IMPP') = 'array';

-- Strip the migrated properties from vcard_extra to prevent double emission on export
UPDATE contacts
SET vcard_extra = json_remove(vcard_extra,
  '$.properties.URL', '$.properties.IMPP', '$.properties.TITLE',
  '$.properties.ROLE', '$.properties.ANNIVERSARY')
WHERE vcard_extra IS NOT NULL AND vcard_extra != '' AND json_valid(vcard_extra)
  AND (
    json_type(vcard_extra, '$.properties.URL') IS NOT NULL OR
    json_type(vcard_extra, '$.properties.IMPP') IS NOT NULL OR
    json_type(vcard_extra, '$.properties.TITLE') IS NOT NULL OR
    json_type(vcard_extra, '$.properties.ROLE') IS NOT NULL OR
    json_type(vcard_extra, '$.properties.ANNIVERSARY') IS NOT NULL
  );
