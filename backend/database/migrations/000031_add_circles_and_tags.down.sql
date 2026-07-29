DROP INDEX IF EXISTS idx_contact_tags_contact_vcard_uid;
DROP INDEX IF EXISTS idx_contact_tags_user_id;
DROP INDEX IF EXISTS idx_contact_tags_tag_contact;
DROP TABLE IF EXISTS contact_tags;

DROP INDEX IF EXISTS idx_tags_user_id;
DROP TABLE IF EXISTS tags;

DROP INDEX IF EXISTS idx_circle_members_member_vcard_uid;
DROP INDEX IF EXISTS idx_circle_members_user_id;
DROP INDEX IF EXISTS idx_circle_members_circle_member;
DROP TABLE IF EXISTS circle_members;

DROP INDEX IF EXISTS idx_circles_user_id;
DROP TABLE IF EXISTS circles;
