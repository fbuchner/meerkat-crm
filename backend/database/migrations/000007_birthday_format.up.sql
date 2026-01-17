-- Convert birthday format from DD.MM.YYYY/DD.MM. to YYYY-MM-DD/--MM-DD

-- Convert contacts.birthday: DD.MM.YYYY -> YYYY-MM-DD, DD.MM. -> --MM-DD
UPDATE contacts SET birthday =
  CASE
    -- DD.MM.YYYY (10 chars) -> YYYY-MM-DD
    WHEN LENGTH(birthday) = 10 AND birthday LIKE '__.__.____' THEN
      SUBSTR(birthday, 7, 4) || '-' || SUBSTR(birthday, 4, 2) || '-' || SUBSTR(birthday, 1, 2)
    -- DD.MM. (6 chars) -> --MM-DD
    WHEN LENGTH(birthday) = 6 AND birthday LIKE '__.__.' THEN
      '--' || SUBSTR(birthday, 4, 2) || '-' || SUBSTR(birthday, 1, 2)
    -- Already in ISO format or other format - keep as-is
    ELSE birthday
  END
WHERE birthday IS NOT NULL AND birthday != '';

-- Convert relationships.birthday: DD.MM.YYYY -> YYYY-MM-DD, DD.MM. -> --MM-DD
UPDATE relationships SET birthday =
  CASE
    -- DD.MM.YYYY (10 chars) -> YYYY-MM-DD
    WHEN LENGTH(birthday) = 10 AND birthday LIKE '__.__.____' THEN
      SUBSTR(birthday, 7, 4) || '-' || SUBSTR(birthday, 4, 2) || '-' || SUBSTR(birthday, 1, 2)
    -- DD.MM. (6 chars) -> --MM-DD
    WHEN LENGTH(birthday) = 6 AND birthday LIKE '__.__.' THEN
      '--' || SUBSTR(birthday, 4, 2) || '-' || SUBSTR(birthday, 1, 2)
    -- Already in ISO format or other format - keep as-is
    ELSE birthday
  END
WHERE birthday IS NOT NULL AND birthday != '';
