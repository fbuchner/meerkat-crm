package controllers

import (
	"log"
	"net/http"
	"perema/models"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateContact(c *gin.Context) {
	var contact models.Contact
	if err := c.ShouldBindJSON(&contact); err != nil {
		log.Println("Error binding JSON for create contact:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save to the database
	db := c.MustGet("db").(*gorm.DB)

	// Save the new contact to the database
	if err := db.Create(&contact).Error; err != nil {
		log.Println("Error saving to database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save contact"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contact created successfully", "contact": contact})
}

// GetAllContacts handles GET requests to fetch contact records with optional field selection, relationships, and pagination.
//
// Query Parameters:
// - `page` (int, optional): The page number for pagination (default: 1).
// - `limit` (int, optional): The number of records per page (default: 25).
// - `fields` (string, optional): A comma-separated list of fields to include in the response.
//   - Example: "firstname,lastname,email"
//   - If omitted, all fields are included.
//
// - `includes` (string, optional): A comma-separated list of related data to preload.
//   - Example: "notes,activities"
//   - If omitted, no relationships are preloaded.
//
// Response:
//   - JSON object with the following structure:
//     {
//     "contacts": [ /* Array of contact records */ ],
//     "total": <total number of contacts>,
//     "page": <current page number>,
//     "limit": <number of records per page>
//     }
//
// Error Handling:
// - Returns HTTP 500 with an error message if the database query fails.
//
// Example Requests:
// - Fetch all fields: GET /contacts?page=1&limit=10
// - Fetch specific fields: GET /contacts?fields=firstname,lastname,email&page=1&limit=5
// - Fetch relationships: GET /contacts?include=notes,activities
func GetContacts(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	// Get pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "25"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 25
	}
	offset := (page - 1) * limit

	// Parse requested fields
	fields := c.Query("fields") // Example: "firstname,lastname,email"
	var selectedFields []string
	includeAllFields := fields == ""

	if !includeAllFields {
		selectedFields = strings.Split(fields, ",")
	}

	// Parse relationships to include
	includes := c.Query("includes") // Example: "notes,activities"
	includedRelationships := strings.Split(includes, ",")
	relationshipMap := map[string]bool{
		"notes":         false,
		"activities":    false,
		"relationships": false,
		"reminders":     false,
	}

	for _, rel := range includedRelationships {
		if _, exists := relationshipMap[rel]; exists {
			relationshipMap[rel] = true
		}
	}

	// Base query
	var contacts []models.Contact
	query := db.Model(&models.Contact{}).Limit(limit).Offset(offset)

	// Include all fields if none are specified
	if !includeAllFields {
		query = query.Select(strings.Join(selectedFields, ", "))
	}

	// Apply search filter
	if searchTerm := c.Query("search"); searchTerm != "" {
		searchTerm = "%" + searchTerm + "%"
		query = query.Where("firstname LIKE ? OR lastname LIKE ? OR nickname LIKE ?", searchTerm, searchTerm, searchTerm)
	}

	if circle := c.Query("circle"); circle != "" {
		query = query.Where("circles LIKE ?", "%"+circle+"%")
	}

	// Preload requested relationships
	if relationshipMap["notes"] {
		query = query.Preload("Notes")
	}
	if relationshipMap["activities"] {
		query = query.Preload("Activities")
	}
	if relationshipMap["relationships"] {
		query = query.Preload("Relationships")
	}
	if relationshipMap["reminders"] {
		query = query.Preload("Reminders")
	}

	// Execute query
	if err := query.Find(&contacts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve contacts"})
		return
	}

	var total int64
	countQuery := db.Model(&models.Contact{})

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
		c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
		return
	}
	c.JSON(http.StatusOK, contact)
}

func UpdateContact(c *gin.Context) {
	id := c.Param("id")
	var contact models.Contact
	db := c.MustGet("db").(*gorm.DB)
	if err := db.First(&contact, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
		return
	}

	if err := c.ShouldBindJSON(&contact); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db.Save(&contact)
	c.JSON(http.StatusOK, contact)
}

func DeleteContact(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	if err := db.Delete(&models.Contact{}, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve circles"})
		return
	}

	// Return the list of unique circle names
	c.JSON(http.StatusOK, circleNames)
}
