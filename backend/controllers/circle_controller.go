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

// CreateCircle creates a new Circle (circle.go) for the authenticated user.
func CreateCircle(c *gin.Context) {
	input, err := middleware.GetValidated[models.CircleInput](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	circle := models.Circle{UserID: userID, Name: input.Name}
	if err := db.Create(&circle).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to save circle").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Circle created successfully", "circle": circle})
}

// GetCircle returns one Circle plus its current membership.
func GetCircle(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var circle models.Circle
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&circle).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Circle").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve circle").WithError(err))
		}
		return
	}

	var members []models.CircleMember
	if err := db.Where("circle_id = ? AND user_id = ?", circle.ID, userID).Find(&members).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve circle members").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"circle": circle, "members": members})
}

// ListCircles returns the authenticated user's Circles, paginated. Named
// ListCircles (not GetCircles) because contact_controller.go already owns
// that name for the legacy flat-Contact.Circles distinct-values endpoint.
//
// When ?include_members=true, each circle carries its current member
// VCardUIDs, so the caller can resolve contact-circle relationships in one
// request instead of N+1.
func ListCircles(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	pagination := GetPaginationParams(c)

	var circles []models.Circle
	var total int64

	baseQuery := db.Model(&models.Circle{}).Where("user_id = ?", userID)
	if err := baseQuery.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to count circles").WithError(err))
		return
	}

	if err := baseQuery.Session(&gorm.Session{}).
		Order("name").
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Find(&circles).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve circles").WithError(err))
		return
	}

	result := gin.H{
		"circles": circles,
		"total":   total,
		"page":    pagination.Page,
		"limit":   pagination.Limit,
	}

	if c.Query("include_members") == "true" {
		circleIDs := make([]string, len(circles))
		for i, circle := range circles {
			circleIDs[i] = circle.ID
		}
		var members []models.CircleMember
		if err := db.Where("circle_id IN ? AND user_id = ?", circleIDs, userID).Find(&members).Error; err != nil {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve circle members").WithError(err))
			return
		}
		result["members"] = members
	}

	c.JSON(http.StatusOK, result)
}

// UpdateCircle updates a Circle's Name. Membership lifecycle is not editable
// here — see AddCircleMember/RemoveCircleMember below.
func UpdateCircle(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var circle models.Circle
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&circle).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Circle").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve circle").WithError(err))
		}
		return
	}

	input, err := middleware.GetValidated[models.CircleInput](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	circle.Name = input.Name
	if err := db.Save(&circle).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to save circle").WithError(err))
		return
	}

	c.JSON(http.StatusOK, circle)
}

// DeleteCircle deletes a Circle. Its CircleMembers cascade-delete at the DB
// level (migrations/000031's ON DELETE CASCADE).
func DeleteCircle(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var circle models.Circle
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&circle).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Circle").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve circle").WithError(err))
		}
		return
	}

	if err := db.Delete(&circle).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to delete circle").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Circle deleted"})
}

// AddCircleMember adds a contact to a Circle. Checks membership existence
// before inserting (rather than relying on the unique-index error) so the
// "already a member" case is a clear ErrAlreadyExists, not a generic
// database-error string.
func AddCircleMember(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var circle models.Circle
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&circle).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Circle").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve circle").WithError(err))
		}
		return
	}

	input, err := middleware.GetValidated[models.CircleMemberInput](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	var contact models.Contact
	if err := db.Where("vcard_uid = ? AND user_id = ?", input.MemberVCardUID, userID).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact").WithDetails("vcard_uid", input.MemberVCardUID))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
		}
		return
	}

	var existing models.CircleMember
	lookupErr := db.Where("circle_id = ? AND member_vcard_uid = ?", circle.ID, input.MemberVCardUID).First(&existing).Error
	if lookupErr == nil {
		apperrors.AbortWithError(c, apperrors.ErrAlreadyExists("Circle membership"))
		return
	}
	if !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to check existing circle membership").WithError(lookupErr))
		return
	}

	member := models.CircleMember{CircleID: circle.ID, UserID: userID, MemberVCardUID: input.MemberVCardUID}
	if err := db.Create(&member).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to add circle member").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member added", "member": member})
}

// RemoveCircleMember removes a contact from a Circle.
func RemoveCircleMember(c *gin.Context) {
	id := c.Param("id")
	vcardUID := c.Param("vcard_uid")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var circle models.Circle
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&circle).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Circle").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve circle").WithError(err))
		}
		return
	}

	result := db.Where("circle_id = ? AND user_id = ? AND member_vcard_uid = ?", circle.ID, userID, vcardUID).
		Delete(&models.CircleMember{})
	if result.Error != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to remove circle member").WithError(result.Error))
		return
	}
	if result.RowsAffected == 0 {
		apperrors.AbortWithError(c, apperrors.ErrNotFound("Circle membership").WithDetails("vcard_uid", vcardUID))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed"})
}
