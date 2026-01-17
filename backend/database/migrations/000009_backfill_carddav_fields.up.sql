-- Backfill etag for existing contacts that don't have one
-- Format: e-{id}-{updated_at_unix}
UPDATE contacts
SET etag = 'e-' || id || '-' || CAST(strftime('%s', updated_at) AS TEXT)
WHERE etag IS NULL OR etag = '';

-- Backfill vcard_uid for existing contacts that don't have one
-- Using a UUID-like format based on id to ensure uniqueness
UPDATE contacts
SET vcard_uid = lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab', abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))
WHERE vcard_uid IS NULL OR vcard_uid = '';
