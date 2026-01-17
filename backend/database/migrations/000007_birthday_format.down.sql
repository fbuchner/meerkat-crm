-- Revert birthday format from YYYY-MM-DD/--MM-DD back to DD.MM.YYYY/DD.MM.

-- Convert contacts.birthday: YYYY-MM-DD -> DD.MM.YYYY, --MM-DD -> DD.MM.
UPDATE contacts SET birthday =
  CASE
    -- YYYY-MM-DD (10 chars) -> DD.MM.YYYY
    WHEN LENGTH(birthday) = 10 AND birthday LIKE '____-__-__' THEN
      SUBSTR(birthday, 9, 2) || '.' || SUBSTR(birthday, 6, 2) || '.' || SUBSTR(birthday, 1, 4)
    -- --MM-DD (7 chars) -> DD.MM.
    WHEN LENGTH(birthday) = 7 AND birthday LIKE '--__-__' THEN
      SUBSTR(birthday, 6, 2) || '.' || SUBSTR(birthday, 3, 2) || '.'
    -- Other format - keep as-is
    ELSE birthday
  END
WHERE birthday IS NOT NULL AND birthday != '';

-- Convert relationships.birthday: YYYY-MM-DD -> DD.MM.YYYY, --MM-DD -> DD.MM.
UPDATE relationships SET birthday =
  CASE
    -- YYYY-MM-DD (10 chars) -> DD.MM.YYYY
    WHEN LENGTH(birthday) = 10 AND birthday LIKE '____-__-__' THEN
      SUBSTR(birthday, 9, 2) || '.' || SUBSTR(birthday, 6, 2) || '.' || SUBSTR(birthday, 1, 4)
    -- --MM-DD (7 chars) -> DD.MM.
    WHEN LENGTH(birthday) = 7 AND birthday LIKE '--__-__' THEN
      SUBSTR(birthday, 6, 2) || '.' || SUBSTR(birthday, 3, 2) || '.'
    -- Other format - keep as-is
    ELSE birthday
  END
WHERE birthday IS NOT NULL AND birthday != '';
