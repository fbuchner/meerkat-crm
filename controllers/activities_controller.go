package controllers

import (
	"log"
	"net/http"
	"perema/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateActivity(c *gin.Context) {
	var activity models.Activity
	if err := c.ShouldBindJSON(&activity); err != nil {
		log.Println("Error binding JSON for create activity:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save to the database
	db := c.MustGet("db").(*gorm.DB)

	// Save the new activity to the database
	if err := db.Create(&activity).Error; err != nil {
		log.Println("Error saving to database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save activity"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Activity created successfully", "activity": activity})
}

func GetActivity(c *gin.Context) {
	id := c.Param("id")
	var activity models.Activity
	db := c.MustGet("db").(*gorm.DB)
	if err := db.First(&activity, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Activity not found"})
		return
	}

	c.JSON(http.StatusOK, activity)
}

func UpdateActivity(c *gin.Context) {
	id := c.Param("id")
	var activity models.Activity
	db := c.MustGet("db").(*gorm.DB)
	if err := db.First(&activity, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Activity not found"})
		return
	}

	if err := c.ShouldBindJSON(&activity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db.Save(&activity)
	c.JSON(http.StatusOK, activity)
}

func DeleteActivity(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	if err := db.Delete(&models.Activity{}, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Activity not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Activity deleted"})
}
