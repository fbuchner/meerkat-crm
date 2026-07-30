-- §3d WP5: the legacy `relationships` table is fully replaced by
-- `relationship_edges` (000027_add_relationship_edges) -- CRUD API,
-- GetGraph, the export CSV, and the frontend all read/write RelationshipEdge
-- now. No production data exists yet to carry over (per the user, this WP
-- was scheduled specifically to run before that becomes a concern), so this
-- is a clean drop rather than a data migration.

DROP INDEX IF EXISTS idx_relationships_related_contact_id;
DROP INDEX IF EXISTS idx_relationships_contact_id;
DROP INDEX IF EXISTS idx_relationships_deleted_at;
DROP INDEX IF EXISTS idx_relationships_user_id;

DROP TABLE IF EXISTS relationships;
