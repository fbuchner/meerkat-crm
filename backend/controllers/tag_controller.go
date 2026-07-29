package controllers

import (
	"errors"
	apperrors "mycorrhizal/errors"
	"mycorrhizal/middleware"
	"mycorrhizal/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateTag creates a new Tag (tag.go) for the authenticated user.
func CreateTag(c *gin.Context) {
	input, err := middleware.GetValidated[models.TagInput](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	tag := models.Tag{UserID: userID, Name: input.Name}
	if err := db.Create(&tag).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to save tag").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tag created successfully", "tag": tag})
}

// GetTag returns one Tag plus the contacts currently tagged with it.
func GetTag(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var tag models.Tag
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&tag).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Tag").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve tag").WithError(err))
		}
		return
	}

	var contactTags []models.ContactTag
	if err := db.Where("tag_id = ? AND user_id = ?", tag.ID, userID).Find(&contactTags).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve tagged contacts").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"tag": tag, "contacts": contactTags})
}

// ListTags returns the authenticated user's Tags, paginated.
func ListTags(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	pagination := GetPaginationParams(c)

	var tags []models.Tag
	var total int64

	baseQuery := db.Model(&models.Tag{}).Where("user_id = ?", userID)
	if err := baseQuery.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to count tags").WithError(err))
		return
	}

	if err := baseQuery.Session(&gorm.Session{}).
		Order("name").
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Find(&tags).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve tags").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tags":  tags,
		"total": total,
		"page":  pagination.Page,
		"limit": pagination.Limit,
	})
}

// UpdateTag updates a Tag's Name. Tagging lifecycle is not editable here —
// see AddContactTag/RemoveContactTag below.
func UpdateTag(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var tag models.Tag
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&tag).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Tag").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve tag").WithError(err))
		}
		return
	}

	input, err := middleware.GetValidated[models.TagInput](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	tag.Name = input.Name
	if err := db.Save(&tag).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to save tag").WithError(err))
		return
	}

	c.JSON(http.StatusOK, tag)
}

// DeleteTag deletes a Tag. Its ContactTags cascade-delete at the DB level
// (migrations/000031's ON DELETE CASCADE).
func DeleteTag(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var tag models.Tag
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&tag).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Tag").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve tag").WithError(err))
		}
		return
	}

	if err := db.Delete(&tag).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to delete tag").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tag deleted"})
}

// AddContactTag tags a contact. Checks tagging existence before inserting
// (rather than relying on the unique-index error) so "already tagged" is a
// clear ErrAlreadyExists, not a generic database-error string.
func AddContactTag(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var tag models.Tag
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&tag).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Tag").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve tag").WithError(err))
		}
		return
	}

	input, err := middleware.GetValidated[models.ContactTagInput](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	var contact models.Contact
	if err := db.Where("vcard_uid = ? AND user_id = ?", input.ContactVCardUID, userID).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact").WithDetails("vcard_uid", input.ContactVCardUID))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
		}
		return
	}

	var existing models.ContactTag
	lookupErr := db.Where("tag_id = ? AND contact_vcard_uid = ?", tag.ID, input.ContactVCardUID).First(&existing).Error
	if lookupErr == nil {
		apperrors.AbortWithError(c, apperrors.ErrAlreadyExists("Tagging"))
		return
	}
	if !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to check existing tagging").WithError(lookupErr))
		return
	}

	contactTag := models.ContactTag{TagID: tag.ID, UserID: userID, ContactVCardUID: input.ContactVCardUID}
	if err := db.Create(&contactTag).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to add tagging").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contact tagged", "tagging": contactTag})
}

// RemoveContactTag untags a contact.
func RemoveContactTag(c *gin.Context) {
	id := c.Param("id")
	vcardUID := c.Param("vcard_uid")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var tag models.Tag
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&tag).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Tag").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve tag").WithError(err))
		}
		return
	}

	result := db.Where("tag_id = ? AND user_id = ? AND contact_vcard_uid = ?", tag.ID, userID, vcardUID).
		Delete(&models.ContactTag{})
	if result.Error != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to remove tagging").WithError(result.Error))
		return
	}
	if result.RowsAffected == 0 {
		apperrors.AbortWithError(c, apperrors.ErrNotFound("Tagging").WithDetails("vcard_uid", vcardUID))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tagging removed"})
}
