package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"mycorrhizal/config"
	apperrors "mycorrhizal/errors"
	"mycorrhizal/logger"
	"mycorrhizal/middleware"
	"mycorrhizal/models"
	"mycorrhizal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TriggerReminders manually triggers the reminder email job (admin only)
func TriggerReminders(c *gin.Context, cfg config.Config) {
	log := logger.FromContext(c)
	db := c.MustGet("db").(*gorm.DB)

	if err := services.SendReminders(db, cfg); err != nil {
		log.Error().Err(err).Msg("Failed to trigger reminder emails")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to send reminder emails").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reminder emails sent successfully"})
}

// TriggerPurge manually triggers the delete-purge job (admin only, T26).
func TriggerPurge(c *gin.Context, cfg config.Config) {
	db := c.MustGet("db").(*gorm.DB)
	services.PurgeSoftDeletedRows(db, cfg)
	c.JSON(http.StatusOK, gin.H{"message": "Purge completed"})
}

// GetCurrentUser returns the current authenticated user's information
func GetCurrentUser(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	db := c.MustGet("db").(*gorm.DB)

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("User"))
			return
		}
		apperrors.AbortWithError(c, apperrors.ErrDatabase("query user").WithError(err))
		return
	}

	c.JSON(http.StatusOK, models.CurrentUserResponse{
		AdminUserResponse: models.AdminUserResponse{
			ID:         user.ID,
			Username:   user.Username,
			Email:      user.Email,
			Language:   user.Language,
			DateFormat: user.DateFormat,
			IsAdmin:    user.IsAdmin,
			CreatedAt:  user.CreatedAt,
			UpdatedAt:  user.UpdatedAt,
		},
		CustomFieldNames:     user.CustomFieldNames,
		EnabledContactFields: user.EnabledContactFields,
	})
}

// ListUsers returns a paginated list of all users (admin only)
func ListUsers(c *gin.Context) {
	log := logger.FromContext(c)
	db := c.MustGet("db").(*gorm.DB)
	pagination := GetPaginationParams(c)

	var total int64
	if err := db.Model(&models.User{}).Count(&total).Error; err != nil {
		log.Error().Err(err).Msg("Failed to count users")
		apperrors.AbortWithError(c, apperrors.ErrDatabase("count users").WithError(err))
		return
	}

	var users []models.User
	if err := db.Order("id ASC").Offset(pagination.Offset).Limit(pagination.Limit).Find(&users).Error; err != nil {
		log.Error().Err(err).Msg("Failed to list users")
		apperrors.AbortWithError(c, apperrors.ErrDatabase("list users").WithError(err))
		return
	}

	userResponses := make([]models.AdminUserResponse, len(users))
	for i, user := range users {
		userResponses[i] = models.AdminUserResponse{
			ID:         user.ID,
			Username:   user.Username,
			Email:      user.Email,
			Language:   user.Language,
			DateFormat: user.DateFormat,
			IsAdmin:    user.IsAdmin,
			CreatedAt:  user.CreatedAt,
			UpdatedAt:  user.UpdatedAt,
		}
	}

	totalPages := int(total) / pagination.Limit
	if int(total)%pagination.Limit > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, models.AdminUsersListResponse{
		Users:      userResponses,
		Total:      total,
		Page:       pagination.Page,
		Limit:      pagination.Limit,
		TotalPages: totalPages,
	})
}

// GetUser returns a single user by ID (admin only)
func GetUser(c *gin.Context) {
	log := logger.FromContext(c)
	db := c.MustGet("db").(*gorm.DB)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("id", "Invalid user ID"))
		return
	}

	var user models.User
	if err := db.First(&user, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("User"))
			return
		}
		log.Error().Err(err).Uint64("user_id", id).Msg("Failed to get user")
		apperrors.AbortWithError(c, apperrors.ErrDatabase("get user").WithError(err))
		return
	}

	c.JSON(http.StatusOK, models.AdminUserResponse{
		ID:         user.ID,
		Username:   user.Username,
		Email:      user.Email,
		Language:   user.Language,
		DateFormat: user.DateFormat,
		IsAdmin:    user.IsAdmin,
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
	})
}

// updates a user's information (admin only)
func UpdateUser(c *gin.Context) {
	log := logger.FromContext(c)
	db := c.MustGet("db").(*gorm.DB)

	currentUserID, ok := currentUserID(c)
	if !ok {
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("id", "Invalid user ID"))
		return
	}

	input, appErr := middleware.GetValidated[models.AdminUserUpdateInput](c)
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	var user models.User
	if err := db.First(&user, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("User"))
			return
		}
		log.Error().Err(err).Uint64("user_id", id).Msg("Failed to get user for update")
		apperrors.AbortWithError(c, apperrors.ErrDatabase("get user").WithError(err))
		return
	}

	// Prevent admin from removing their own admin status
	if input.IsAdmin != nil && !*input.IsAdmin && user.ID == currentUserID {
		apperrors.AbortWithError(c, apperrors.ErrForbidden("Cannot remove your own admin status"))
		return
	}

	// Check if trying to remove the last admin
	if input.IsAdmin != nil && !*input.IsAdmin && user.IsAdmin {
		var adminCount int64
		if err := db.Model(&models.User{}).Where("is_admin = ?", true).Count(&adminCount).Error; err != nil {
			log.Error().Err(err).Msg("Failed to count admins")
			apperrors.AbortWithError(c, apperrors.ErrDatabase("count admins").WithError(err))
			return
		}
		if adminCount <= 1 {
			apperrors.AbortWithError(c, apperrors.ErrForbidden("Cannot remove the last admin"))
			return
		}
	}

	// Apply updates
	if input.Username != nil {
		user.Username = strings.ToLower(*input.Username)
	}
	if input.Email != nil {
		user.Email = strings.ToLower(*input.Email)
	}
	if input.Password != nil {
		hashedPassword, err := services.HashPassword(*input.Password)
		if err != nil {
			if errors.Is(err, services.ErrPasswordTooLong) {
				apperrors.AbortWithError(c, apperrors.ErrValidation(err.Error()))
			} else {
				log.Error().Err(err).Msg("Failed to hash password during admin update")
				apperrors.AbortWithError(c, apperrors.ErrInternal("Could not hash password").WithError(err))
			}
			return
		}
		user.Password = hashedPassword
		// An admin resetting someone's password must end that user's existing
		// sessions, otherwise the reset does not actually lock anyone out.
		user.TokenVersion++
	}
	if input.IsAdmin != nil {
		user.IsAdmin = *input.IsAdmin
	}

	if err := db.Save(&user).Error; err != nil {
		log.Error().Err(err).Uint("user_id", user.ID).Msg("Failed to update user")
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			apperrors.AbortWithError(c, apperrors.ErrAlreadyExists("User").WithDetails("field", "username or email"))
			return
		}
		apperrors.AbortWithError(c, apperrors.ErrDatabase("update user").WithError(err))
		return
	}

	c.JSON(http.StatusOK, models.AdminUserResponse{
		ID:         user.ID,
		Username:   user.Username,
		Email:      user.Email,
		Language:   user.Language,
		DateFormat: user.DateFormat,
		IsAdmin:    user.IsAdmin,
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
	})
}

// DeleteUser deletes a user and all their data (admin only)
func DeleteUser(c *gin.Context) {
	// DeleteUser is the single deliberate call-site exception to the
	// soft-delete rule (T26). users.email/username are UNIQUE, so a
	// soft-deleted account blocks re-registration forever. Here, and
	// only here, we use Unscoped() to genuinely remove the user row
	// and all soft-deleting children. No sync client survives a deleted
	// account, so there is nothing to tombstone for.
	log := logger.FromContext(c)
	db := c.MustGet("db").(*gorm.DB)

	currentUserID, ok := currentUserID(c)
	if !ok {
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("id", "Invalid user ID"))
		return
	}

	// Prevent admin from deleting themselves
	if uint(id) == currentUserID {
		apperrors.AbortWithError(c, apperrors.ErrForbidden("Cannot delete your own account"))
		return
	}

	var user models.User
	if err := db.First(&user, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("User"))
			return
		}
		log.Error().Err(err).Uint64("user_id", id).Msg("Failed to get user for deletion")
		apperrors.AbortWithError(c, apperrors.ErrDatabase("get user").WithError(err))
		return
	}

	// Check if trying to delete the last admin
	if user.IsAdmin {
		var adminCount int64
		if err := db.Model(&models.User{}).Where("is_admin = ?", true).Count(&adminCount).Error; err != nil {
			log.Error().Err(err).Msg("Failed to count admins")
			apperrors.AbortWithError(c, apperrors.ErrDatabase("count admins").WithError(err))
			return
		}
		if adminCount <= 1 {
			apperrors.AbortWithError(c, apperrors.ErrForbidden("Cannot delete the last admin"))
			return
		}
	}

	// Delete user's data in a transaction
	err = db.Transaction(func(tx *gorm.DB) error {
		userID := uint(id)

		// Delete reminders (hard — user account gone, no tombstoning needed)
		if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&models.Reminder{}).Error; err != nil {
			return err
		}

		// Delete notes (hard)
		if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&models.Note{}).Error; err != nil {
			return err
		}

		// Delete activity_contacts associations (many-to-many)
		if err := tx.Exec("DELETE FROM activity_contacts WHERE activity_id IN (SELECT id FROM activities WHERE user_id = ?)", userID).Error; err != nil {
			return err
		}

		// Delete activities (hard)
		if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&models.Activity{}).Error; err != nil {
			return err
		}

		// Delete webhook deliveries, then webhooks (child before parent;
		// WebhookDelivery has no direct UserID, only WebhookID)
		if err := tx.Exec("DELETE FROM webhook_deliveries WHERE webhook_id IN (SELECT id FROM webhooks WHERE user_id = ?)", userID).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.Webhook{}).Error; err != nil {
			return err
		}

		// Delete CardDAV contact sync links, then subscriptions (child before parent)
		if err := tx.Where("user_id = ?", userID).Delete(&models.ContactSyncLink{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.ContactSubscription{}).Error; err != nil {
			return err
		}

		// Delete household memberships, then households (child before parent)
		if err := tx.Where("user_id = ?", userID).Delete(&models.HouseholdMember{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.Household{}).Error; err != nil {
			return err
		}

		// Delete circle memberships, then circles (child before parent)
		if err := tx.Where("user_id = ?", userID).Delete(&models.CircleMember{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.Circle{}).Error; err != nil {
			return err
		}

		// Delete contact tags, then tags (child before parent)
		if err := tx.Where("user_id = ?", userID).Delete(&models.ContactTag{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.Tag{}).Error; err != nil {
			return err
		}

		// Delete custom field values, then field definitions (child before parent)
		if err := tx.Where("user_id = ?", userID).Delete(&models.FieldValue{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.FieldDefinition{}).Error; err != nil {
			return err
		}

		// Delete CardDAV sync token
		if err := tx.Where("user_id = ?", userID).Delete(&models.CardDAVSync{}).Error; err != nil {
			return err
		}

		// Delete API tokens
		if err := tx.Where("user_id = ?", userID).Delete(&models.ApiToken{}).Error; err != nil {
			return err
		}

		// Delete reminder completions
		if err := tx.Where("user_id = ?", userID).Delete(&models.ReminderCompletion{}).Error; err != nil {
			return err
		}

		// Delete calendar event links, then calendar subscriptions (child before parent)
		if err := tx.Where("user_id = ?", userID).Delete(&models.CalendarEventLink{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.CalendarSubscription{}).Error; err != nil {
			return err
		}

		// Delete relationship-graph edges
		if err := tx.Where("user_id = ?", userID).Delete(&models.RelationshipEdge{}).Error; err != nil {
			return err
		}

		// Delete life events (hard)
		if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&models.LifeEvent{}).Error; err != nil {
			return err
		}

		// Delete preferences (hard)
		if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&models.Preference{}).Error; err != nil {
			return err
		}

		// Delete contacts (hard)
		if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&models.Contact{}).Error; err != nil {
			return err
		}

		// Delete user (hard — accounts must be re-registerable, T26)
		if err := tx.Unscoped().Delete(&user).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Error().Err(err).Uint64("user_id", id).Msg("Failed to delete user")
		apperrors.AbortWithError(c, apperrors.ErrDatabase("delete user").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}
