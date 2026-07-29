-- Interaction fields on Activity (WP-84, docs/fork-plan/91-envelope-data-model.md §91.7).
-- Generalizes the existing Activity table in place rather than replacing it: uuid/type/
-- external_ref are additive columns alongside the existing INTEGER PK, matching how
-- contacts.vcard_uid (000008/000009) was added to a table with existing production rows.
ALTER TABLE activities ADD COLUMN uuid TEXT;
ALTER TABLE activities ADD COLUMN type TEXT;
ALTER TABLE activities ADD COLUMN external_ref TEXT;

CREATE INDEX IF NOT EXISTS idx_activities_uuid ON activities(uuid);

-- Backfill uuid for existing activities that don't have one, same randomblob-based
-- UUID generation 000009_backfill_carddav_fields used for contacts.vcard_uid.
UPDATE activities
SET uuid = lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab', abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))
WHERE uuid IS NULL OR uuid = '';
