-- Fix contacts with empty vcard_uid (created before BeforeCreate hook was added)
UPDATE contacts 
SET vcard_uid = lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab', abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))
WHERE vcard_uid IS NULL OR vcard_uid = '';
