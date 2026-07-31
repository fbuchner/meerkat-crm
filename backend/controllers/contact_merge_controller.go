package controllers

import (
	"errors"
	apperrors "mycorrhizal/errors"
	"mycorrhizal/middleware"
	"mycorrhizal/models"
	"mycorrhizal/services"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// loadMergePair validates and loads the keeper/loser contacts for a merge
// request: both must exist and be owned by userID, and the two IDs must
// differ. Shared by preview and commit so the two endpoints reject the same
// malformed requests identically.
func loadMergePair(db *gorm.DB, userID uint, input *models.ContactMergeRequest) (keeper, loser models.Contact, appErr *apperrors.AppError) {
	if input.KeepID == input.MergeID {
		return keeper, loser, apperrors.ErrInvalidInput("merge_id", "must differ from keep_id")
	}
	if err := db.Where("user_id = ?", userID).First(&keeper, input.KeepID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return keeper, loser, apperrors.ErrNotFound("Contact").WithDetails("keep_id", input.KeepID)
		}
		return keeper, loser, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err)
	}
	if err := db.Where("user_id = ?", userID).First(&loser, input.MergeID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return keeper, loser, apperrors.ErrNotFound("Contact").WithDetails("merge_id", input.MergeID)
		}
		return keeper, loser, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err)
	}
	return keeper, loser, nil
}

// missingResolutions returns the field keys from conflicts that have no
// matching entry in resolutions -- used to reject a commit that doesn't
// cover every real conflict, across both the scalar and FieldValue conflict
// sets.
func missingResolutions(conflicts []models.ContactMergeFieldConflict, resolutions map[string]string) []string {
	var missing []string
	for _, conflict := range conflicts {
		if _, ok := resolutions[conflict.Field]; !ok {
			missing = append(missing, conflict.Field)
		}
	}
	return missing
}

// PreviewContactMerge computes and returns the merge outcome for a contact
// pair without writing anything -- the field resolution (union/dedup for
// multi-valued fields, auto-resolved and conflicting scalars), FieldValue
// conflicts, and association counts. Shares its computation functions
// verbatim with CommitContactMerge so the two can never disagree.
func PreviewContactMerge(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	input, err := middleware.GetValidated[models.ContactMergeRequest](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	keeper, loser, appErr := loadMergePair(db, userID, input)
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	resolution := services.ComputeContactMergeResolution(&keeper, &loser)
	fvConflicts, err2 := services.ComputeFieldValueConflicts(db, userID, keeper.VCardUID, loser.VCardUID)
	if err2 != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to compute merge preview").WithError(err2))
		return
	}
	resolution.FieldValueConflicts = fvConflicts

	counts, err2 := services.ComputeContactMergeAssociationCounts(db, userID, loser.ID, loser.VCardUID)
	if err2 != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to compute merge preview").WithError(err2))
		return
	}

	c.JSON(http.StatusOK, models.ContactMergePreviewResponse{
		KeepID:            keeper.ID,
		MergeID:           loser.ID,
		Resolution:        *resolution,
		AssociationCounts: counts,
	})
}

// CommitContactMerge merges loser into keeper: applies the field resolution
// (using the caller's picks for any conflict), re-points every association,
// discards what can't be re-pointed (ContactSyncLink, the loser's photo),
// soft-deletes the loser, and writes an audit note on the keeper.
//
// Conflicts are recomputed fresh from the database here rather than trusting
// anything the client echoes back from a prior preview call -- if either
// contact was edited between preview and commit, this catches it and rejects
// with the specific missing field(s) rather than silently picking for the
// user.
func CommitContactMerge(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	input, err := middleware.GetValidated[models.ContactMergeRequest](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	keeper, loser, appErr := loadMergePair(db, userID, input)
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	resolution := services.ComputeContactMergeResolution(&keeper, &loser)
	fvConflicts, err2 := services.ComputeFieldValueConflicts(db, userID, keeper.VCardUID, loser.VCardUID)
	if err2 != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to merge contacts").WithError(err2))
		return
	}
	resolution.FieldValueConflicts = fvConflicts

	missing := missingResolutions(resolution.Conflicts, input.Resolutions)
	missing = append(missing, missingResolutions(resolution.FieldValueConflicts, input.Resolutions)...)
	if len(missing) > 0 {
		apperrors.AbortWithError(c, apperrors.ErrValidation("Unresolved merge conflicts").WithDetails("fields", missing))
		return
	}

	var noteContent string
	txErr := db.Transaction(func(tx *gorm.DB) error {
		if err := services.ApplyContactMergeResolution(&keeper, resolution, input.Resolutions); err != nil {
			return err // defense in depth; already validated above
		}
		if err := tx.Save(&keeper).Error; err != nil {
			return err
		}

		// Snapshot pre-repoint counts for the audit note inside the same
		// transaction, so they reflect exactly what's about to move (the
		// loser could otherwise have been edited between preview and here).
		counts, err := services.ComputeContactMergeAssociationCounts(tx, userID, loser.ID, loser.VCardUID)
		if err != nil {
			return err
		}

		edgesDropped, err := services.RepointContactAssociations(tx, userID, &keeper, &loser, resolution.FieldValueConflicts, input.Resolutions)
		if err != nil {
			return err
		}

		// ContactSyncLink cleanup (never re-pointed) + a defense-in-depth
		// no-op sweep of anything RepointContactAssociations already moved.
		if err := deleteContactAssociations(tx, loser, userID); err != nil {
			return err
		}

		if err := tx.Delete(&loser).Error; err != nil {
			return err
		}

		noteContent = services.BuildContactMergeNoteContent(&loser, resolution, input.Resolutions, counts, edgesDropped)
		if noteContent != "" {
			note := models.Note{UserID: userID, ContactID: &keeper.ID, Content: noteContent, Date: time.Now()}
			if err := tx.Create(&note).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to merge contacts").WithError(txErr))
		return
	}

	// Photo cleanup outside the transaction, matching DeleteContact -- file
	// deletion can't be rolled back. The loser's photo file is discarded;
	// the keeper's own Photo/PhotoThumbnail was never touched.
	deleteContactPhotos(c, loser)

	go services.TriggerWebhooks(db, currentConfig(c), userID, "contact.updated", keeper)
	go services.TriggerWebhooks(db, currentConfig(c), userID, "contact.deleted", gin.H{"id": loser.ID})

	c.JSON(http.StatusOK, gin.H{
		"message": "Contacts merged",
		"contact": models.NewContactRecordResponse(&keeper, currentConfig(c).ProfilePhotoDir, db),
	})
}
