package controllers

import (
	"errors"
	"net/http"
	apperrors "perema/errors"
	"perema/logger"
	"perema/models"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateContact(c *gin.Context) {
	// Save to the database
	db := c.MustGet("db").(*gorm.DB)

	// Get validated input from validation middleware
	validated, exists := c.Get("validated")
	if !exists {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("contact", "validation data not found"))
		return
	}

	contactInput, ok := validated.(*models.ContactInput)
	if !ok {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("contact", "invalid validation data type"))
		return
	}

	// Create contact from validated input
	contact := models.Contact{
		Firstname:          contactInput.Firstname,
		Lastname:           contactInput.Lastname,
		Nickname:           contactInput.Nickname,
		Gender:             contactInput.Gender,
		Email:              contactInput.Email,
		Phone:              contactInput.Phone,
		Birthday:           contactInput.Birthday,
		Address:            contactInput.Address,
		HowWeMet:           contactInput.HowWeMet,
		FoodPreference:     contactInput.FoodPreference,
		WorkInformation:    contactInput.WorkInformation,
		ContactInformation: contactInput.ContactInformation,
		Circles:            contactInput.Circles,
	}

	// Save the new contact to the database
	if err := db.Create(&contact).Error; err != nil {
		logger.FromContext(c).Error().Err(err).Msg("Error saving contact to database")
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to save contact").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contact created successfully", "contact": contact})
}

func GetContacts(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	// Get pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "25"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 25
	}
	offset := (page - 1) * limit

	// Define allowed fields and parse requested fields with validation
	allowedFields := []string{"ID", "firstname", "lastname", "nickname", "gender", "email", "phone", "birthday", "address", "how_we_met", "food_preference", "work_information", "contact_information", "circles"}
	var selectedFields []string
	fields := c.Query("fields")
	if fields != "" {
		for _, field := range strings.Split(fields, ",") {
			if slices.Contains(allowedFields, field) { // Validate field
				selectedFields = append(selectedFields, field)
			}
		}
	} else {
		selectedFields = allowedFields // Use all allowed fields if none are specified
	}

	// Parse relationships to include with validation
	var relationshipMap = map[string]bool{
		"notes":         false,
		"activities":    false,
		"relationships": false,
		"reminders":     false,
	}
	includes := c.Query("includes")
	for _, rel := range strings.Split(includes, ",") {
		if _, exists := relationshipMap[rel]; exists {
			relationshipMap[rel] = true
		}
	}

	var contacts []models.Contact
	query := db.Model(&models.Contact{}).Limit(limit).Offset(offset)

	if len(selectedFields) > 0 {
		query = query.Select(selectedFields)
	}

	// Apply search filter using parameterization
	if searchTerm := c.Query("search"); searchTerm != "" {
		searchTermParam := "%" + searchTerm + "%"
		query = query.Where("firstname LIKE ? OR lastname LIKE ? OR nickname LIKE ?", searchTermParam, searchTermParam, searchTermParam)
	}

	if circle := c.Query("circle"); circle != "" {
		query = query.Where("circles LIKE ?", "%"+circle+"%") // Using parameterization
	}

	// Preload requested relationships
	for rel, include := range relationshipMap {
		if include {
			query = query.Preload(rel)
		}
	}

	// Execute query
	if err := query.Find(&contacts).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contacts").WithError(err))
		return
	}

	var total int64
	countQuery := db.Model(&models.Contact{})

	// Apply the same search filters to the count query
	if searchTerm := c.Query("search"); searchTerm != "" {
		searchTermParam := "%" + searchTerm + "%"
		countQuery = countQuery.Where("firstname LIKE ? OR lastname LIKE ? OR nickname LIKE ?", searchTermParam, searchTermParam, searchTermParam)
	}

	if circle := c.Query("circle"); circle != "" {
		countQuery = countQuery.Where("circles LIKE ?", "%"+circle+"%")
	}

	countQuery.Count(&total)

	// Respond with contacts and pagination metadata
	c.JSON(http.StatusOK, gin.H{
		"contacts": contacts,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

func GetContact(c *gin.Context) {
	id := c.Param("id")
	var contact models.Contact
	db := c.MustGet("db").(*gorm.DB)
	if err := db.Preload("Notes").Preload("Activities").Preload("Relationships").Preload("Reminders").First(&contact, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
		}
		return
	}
	c.JSON(http.StatusOK, contact)
}

func UpdateContact(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)

	var contact models.Contact
	if err := db.First(&contact, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
		}
		return
	}

	// Get validated input from validation middleware
	validated, exists := c.Get("validated")
	if !exists {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("contact", "validation data not found"))
		return
	}

	contactInput, ok := validated.(*models.ContactInput)
	if !ok {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("contact", "invalid validation data type"))
		return
	}

	// Updateable fields
	contact.Firstname = contactInput.Firstname
	contact.Lastname = contactInput.Lastname
	contact.Nickname = contactInput.Nickname
	contact.Gender = contactInput.Gender
	contact.Email = contactInput.Email
	contact.Phone = contactInput.Phone
	contact.Birthday = contactInput.Birthday
	contact.Address = contactInput.Address
	contact.HowWeMet = contactInput.HowWeMet
	contact.FoodPreference = contactInput.FoodPreference
	contact.WorkInformation = contactInput.WorkInformation
	contact.ContactInformation = contactInput.ContactInformation
	contact.Circles = contactInput.Circles

	db.Updates(&contact)

	c.JSON(http.StatusOK, contact)
}

func DeleteContact(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)

	// Check if contact exists first
	var contact models.Contact
	if err := db.First(&contact, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
		}
		return
	}

	if err := db.Delete(&contact).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to delete contact").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contact deleted"})
}

// GetCircles returns all unique circles associated with contacts.
func GetCircles(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	var circleNames []string

	// Raw SQL query to extract unique circle names
	err := db.Raw(`SELECT DISTINCT json_each.value AS circle
	               FROM contacts, json_each(contacts.circles)`).Scan(&circleNames).Error
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve circles").WithError(err))
		return
	}

	// Return the list of unique circle names
	c.JSON(http.StatusOK, circleNames)
}
