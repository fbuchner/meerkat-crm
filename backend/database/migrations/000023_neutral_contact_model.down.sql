-- Reversible: card/crm/passthrough/fn/org (see 000023 up.sql) are a pure
-- addition -- no legacy column was touched to populate them, so dropping
-- them here loses only the neutral-model duplicate, never the source data.
--
-- NOTE: ALTER TABLE ... DROP COLUMN requires SQLite >= 3.35.0 (2021-03-12).
-- This repo's driver (github.com/glebarez/sqlite, backed by
-- modernc.org/sqlite) bundles a version well past that, and this exact
-- direct-DROP-COLUMN approach is already the established precedent in this
-- migrations directory (see 000020_add_vcard_fields.down.sql).
ALTER TABLE contacts DROP COLUMN org;
ALTER TABLE contacts DROP COLUMN fn;
ALTER TABLE contacts DROP COLUMN passthrough;
ALTER TABLE contacts DROP COLUMN crm;
ALTER TABLE contacts DROP COLUMN card;
