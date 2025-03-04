package controllers

import (
	"net/http"
	"perema/models"
	"perema/services"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func RegisterUser(context *gin.Context) {
	var user models.User
	if err := context.ShouldBindJSON(&user); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	hashedPassword, err := services.HashPassword(user.Password)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Could not hash password"})
		return
	}
	user.Password = hashedPassword

	db := context.MustGet("db").(*gorm.DB)
	if err := db.Create(&user).Error; err != nil {
		context.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
}
func LoginUser(context *gin.Context) {
	var user models.User
	var foundUser models.User

	if err := context.ShouldBindJSON(&user); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	db := context.MustGet("db").(*gorm.DB)
	if err := db.Where("email = ?", user.Email).First(&foundUser).Error; err != nil {
		context.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Compare the hashed password
	if err := bcrypt.CompareHashAndPassword([]byte(foundUser.Password), []byte(user.Password)); err != nil {
		context.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Create JWT token
	tokenString, err := services.GenerateToken(foundUser)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	context.JSON(http.StatusOK, gin.H{"token": tokenString})
}
