-- LifeEvent entity (WP-84, docs/fork-plan/91-envelope-data-model.md §91.6) -- permanent facts
-- about an entity's life, distinct from Interaction (activities, "what happened between us").
-- id is TEXT (UUID), matching households (000029) -- generated in Go
-- (models/life_event.go's BeforeCreate).
CREATE TABLE IF NOT EXISTS life_events (
    id TEXT PRIMARY KEY,
    created_at DATETIME, updated_at DATETIME,
    user_id INTEGER NOT NULL,
    entity_id TEXT NOT NULL,
    type TEXT,
    date TEXT,
    description TEXT,
    source TEXT,
    related_entity_ids TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_life_events_user_id ON life_events(user_id);
CREATE INDEX IF NOT EXISTS idx_life_events_entity_id ON life_events(entity_id);
