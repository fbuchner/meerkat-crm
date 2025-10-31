package controllers

import (
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"perema/logger"
	"perema/models"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nfnt/resize"
	"gorm.io/gorm"
)

func GetProfilePicture(c *gin.Context) {
	idParam := c.Param("id")
	contactID, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contact ID"})
		return
	}
	var contact models.Contact
	db := c.MustGet("db").(*gorm.DB)

	// Find the contact in the database
	if err := db.First(&contact, contactID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find contact"})
		return
	}

	uploadDir := os.Getenv("PROFILE_PHOTO_DIR")

	if contact.Photo == "" {
		filePath := "./static/placeholder-avatar.png"
		c.File(filePath)
		return
	}

	// Construct the full path to the image
	filePath := filepath.Join(uploadDir, contact.Photo)

	logger.FromContext(c).Debug().Str("file_path", filePath).Msg("Serving profile picture")

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	// Serve the image
	c.File(filePath)
}

func AddPhotoToContact(c *gin.Context) {
	idParam := c.Param("id")
	contactID, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contact ID"})
		return
	}
	var contact models.Contact
	db := c.MustGet("db").(*gorm.DB)

	// Find the contact in the database
	if err := db.First(&contact, contactID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find contact"})
		return
	}

	// Check if there's an uploaded file
	file, err := c.FormFile("photo")
	if err == nil {
		// Handle the file upload
		uploadDir := os.Getenv("PROFILE_PHOTO_DIR")
		if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
			return
		}
		photoPath, thumbnailPath, err := processAndSavePhoto(file, uploadDir)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process photo"})
			return
		}
		contact.Photo = photoPath
		contact.PhotoThumbnail = thumbnailPath
	}

	// Save the updated contact
	if err := db.Save(&contact).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update contact"})
		return
	}

	c.JSON(http.StatusOK, contact)
}

func processAndSavePhoto(file *multipart.FileHeader, uploadDir string) (string, string, error) {
	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		return "", "", err
	}
	defer src.Close()

	// Detect the image format
	buf := make([]byte, 512)
	_, err = src.Read(buf)
	if err != nil {
		return "", "", err
	}
	contentType := http.DetectContentType(buf)

	// Validate supported content types
	if contentType != "image/jpeg" && contentType != "image/png" {
		return "", "", errors.New("unsupported file format")
	}

	// Rewind the file reader
	src.Seek(0, 0)

	// Decode the image (handle both JPEG and PNG)
	var img image.Image
	switch contentType {
	case "image/jpeg":
		img, err = jpeg.Decode(src)
	case "image/png":
		img, err = png.Decode(src)
	}
	if err != nil {
		return "", "", err
	}

	// Generate unique filenames
	baseFilename := uuid.New().String()
	photoPath := baseFilename + "_photo.jpg" // Always save as JPG
	thumbnailPath := baseFilename + "_thumbnail.jpg"

	// Create the output directory
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		return "", "", err
	}

	// Save the original photo as JPG
	fullPhotoPath := filepath.Join(uploadDir, photoPath)
	if err := saveImage(fullPhotoPath, img); err != nil {
		return "", "", err
	}

	// Create and save the thumbnail
	thumbnail := resize.Resize(100, 100, img, resize.Lanczos3)
	fullThumbnailPath := filepath.Join(uploadDir, thumbnailPath)
	if err := saveImage(fullThumbnailPath, thumbnail); err != nil {
		return "", "", err
	}

	return photoPath, thumbnailPath, nil
}

func saveImage(path string, img image.Image) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	// Always encode as JPEG
	return jpeg.Encode(out, img, &jpeg.Options{Quality: 85})
}
