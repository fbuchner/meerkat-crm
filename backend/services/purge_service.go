package services

import (
	"mycorrhizal/config"
	"mycorrhizal/logger"
	"mycorrhizal/models"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const purgeMinInterval = 6 * time.Hour

// HardDeleteSoftDeletedContacts hard-deletes soft-deleted rows older than
// the retention window. Called by both the scheduled cron job and the admin
// "purge now" trigger (T26).
//
// Purges in FK-respecting order: children before parents, so ON DELETE
// CASCADE declarations on the children can fire cleanly for the parents.
// Soft-deleting children that a parent's hard-delete would miss are
// deleted explicitly.
func HardDeleteSoftDeletedContacts(db *gorm.DB, cfg config.Config) {
	cutoff := time.Now().AddDate(0, 0, -cfg.DeleteRetentionDays)

	tx := db.Session(&gorm.Session{FullSaveAssociations: false})

	// Children: soft-deleted rows past retention
	if err := tx.Unscoped().Where("deleted_at IS NOT NULL AND deleted_at < ?", cutoff).Delete(&models.Note{}).Error; err != nil {
		logger.Error().Err(err).Msg("purge: failed to delete soft-deleted notes")
	}
	if err := tx.Unscoped().Where("deleted_at IS NOT NULL AND deleted_at < ?", cutoff).Delete(&models.Activity{}).Error; err != nil {
		logger.Error().Err(err).Msg("purge: failed to delete soft-deleted activities")
	}
	if err := tx.Unscoped().Where("deleted_at IS NOT NULL AND deleted_at < ?", cutoff).Delete(&models.Reminder{}).Error; err != nil {
		logger.Error().Err(err).Msg("purge: failed to delete soft-deleted reminders")
	}
	if err := tx.Unscoped().Where("deleted_at IS NOT NULL AND deleted_at < ?", cutoff).Delete(&models.LifeEvent{}).Error; err != nil {
		logger.Error().Err(err).Msg("purge: failed to delete soft-deleted life events")
	}

	// Parents: soft-deleted contacts past retention
	result := tx.Unscoped().Clauses(clause.Returning{}).Where(
		"deleted_at IS NOT NULL AND deleted_at < ?", cutoff,
	).Delete(&models.Contact{})
	if result.Error != nil {
		logger.Error().Err(result.Error).Msg("purge: failed to delete soft-deleted contacts")
	} else {
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

	HardDeleteSoftDeletedContacts(db, cfg)
}
