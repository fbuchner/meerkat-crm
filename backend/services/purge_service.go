package services

import (
	"mycorrhizal/config"
	"mycorrhizal/logger"
	"mycorrhizal/models"
	"time"

	"gorm.io/gorm"
)

// purgeMinInterval is slightly less than the 24h cron cadence so a natural
// clock-skew or overlap doesn't cause a skipped run.
const purgeMinInterval = 23 * time.Hour

// PurgeSoftDeletedRows hard-deletes soft-deleted rows older than the
// retention window. Called by both the scheduled cron job and the admin
// "purge now" trigger (T26).
//
// Purges in FK-respecting order: children before parents, so ON DELETE
// CASCADE declarations on the children can fire cleanly for the parents.
// Edge/join-shaped rows (activity_contacts, circle_members, contact_tags,
// etc.) that reference contacts are cleaned up explicitly before the
// contacts themselves are purged.
func PurgeSoftDeletedRows(db *gorm.DB, cfg config.Config) {
	cutoff := time.Now().AddDate(0, 0, -cfg.DeleteRetentionDays)

	// activity_contacts has no soft-delete. Clean up rows referencing
	// now-purged contacts as defense-in-depth.
	if err := db.Exec(
		"DELETE FROM activity_contacts WHERE contact_id IN (SELECT id FROM contacts WHERE deleted_at IS NOT NULL AND deleted_at < ?)",
		cutoff,
	).Error; err != nil {
		logger.Error().Err(err).Msg("purge: failed to delete orphaned activity_contacts")
	}

	// Soft-deleted children past retention.
	for _, model := range []any{
		&models.Note{},
		&models.Activity{},
		&models.Reminder{},
		&models.LifeEvent{},
		&models.Preference{},
	} {
		if err := db.Unscoped().Where("deleted_at IS NOT NULL AND deleted_at < ?", cutoff).Delete(model).Error; err != nil {
			logger.Error().Err(err).Msg("purge: failed to delete soft-deleted rows")
		}
	}

	// Edge-shaped rows referencing contacts being purged — defense-in-depth.
	type cleanup struct {
		query string
		args  []interface{}
		desc  string
	}
	cleanups := []cleanup{
		{
			"DELETE FROM circle_members WHERE member_vcard_uid IN (SELECT vcard_uid FROM contacts WHERE deleted_at IS NOT NULL AND deleted_at < ?)",
			[]interface{}{cutoff}, "circle_members",
		},
		{
			"DELETE FROM contact_tags WHERE contact_vcard_uid IN (SELECT vcard_uid FROM contacts WHERE deleted_at IS NOT NULL AND deleted_at < ?)",
			[]interface{}{cutoff}, "contact_tags",
		},
		{
			"DELETE FROM household_members WHERE member_vcard_uid IN (SELECT vcard_uid FROM contacts WHERE deleted_at IS NOT NULL AND deleted_at < ?)",
			[]interface{}{cutoff}, "household_members",
		},
		{
			"DELETE FROM field_values WHERE entity_id IN (SELECT vcard_uid FROM contacts WHERE deleted_at IS NOT NULL AND deleted_at < ?)",
			[]interface{}{cutoff}, "field_values",
		},
		{
			"DELETE FROM preferences WHERE entity_id IN (SELECT vcard_uid FROM contacts WHERE deleted_at IS NOT NULL AND deleted_at < ?)",
			[]interface{}{cutoff}, "preferences",
		},
		{
			"DELETE FROM relationship_edges WHERE source_id IN (SELECT vcard_uid FROM contacts WHERE deleted_at IS NOT NULL AND deleted_at < ?) OR target_id IN (SELECT vcard_uid FROM contacts WHERE deleted_at IS NOT NULL AND deleted_at < ?)",
			[]interface{}{cutoff, cutoff}, "relationship_edges",
		},
		{
			"DELETE FROM contact_sync_links WHERE contact_id IN (SELECT id FROM contacts WHERE deleted_at IS NOT NULL AND deleted_at < ?)",
			[]interface{}{cutoff}, "contact_sync_links",
		},
		{
			"DELETE FROM calendar_event_links WHERE activity_id IN (SELECT id FROM activities WHERE deleted_at IS NOT NULL AND deleted_at < ?)",
			[]interface{}{cutoff}, "calendar_event_links",
		},
	}
	for _, c := range cleanups {
		if err := db.Exec(c.query, c.args...).Error; err != nil {
			logger.Error().Err(err).Str("table", c.desc).Msg("purge: failed to clean up edge rows")
		}
	}

	// Parents: soft-deleted contacts past retention.
	result := db.Unscoped().Where(
		"deleted_at IS NOT NULL AND deleted_at < ?", cutoff,
	).Delete(&models.Contact{})
	if result.Error != nil {
		logger.Error().Err(result.Error).Msg("purge: failed to delete soft-deleted contacts")
	} else if result.RowsAffected > 0 {
		logger.Info().
			Int64("rows", result.RowsAffected).
			Time("cutoff", cutoff).
			Msg("Purged soft-deleted rows")
	}
}

// PurgeDeletedRows is the scheduled cron entry point. It acquires a job lock
// to prevent concurrent purges across multiple instances.
func PurgeDeletedRows(db *gorm.DB, cfg config.Config) {
	acquired, err := acquireJobLock(db, models.JobNamePurgeDeleted, purgeMinInterval)
	if err != nil {
		logger.Error().Err(err).Msg("purge: failed to check job lock")
		return
	}
	if !acquired {
		return
	}
	defer func() {
		if err := releaseJobLock(db, models.JobNamePurgeDeleted, true); err != nil {
			logger.Error().Err(err).Msg("purge: failed to release job lock")
		}
	}()

	PurgeSoftDeletedRows(db, cfg)
}
