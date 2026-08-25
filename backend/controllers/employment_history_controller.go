package controllers

import (
	"errors"
	apperrors "meerkat/errors"
	"meerkat/middleware"
	"meerkat/models"
	"meerkat/services"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func findOwnedContact(db *gorm.DB, userID uint, contactID string) (models.Contact, *apperrors.AppError) {
	var contact models.Contact
	if err := db.Where("user_id = ?", userID).First(&contact, contactID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return contact, apperrors.ErrNotFound("Contact").WithDetails("id", contactID)
		}
		return contact, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err)
	}
	return contact, nil
}

// GetEmploymentHistory retrieves all employment history entries for a given
// contact, most recent / open-ended first.
func GetEmploymentHistory(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	contactID := c.Param("id")

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	if _, appErr := findOwnedContact(db, userID, contactID); appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	var entries []models.EmploymentHistory
	if err := db.Where("user_id = ? AND contact_id = ?", userID, contactID).Find(&entries).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve employment history").WithError(err))
		return
	}

	services.SortEmploymentHistory(entries)

	c.JSON(http.StatusOK, gin.H{"employment_history": entries})
}

// validateEmploymentHistoryDates rejects an entry whose EndDate is before its
// StartDate. Either field may be empty (an open-ended or undated entry), in
// which case there is nothing to compare.
func validateEmploymentHistoryDates(startDate, endDate string) *apperrors.AppError {
	if startDate == "" || endDate == "" {
		return nil
	}

	start, startOk := models.PartialDateStart(startDate)
	end, endOk := models.PartialDateStart(endDate)
	if !startOk || !endOk {
		return nil
	}

	if end.Before(start) {
		return apperrors.ErrInvalidInput("end_date", "End date must not be before start date")
	}

	return nil
}

// CreateEmploymentHistory creates a new employment history entry for a given contact.
func CreateEmploymentHistory(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	contactID := c.Param("id")
	contact, appErr := findOwnedContact(db, userID, contactID)
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	input, err := middleware.GetValidated[models.EmploymentHistoryInput](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	if err := validateEmploymentHistoryDates(input.StartDate, input.EndDate); err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	entry := models.EmploymentHistory{
		UserID:       userID,
		ContactID:    contact.ID,
		Organization: input.Organization,
		Department:   input.Department,
		JobTitle:     input.JobTitle,
		Role:         input.Role,
		StartDate:    input.StartDate,
		EndDate:      input.EndDate,
		Notes:        input.Notes,
	}

	txErr := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&entry).Error; err != nil {
			return err
		}
		return services.RecomputeContactEmploymentScalars(tx, userID, contact.ID)
	})
	if txErr != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to save employment history entry").WithError(txErr))
		return
	}

	go services.TriggerWebhooks(db, currentConfig(c), userID, "employment_history.created", entry)
	c.JSON(http.StatusCreated, gin.H{"employment_history": entry})
}

func findOwnedEntry(db *gorm.DB, userID uint, contactID, entryID string) (models.EmploymentHistory, *apperrors.AppError) {
	var entry models.EmploymentHistory
	if err := db.Where("user_id = ? AND contact_id = ?", userID, contactID).First(&entry, entryID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entry, apperrors.ErrNotFound("Employment history entry").WithDetails("id", entryID)
		}
		return entry, apperrors.ErrDatabase("Failed to retrieve employment history entry").WithError(err)
	}
	return entry, nil
}

// UpdateEmploymentHistory updates an existing employment history entry.
func UpdateEmploymentHistory(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	entry, appErr := findOwnedEntry(db, userID, c.Param("id"), c.Param("eid"))
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	input, err := middleware.GetValidated[models.EmploymentHistoryInput](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	if err := validateEmploymentHistoryDates(input.StartDate, input.EndDate); err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	entry.Organization = input.Organization
	entry.Department = input.Department
	entry.JobTitle = input.JobTitle
	entry.Role = input.Role
	entry.StartDate = input.StartDate
	entry.EndDate = input.EndDate
	entry.Notes = input.Notes

	txErr := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&entry).Error; err != nil {
			return err
		}
		return services.RecomputeContactEmploymentScalars(tx, userID, entry.ContactID)
	})
	if txErr != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to update employment history entry").WithError(txErr))
		return
	}

	go services.TriggerWebhooks(db, currentConfig(c), userID, "employment_history.updated", entry)
	c.JSON(http.StatusOK, gin.H{"employment_history": entry})
}

// DeleteEmploymentHistory deletes an employment history entry.
func DeleteEmploymentHistory(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	entry, appErr := findOwnedEntry(db, userID, c.Param("id"), c.Param("eid"))
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	contactID := entry.ContactID

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&entry).Error; err != nil {
			return err
		}
		return services.RecomputeContactEmploymentScalars(tx, userID, contactID)
	})
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to delete employment history entry").WithError(err))
		return
	}

	go services.TriggerWebhooks(db, currentConfig(c), userID, "employment_history.deleted", gin.H{"id": entry.ID})
	c.JSON(http.StatusOK, gin.H{"message": "Employment history entry deleted"})
}

// EndEmploymentHistory is a convenience action that closes out a still-open
// entry by setting its EndDate to today.
func EndEmploymentHistory(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	entry, appErr := findOwnedEntry(db, userID, c.Param("id"), c.Param("eid"))
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	if entry.EndDate != "" {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("end_date", "Entry is already closed"))
		return
	}

	entry.EndDate = time.Now().UTC().Format("2006-01-02")

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&entry).Error; err != nil {
			return err
		}
		return services.RecomputeContactEmploymentScalars(tx, userID, entry.ContactID)
	})
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to update employment history entry").WithError(err))
		return
	}

	go services.TriggerWebhooks(db, currentConfig(c), userID, "employment_history.updated", entry)
	c.JSON(http.StatusOK, gin.H{"employment_history": entry})
}
