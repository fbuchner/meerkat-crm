-- Structured personal facts (T20a, docs/fork-plan/91-envelope-data-model.md §91.9) --
-- the generalization of the single free-text contacts.food_preference column (which is
-- migrated into a food-category preference by cmd/backfill-preferences, not dropped here —
-- the column stays until that backfill has demonstrably run).
--
-- id is TEXT (UUID), matching every other UUID-PK entity (households 000029, life_events
-- 000032, field_definitions 000033) -- generated in Go (models/preference.go's BeforeCreate).
--
-- Soft-deletes (deleted_at): preferences are user-authored content about a person, the same
-- shape as life_events (000032), NOT a join-shaped row -- and they carry no natural-key unique
-- constraint, so a soft-deleted row never blocks re-creating the same (entity, category, value).
CREATE TABLE IF NOT EXISTS preferences (
    id TEXT PRIMARY KEY,
    created_at DATETIME, updated_at DATETIME, deleted_at DATETIME,
    user_id INTEGER NOT NULL,
    entity_id TEXT NOT NULL,
    category TEXT NOT NULL,
    key TEXT,
    value TEXT NOT NULL,
    source TEXT,
    confidence REAL,
    last_confirmed DATETIME,
    sensitivity TEXT NOT NULL DEFAULT 'normal',
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_preferences_user_id ON preferences(user_id);
CREATE INDEX IF NOT EXISTS idx_preferences_entity_id ON preferences(entity_id);
CREATE INDEX IF NOT EXISTS idx_preferences_sensitivity ON preferences(sensitivity);
