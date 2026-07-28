-- Neutral RFC 9553/9554/9555 contact model columns (WP-70, P1). These are a
-- second, parallel representation of the same data already stored in the
-- legacy flat/array columns above -- purely additive: nothing existing is
-- removed, renamed, or stops being populated. Populated going forward by
-- Contact.BeforeSave (backend/models/contact.go, via RecordFromContact +
-- contactmodel.DeriveProjection), and backfilled for pre-existing rows by
-- the one-shot cmd/backfill-contact-records tool. See
-- docs/fork-plan/50-integration-and-rebrand.md WP-70.
--
-- card/crm/passthrough hold JSON-serialized contactmodel.Card /
-- contactmodel.CRMEnvelope / contactmodel.Passthrough respectively (same
-- `type:text;serializer:json` GORM convention already used for
-- emails/phones/addresses/urls/impps/circles/custom_fields on this table).
-- fn/org are new plain-text derived projection scalars
-- (contactmodel.Projection.FN / .Org) with no existing legacy analog.
ALTER TABLE contacts ADD COLUMN card TEXT DEFAULT '{}';
ALTER TABLE contacts ADD COLUMN crm TEXT DEFAULT '{}';
ALTER TABLE contacts ADD COLUMN passthrough TEXT DEFAULT '{}';
ALTER TABLE contacts ADD COLUMN fn TEXT DEFAULT '';
ALTER TABLE contacts ADD COLUMN org TEXT DEFAULT '';
