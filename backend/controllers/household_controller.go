package controllers

import (
	"errors"
	apperrors "mycorrhizal/errors"
	"mycorrhizal/middleware"
	"mycorrhizal/models"
	"mycorrhizal/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateHousehold creates a new Household for the authenticated user.
func CreateHousehold(c *gin.Context) {
	input, err := middleware.GetValidated[models.HouseholdInput](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	household := models.Household{UserID: userID, Name: input.Name, Type: input.Type, Address: input.Address}
	if err := db.Create(&household).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to save household").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Household created successfully", "household": household})
}

// GetHousehold returns one Household plus its current membership.
func GetHousehold(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var household models.Household
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&household).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Household").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve household").WithError(err))
		}
		return
	}

	var members []models.HouseholdMember
	if err := db.Where("household_id = ? AND user_id = ?", household.ID, userID).Find(&members).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve household members").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"household": household, "members": members})
}

// ListHouseholds returns the authenticated user's Households, paginated.
// When ?include_members=true, each household carries its current member
// rows, so the caller can resolve household membership in one request
// instead of N+1.
func ListHouseholds(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	pagination := GetPaginationParams(c)

	var households []models.Household
	var total int64

	baseQuery := db.Model(&models.Household{}).Where("user_id = ?", userID)
	if err := baseQuery.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to count households").WithError(err))
		return
	}

	if err := baseQuery.Session(&gorm.Session{}).
		Order("name").
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Find(&households).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve households").WithError(err))
		return
	}

	result := gin.H{
		"households": households,
		"total":      total,
		"page":       pagination.Page,
		"limit":      pagination.Limit,
	}

	if c.Query("include_members") == "true" {
		householdIDs := make([]string, len(households))
		for i, household := range households {
			householdIDs[i] = household.ID
		}
		var members []models.HouseholdMember
		if err := db.Where("household_id IN ? AND user_id = ?", householdIDs, userID).Find(&members).Error; err != nil {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve household members").WithError(err))
			return
		}
		result["members"] = members
	}

	c.JSON(http.StatusOK, result)
}

// UpdateHousehold updates a Household's Name and Type. Membership lifecycle
// is not editable here — see AddHouseholdMember/RemoveHouseholdMember below.
func UpdateHousehold(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var household models.Household
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&household).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Household").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve household").WithError(err))
		}
		return
	}

	input, err := middleware.GetValidated[models.HouseholdInput](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	household.Name = input.Name
	household.Type = input.Type
	household.Address = input.Address
	if err := db.Save(&household).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to save household").WithError(err))
		return
	}

	c.JSON(http.StatusOK, household)
}

// DeleteHousehold deletes a Household. Its HouseholdMembers cascade-delete
// at the DB level (migrations/000029's ON DELETE CASCADE).
func DeleteHousehold(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var household models.Household
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&household).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Household").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve household").WithError(err))
		}
		return
	}

	if err := db.Delete(&household).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to delete household").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Household deleted"})
}

// AddHouseholdMember adds a contact to a Household with an optional role.
// Checks membership existence before inserting (rather than relying on the
// unique-index error) so the "already a member" case is a clear
// ErrAlreadyExists, not a generic database-error string — same idiom as
// AddCircleMember.
func AddHouseholdMember(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var household models.Household
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&household).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Household").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve household").WithError(err))
		}
		return
	}

	input, err := middleware.GetValidated[models.HouseholdMemberInput](c)
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

	var existing models.HouseholdMember
	lookupErr := db.Where("household_id = ? AND member_vcard_uid = ?", household.ID, input.MemberVCardUID).First(&existing).Error
	if lookupErr == nil {
		apperrors.AbortWithError(c, apperrors.ErrAlreadyExists("Household membership"))
		return
	}
	if !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to check existing household membership").WithError(lookupErr))
		return
	}

	member := models.HouseholdMember{
		HouseholdID:    household.ID,
		UserID:         userID,
		MemberVCardUID: input.MemberVCardUID,
		Role:           input.Role,
		Since:          input.Since,
		Until:          input.Until,
	}
	if err := db.Create(&member).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to add household member").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member added", "member": member})
}

// RemoveHouseholdMember removes a contact from a Household.
func RemoveHouseholdMember(c *gin.Context) {
	id := c.Param("id")
	vcardUID := c.Param("vcard_uid")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var household models.Household
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&household).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Household").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve household").WithError(err))
		}
		return
	}

	result := db.Where("household_id = ? AND user_id = ? AND member_vcard_uid = ?", household.ID, userID, vcardUID).
		Delete(&models.HouseholdMember{})
	if result.Error != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to remove household member").WithError(result.Error))
		return
	}
	if result.RowsAffected == 0 {
		apperrors.AbortWithError(c, apperrors.ErrNotFound("Household membership").WithDetails("vcard_uid", vcardUID))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed"})
}

// UpdateHouseholdMember updates a member's role in-place without the
// remove-then-re-add dance (T1 review: B3+B4). Only Role is editable.
func UpdateHouseholdMember(c *gin.Context) {
	id := c.Param("id")
	memberVCardUID := c.Param("vcard_uid")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Verify household ownership.
	var household models.Household
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&household).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Household").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve household").WithError(err))
		}
		return
	}

	type memberUpdate struct {
		Role string `json:"role"`
	}
	var input memberUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		apperrors.AbortWithError(c, apperrors.ErrValidation("Invalid request body"))
		return
	}

	result := db.Model(&models.HouseholdMember{}).
		Where("household_id = ? AND member_vcard_uid = ? AND user_id = ?", id, memberVCardUID, userID).
		Update("role", input.Role)

	if result.Error != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to update member role").WithError(result.Error))
		return
	}
	if result.RowsAffected == 0 {
		apperrors.AbortWithError(c, apperrors.ErrNotFound("Member"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member updated"})
}

// SuggestHouseholdRelationships is the wiring that finally connects the
// suggestion engine (services.GenerateHouseholdSuggestions) to a running
// app — the review surface in RelationshipEdgeList.tsx has been provably
// dead since it shipped, because nothing could ever produce a suggested
// edge. This endpoint is the missing trigger.
//
// GenerateHouseholdSuggestions re-scans the household's CURRENT membership
// and (its own doc comment) idempotently persists a suggested edge for every
// applicable pair, returning only the edges it newly created. Re-running this
// endpoint therefore never duplicates: the engine checks both storage
// directions and any existing status before creating. The household is loaded
// user-scoped here, so the engine can never be pointed at another user's
// household or made to fabricate edges under someone else's UserID.
func SuggestHouseholdRelationships(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var household models.Household
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&household).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Household").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve household").WithError(err))
		}
		return
	}

	created, err := services.GenerateHouseholdSuggestions(db, household)
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to generate household relationship suggestions").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Relationship suggestions generated",
		"household_id":    household.ID,
		"suggested_edges": created,
		"total":           len(created),
	})
}
