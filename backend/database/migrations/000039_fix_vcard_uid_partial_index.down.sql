DROP INDEX IF EXISTS idx_contacts_vcard_uid_user;
CREATE UNIQUE INDEX idx_contacts_vcard_uid_user
    ON contacts(user_id, vcard_uid)
    WHERE vcard_uid IS NOT NULL;
