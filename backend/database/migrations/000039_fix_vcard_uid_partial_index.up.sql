-- T26: Recreate the contacts vcard_uid unique index to exclude soft-deleted
-- rows. The old index blocked re-importing a contact with the same vcard_uid
-- after it was soft-deleted. With `WHERE deleted_at IS NULL`, a deleted
-- contact's vcard_uid is released for re-use.
DROP INDEX IF EXISTS idx_contacts_vcard_uid_user;
CREATE UNIQUE INDEX idx_contacts_vcard_uid_user
    ON contacts(user_id, vcard_uid)
    WHERE vcard_uid IS NOT NULL AND deleted_at IS NULL;
