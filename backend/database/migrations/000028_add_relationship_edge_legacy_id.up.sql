-- Bookkeeping for cmd/backfill-relationship-edges (WP-81): tracks which legacy
-- `relationships` row (if any) a given `relationship_edges` row was migrated
-- from, so the migration tool can tell "already migrated" from "not yet" and
-- be safely re-run. NULL for every edge created any other way (user-created,
-- WP-83 household-suggested, ...) — this column is never read by application
-- code, only by the migration tool itself.
ALTER TABLE relationship_edges ADD COLUMN legacy_relationship_id INTEGER;

CREATE UNIQUE INDEX IF NOT EXISTS idx_relationship_edges_legacy_relationship_id
    ON relationship_edges(legacy_relationship_id) WHERE legacy_relationship_id IS NOT NULL;
