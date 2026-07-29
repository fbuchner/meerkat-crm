-- Household entity + membership (WP-83, docs/fork-plan/91-envelope-data-model.md §91.3).
-- id is TEXT (UUID), matching relationship_edges (000027) — the second UUID-primary-key
-- entity in this schema, generated in Go (models/household.go's BeforeCreate).
CREATE TABLE IF NOT EXISTS households (
    id TEXT PRIMARY KEY,
    created_at DATETIME, updated_at DATETIME,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    address TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_households_user_id ON households(user_id);

-- household_members is the membership edge (§91.3): household_id + member_vcard_uid
-- (a Contact.vcard_uid, not contacts.id — the graph invariant that every relationship-
-- graph endpoint is referenced by its stable UUID, same convention as
-- relationship_edges.source_id/target_id).
CREATE TABLE IF NOT EXISTS household_members (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME, updated_at DATETIME,
    household_id TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    member_vcard_uid TEXT NOT NULL,
    role TEXT,
    since TEXT,
    until TEXT,
    FOREIGN KEY (household_id) REFERENCES households(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_household_members_household_member
    ON household_members(household_id, member_vcard_uid);
CREATE INDEX IF NOT EXISTS idx_household_members_user_id ON household_members(user_id);
CREATE INDEX IF NOT EXISTS idx_household_members_member_vcard_uid ON household_members(member_vcard_uid);
