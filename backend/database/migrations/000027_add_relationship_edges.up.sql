-- Relationship graph edge entity (WP-80, docs/fork-plan/91-envelope-data-model.md §91.2).
-- Distinct from the legacy `relationships` table (000001_initial_schema) — that table is
-- untouched by this migration; migrating its data onto this graph is WP-81, not this WP.
--
-- id is TEXT (UUID), not INTEGER AUTOINCREMENT: the first UUID-primary-key entity in this
-- schema, generated in Go (models/relationship_edge.go's BeforeCreate) rather than by SQLite.
--
-- source_id/target_id reference contacts.vcard_uid (added in 000008_add_carddav_fields), not
-- contacts.id — the graph invariant that every relationship endpoint is an entity referenced by
-- its stable UUID (§90 D3), which vcard_uid already is for every contact.
CREATE TABLE IF NOT EXISTS relationship_edges (
    id TEXT PRIMARY KEY,
    created_at DATETIME, updated_at DATETIME,
    user_id INTEGER NOT NULL,
    source_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    type TEXT NOT NULL,
    directional INTEGER NOT NULL DEFAULT 0,
    metadata TEXT,
    source TEXT NOT NULL,
    confidence REAL NOT NULL DEFAULT 1,
    status TEXT NOT NULL,
    sensitivity TEXT NOT NULL DEFAULT 'normal',
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_relationship_edges_user_id ON relationship_edges(user_id);
CREATE INDEX IF NOT EXISTS idx_relationship_edges_source_id ON relationship_edges(source_id);
CREATE INDEX IF NOT EXISTS idx_relationship_edges_target_id ON relationship_edges(target_id);
CREATE INDEX IF NOT EXISTS idx_relationship_edges_type ON relationship_edges(type);
CREATE INDEX IF NOT EXISTS idx_relationship_edges_status ON relationship_edges(status);
CREATE INDEX IF NOT EXISTS idx_relationship_edges_sensitivity ON relationship_edges(sensitivity);
