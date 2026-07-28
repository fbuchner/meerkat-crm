package controllers

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"mycorrhizal/config"
	apperrors "mycorrhizal/errors"
	"mycorrhizal/httputil"
	"mycorrhizal/logger"
	"mycorrhizal/models"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gen2brain/heic"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nfnt/resize"
	"gorm.io/gorm"
)

func GetProfilePicture(c *gin.Context, cfg *config.Config) {
	idParam := c.Param("id")
	contactID, err := strconv.Atoi(idParam)
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrValidation("Invalid contact ID"))
		return
	}
	var contact models.Contact
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Find the contact in the database
	if err := db.Where("user_id = ?", userID).First(&contact, contactID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact"))
			return
		}
		apperrors.AbortWithError(c, apperrors.ErrDatabase("find").WithError(err))
		return
	}

	// Check if thumbnail is requested via query parameter
	wantsThumbnail := c.Query("thumbnail") == "true"

	if wantsThumbnail {
		// Serve thumbnail from database (base64 data URL)
		if contact.PhotoThumbnail == "" {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Thumbnail"))
			return
		}

		// Parse and decode base64 data URL
		// Format: data:image/jpeg;base64,<data>
		if !strings.HasPrefix(contact.PhotoThumbnail, "data:") {
			// Legacy file-based thumbnail - no longer supported
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Thumbnail"))
			return
		}

		parts := strings.SplitN(contact.PhotoThumbnail, ",", 2)
		if len(parts) != 2 {
			apperrors.AbortWithError(c, apperrors.ErrInternal("Invalid thumbnail format"))
			return
		}

		imageData, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to decode thumbnail").WithError(err))
			return
		}

		c.Data(http.StatusOK, "image/jpeg", imageData)
		return
	}

	// Serve full photo from file using validated config path
	if contact.Photo == "" {
		apperrors.AbortWithError(c, apperrors.ErrNotFound("Photo"))
		return
	}

	// Prevent path traversal attacks
	if strings.Contains(contact.Photo, "..") || filepath.IsAbs(contact.Photo) {
		apperrors.AbortWithError(c, apperrors.ErrValidation("Invalid photo path"))
		return
	}

	filePath := filepath.Join(cfg.ProfilePhotoDir, contact.Photo)
	logger.FromContext(c).Debug().Str("file_path", filePath).Msg("Serving profile picture")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		apperrors.AbortWithError(c, apperrors.ErrNotFound("Image"))
		return
	}

	c.File(filePath)
}

func AddPhotoToContact(c *gin.Context, cfg *config.Config) {
	// Check if demo mode is enabled - photo uploads are disabled in demo
	if os.Getenv("DEMO_MODE") == "true" {
		apperrors.AbortWithError(c, apperrors.ErrForbidden("Photo uploads are disabled in demo mode"))
		return
	}

	idParam := c.Param("id")
	contactID, err := strconv.Atoi(idParam)
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrValidation("Invalid contact ID"))
		return
	}
	var contact models.Contact
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Find the contact in the database
	if err := db.Where("user_id = ?", userID).First(&contact, contactID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact"))
			return
		}
		apperrors.AbortWithError(c, apperrors.ErrDatabase("find").WithError(err))
		return
	}

	// Check if there's an uploaded file
	file, err := c.FormFile("photo")
	if err == nil {
		// Validate file size (10MB limit to prevent DoS)
		const maxFileSize = 10 * 1024 * 1024 // 10MB
		if file.Size > maxFileSize {
			apperrors.AbortWithError(c, apperrors.ErrValidation("File too large. Maximum size is 10MB"))
			return
		}

		// Use validated upload directory from config
		if err := os.MkdirAll(cfg.ProfilePhotoDir, os.ModePerm); err != nil {
			apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to create upload directory").WithError(err))
			return
		}
		photoPath, thumbnailPath, err := processAndSavePhoto(file, cfg.ProfilePhotoDir)
		if err != nil {
			apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to process photo").WithError(err))
			return
		}
		oldPhoto := contact.Photo
		contact.Photo = photoPath
		contact.PhotoThumbnail = thumbnailPath

		// Save the updated contact
		if err := db.Save(&contact).Error; err != nil {
			// Clean up newly saved file since DB update failed
			os.Remove(filepath.Join(cfg.ProfilePhotoDir, photoPath))
			apperrors.AbortWithError(c, apperrors.ErrDatabase("update").WithError(err))
			return
		}

		// Delete old photo file after db update
		if oldPhoto != "" {
			os.Remove(filepath.Join(cfg.ProfilePhotoDir, oldPhoto))
		}

		c.JSON(http.StatusOK, contact)
		return
	}

	// Save the updated contact (no new photo uploaded)
	if err := db.Save(&contact).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("update").WithError(err))
		return
	}

	c.JSON(http.StatusOK, contact)
}

// processAndSavePhoto processes an uploaded photo and returns:
// - photoPath: filename of the saved full-size photo
// - thumbnailBase64: base64 data URL of the thumbnail (stored in DB)
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

	// Check for HEIC/HEIF format (http.DetectContentType doesn't recognize it)
	// HEIC files have "ftyp" followed by "heic", "heix", "hevc", "hevx", or "mif1" at byte 4
	if contentType == "application/octet-stream" && len(buf) >= 12 {
		if string(buf[4:8]) == "ftyp" {
			brand := string(buf[8:12])
			if brand == "heic" || brand == "heix" || brand == "hevc" || brand == "hevx" || brand == "mif1" {
				contentType = "image/heic"
			}
		}
	}

	// Validate supported content types
	if contentType != "image/jpeg" && contentType != "image/png" &&
		contentType != "image/heic" && contentType != "image/heif" {
		return "", "", errors.New("unsupported file format")
	}

	// Rewind the file reader
	src.Seek(0, 0)

	// Decode the image (handle JPEG, PNG, and HEIC)
	var img image.Image
	switch contentType {
	case "image/jpeg":
		img, err = jpeg.Decode(src)
	case "image/png":
		img, err = png.Decode(src)
	case "image/heic", "image/heif":
		img, err = heic.Decode(src)
	}
	if err != nil {
		return "", "", err
	}

	// Crop to centered square if rectangular
	img = cropToSquare(img)

	// Generate unique filename for full photo
	baseFilename := uuid.New().String()
	photoPath := baseFilename + "_photo.jpg" // Always save as JPG

	// Create the output directory
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		return "", "", err
	}

	// Save the photo as JPG (max 400x400, only downscale)
	const maxPhotoSize = 400
	bounds := img.Bounds()
	photoImg := img
	if bounds.Dx() > maxPhotoSize || bounds.Dy() > maxPhotoSize {
		photoImg = resize.Resize(maxPhotoSize, maxPhotoSize, img, resize.Lanczos3)
	}
	fullPhotoPath := filepath.Join(uploadDir, photoPath)
	if err := saveImage(fullPhotoPath, photoImg); err != nil {
		return "", "", err
	}

	// Create thumbnail and encode as base64 data URL
	thumbnail := resize.Resize(48, 48, img, resize.Lanczos3)
	var thumbnailBuf bytes.Buffer
	if err := jpeg.Encode(&thumbnailBuf, thumbnail, &jpeg.Options{Quality: 85}); err != nil {
		return "", "", err
	}
	thumbnailBase64 := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(thumbnailBuf.Bytes())

	return photoPath, thumbnailBase64, nil
}

// cropToSquare crops an image to a centered square
// If the image is already square, it returns the original image unchanged
func cropToSquare(img image.Image) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Already square, return as-is
	if width == height {
		return img
	}

	// Calculate the size of the square (use the smaller dimension)
	size := width
	if height < width {
		size = height
	}

	// Calculate crop offset to center the square
	offsetX := (width - size) / 2
	offsetY := (height - size) / 2

	// Create a new RGBA image for the cropped result
	cropped := image.NewRGBA(image.Rect(0, 0, size, size))

	// Copy the centered square region
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			cropped.Set(x, y, img.At(bounds.Min.X+offsetX+x, bounds.Min.Y+offsetY+y))
		}
	}

	return cropped
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

// ProxyImage fetches an image from a URL and returns it to the client.
// This is used to work around CORS restrictions when fetching images from external URLs.
func ProxyImage(c *gin.Context) {
	imageURL := c.Query("url")
	if imageURL == "" {
		apperrors.AbortWithError(c, apperrors.ErrMissingField("url"))
		return
	}

	// Use the shared SSRF-protected fetch function
	body, contentType, err := httputil.FetchImageFromURL(imageURL)
	if err != nil {
		logger.FromContext(c).Warn().Err(err).Str("url", imageURL).Msg("Failed to fetch image from URL")
		apperrors.AbortWithError(c, apperrors.ErrExternal("image proxy", "Failed to fetch image").WithError(err))
		return
	}

	// Reject SVG: fetch.go only checks for an "image/" prefix, which SVG satisfies,
	// but SVG documents can embed <script> and execute JS in this API's origin
	// (which holds the httpOnly auth cookie) once served back to the browser.
	// Every other "image/*" type (png, jpeg, gif, webp, ...) is passive raster/vector
	// data with no script capability, so they're safe to pass through unchanged.
	lowerContentType := strings.ToLower(contentType)
	if strings.HasPrefix(lowerContentType, "image/svg") {
		logger.FromContext(c).Warn().Str("url", imageURL).Str("content_type", contentType).Msg("Rejected SVG image from proxy (XSS risk)")
		apperrors.AbortWithError(c, apperrors.ErrValidation("SVG images are not supported by the image proxy"))
		return
	}

	// Defense-in-depth beyond the global "default-src 'none'" CSP (set by
	// SecurityHeadersMiddleware for every response, including this one): lock
	// down how this specific response can be interpreted if it's ever loaded
	// as a top-level document (e.g. a user pasting the proxy URL directly into
	// the address bar) rather than fetched as an <img> resource.
	//
	// "inline" (not "attachment") because the proxy's whole purpose is to be
	// embedded via <img src="..."> in the frontend; Content-Disposition is only
	// consulted for navigation/download, not for <img> fetches, so this has no
	// effect on normal use. We still set it explicitly (rather than relying on
	// browser default) so a direct navigation renders the image inline instead
	// of prompting a download, matching the endpoint's actual purpose.
	c.Header("Content-Disposition", "inline")

	// Return the image with appropriate content type
	c.Data(http.StatusOK, contentType, body)
}
