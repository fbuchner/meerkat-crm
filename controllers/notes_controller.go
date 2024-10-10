package controllers

import (
	"log"
	"net/http"
	"perema/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateNote(c *gin.Context) {
	var note models.Note
	if err := c.ShouldBindJSON(&note); err != nil {
		log.Println("Error binding JSON for create note:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save to the database
	db := c.MustGet("db").(*gorm.DB)

	// Save the new note to the database
	if err := db.Create(&note).Error; err != nil {
		log.Println("Error saving to database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save note"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Note created successfully", "note": note})
}

func GetNote(c *gin.Context) {
	id := c.Param("id")
	var note models.Note
	db := c.MustGet("db").(*gorm.DB)
	if err := db.First(&note, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Note not found"})
		return
	}

	c.JSON(http.StatusOK, note)
}

func UpdateNote(c *gin.Context) {
	id := c.Param("id")
	var note models.Note
	db := c.MustGet("db").(*gorm.DB)
	if err := db.First(&note, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Note not found"})
		return
	}

	if err := c.ShouldBindJSON(&note); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db.Save(&note)
	c.JSON(http.StatusOK, note)
}

func DeleteNote(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	if err := db.Delete(&models.Note{}, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Note not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Note deleted"})
}
