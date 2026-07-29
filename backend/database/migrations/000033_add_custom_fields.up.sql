-- User-defined custom fields (WP-84b, docs/fork-plan/94-custom-fields.md) -- generalizes the
-- existing untyped v1 (users.custom_field_names, contacts.custom_fields, both untouched by this
-- migration) into typed, validated, optionally standards-projected fields.
--
-- id is TEXT (UUID), matching households (000029) -- generated in Go
-- (models/field_definition.go's BeforeCreate).
CREATE TABLE IF NOT EXISTS field_definitions (
    id TEXT PRIMARY KEY,
    created_at DATETIME, updated_at DATETIME,
    user_id INTEGER NOT NULL,
    label TEXT NOT NULL,
    key TEXT NOT NULL,
    target TEXT NOT NULL,
    type TEXT NOT NULL,
    constraints TEXT,
    projection TEXT NOT NULL DEFAULT 'internal-only',
    sensitivity TEXT NOT NULL DEFAULT 'normal',
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_field_definitions_user_key ON field_definitions(user_id, key);
CREATE INDEX IF NOT EXISTS idx_field_definitions_sensitivity ON field_definitions(sensitivity);

-- field_values is the data half: one typed value of a field_definition for one entity
-- (entity_id is a contacts.vcard_uid, not contacts.id -- the graph invariant, same convention as
-- household_members.member_vcard_uid / contact_tags.contact_vcard_uid).
CREATE TABLE IF NOT EXISTS field_values (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME, updated_at DATETIME,
    field_definition_id TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    entity_id TEXT NOT NULL,
    value TEXT,
    FOREIGN KEY (field_definition_id) REFERENCES field_definitions(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_field_values_definition_entity
    ON field_values(field_definition_id, entity_id);
CREATE INDEX IF NOT EXISTS idx_field_values_user_id ON field_values(user_id);
CREATE INDEX IF NOT EXISTS idx_field_values_entity_id ON field_values(entity_id);
