-- Circle + Tag entities (WP-84, docs/fork-plan/91-envelope-data-model.md §91.5) --
-- a remodel of the flat contacts.circles field into two orthogonal groupings. contacts.circles
-- itself is untouched by this migration (backend-only/additive scope; the data migration into
-- these new tables is WP-84c, a separate, user-assisted step per §91.5).
--
-- id is TEXT (UUID), matching households (000029) -- generated in Go (models/circle.go,
-- models/tag.go's BeforeCreate).
CREATE TABLE IF NOT EXISTS circles (
    id TEXT PRIMARY KEY,
    created_at DATETIME, updated_at DATETIME,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_circles_user_id ON circles(user_id);

-- circle_members is the membership join (§91.5's "members"): circle_id + member_vcard_uid
-- (a contacts.vcard_uid, not contacts.id -- the graph invariant, same convention as
-- household_members.member_vcard_uid).
CREATE TABLE IF NOT EXISTS circle_members (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME, updated_at DATETIME,
    circle_id TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    member_vcard_uid TEXT NOT NULL,
    FOREIGN KEY (circle_id) REFERENCES circles(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_circle_members_circle_member
    ON circle_members(circle_id, member_vcard_uid);
CREATE INDEX IF NOT EXISTS idx_circle_members_user_id ON circle_members(user_id);
CREATE INDEX IF NOT EXISTS idx_circle_members_member_vcard_uid ON circle_members(member_vcard_uid);

-- tags is an attribute of the person (§91.5) -- projects onto Card.Keywords (see
-- models/contact_record.go's projectTags), unlike circles which stay internal-only.
CREATE TABLE IF NOT EXISTS tags (
    id TEXT PRIMARY KEY,
    created_at DATETIME, updated_at DATETIME,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tags_user_id ON tags(user_id);

-- contact_tags is the tagging join (§91.5's "taggings (contact ↔ tag)").
CREATE TABLE IF NOT EXISTS contact_tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME, updated_at DATETIME,
    tag_id TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    contact_vcard_uid TEXT NOT NULL,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_contact_tags_tag_contact
    ON contact_tags(tag_id, contact_vcard_uid);
CREATE INDEX IF NOT EXISTS idx_contact_tags_user_id ON contact_tags(user_id);
CREATE INDEX IF NOT EXISTS idx_contact_tags_contact_vcard_uid ON contact_tags(contact_vcard_uid);
