package controllers

import (
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"meerkat/logger"
	"meerkat/models"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Find the contact in the database
	if err := db.Where("user_id = ?", userID).First(&contact, contactID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find contact"})
		return
	}

	uploadDir := os.Getenv("PROFILE_PHOTO_DIR")

	// Check if thumbnail is requested via query parameter
	wantsThumbnail := c.Query("thumbnail") == "true"

	// Determine which photo to serve
	var photoFilename string
	if wantsThumbnail {
		photoFilename = contact.PhotoThumbnail
	} else {
		photoFilename = contact.Photo
	}

	if photoFilename == "" {
		filePath := "./static/placeholder-avatar.svg"
		c.File(filePath)
		return
	}

	// Construct the full path to the image
	filePath := filepath.Join(uploadDir, photoFilename)

	logger.FromContext(c).Debug().Str("file_path", filePath).Bool("thumbnail", wantsThumbnail).Msg("Serving profile picture")

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

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Find the contact in the database
	if err := db.Where("user_id = ?", userID).First(&contact, contactID).Error; err != nil {
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
		// Validate file size (10MB limit to prevent DoS)
		const maxFileSize = 10 * 1024 * 1024 // 10MB
		if file.Size > maxFileSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File too large. Maximum size is 10MB"})
			return
		}

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
	fullPhoto := resize.Resize(125, 125, img, resize.Lanczos3)
	fullPhotoPath := filepath.Join(uploadDir, photoPath)
	if err := saveImage(fullPhotoPath, fullPhoto); err != nil {
		return "", "", err
	}

	// Create and save the thumbnail
	thumbnail := resize.Resize(48, 48, img, resize.Lanczos3)
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

// isPrivateIP checks if an IP address is in a private/reserved range
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true
	}

	// Check for loopback
	if ip.IsLoopback() {
		return true
	}

	// Check for link-local (includes cloud metadata endpoint 169.254.169.254)
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Check for private ranges
	if ip.IsPrivate() {
		return true
	}

	// Check for unspecified (0.0.0.0 or ::)
	if ip.IsUnspecified() {
		return true
	}

	return false
}

// validateURLForSSRF checks if a URL is safe to fetch (not pointing to internal resources)
func validateURLForSSRF(rawURL string) (*url.URL, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, errors.New("invalid URL format")
	}

	// Only allow http and https schemes
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, errors.New("only http and https URLs are allowed")
	}

	host := parsedURL.Hostname()
	if host == "" {
		return nil, errors.New("URL must have a host")
	}

	// Block common internal hostnames
	lowerHost := strings.ToLower(host)
	blockedHosts := []string{"localhost", "127.0.0.1", "0.0.0.0", "::1", "[::1]"}
	for _, blocked := range blockedHosts {
		if lowerHost == blocked {
			return nil, errors.New("access to internal hosts is not allowed")
		}
	}

	// Resolve the hostname to IP addresses
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, errors.New("failed to resolve hostname")
	}

	// Check all resolved IPs
	for _, ip := range ips {
		if isPrivateIP(ip) {
			return nil, errors.New("access to internal IP addresses is not allowed")
		}
	}

	return parsedURL, nil
}

// ProxyImage fetches an image from a URL and returns it to the client.
// This is used to work around CORS restrictions when fetching images from external URLs.
func ProxyImage(c *gin.Context) {
	imageURL := c.Query("url")
	if imageURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL parameter is required"})
		return
	}

	// Validate the URL and check for SSRF
	_, err := validateURLForSSRF(imageURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create HTTP client with timeout and disabled redirects to prevent SSRF via redirects
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Validate redirect target
			_, err := validateURLForSSRF(req.URL.String())
			if err != nil {
				return errors.New("redirect to disallowed location")
			}
			if len(via) >= 3 {
				return errors.New("too many redirects")
			}
			return nil
		},
	}

	// Fetch the image
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create request"})
		return
	}

	// Set a user agent to avoid being blocked by some servers
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MeerkatCRM/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		logger.FromContext(c).Warn().Err(err).Str("url", imageURL).Msg("Failed to fetch image from URL")
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch image from URL"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch image: remote server returned " + resp.Status})
		return
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL does not point to an image"})
		return
	}

	// Limit response size (10MB)
	const maxSize = 10 * 1024 * 1024
	limitedReader := io.LimitReader(resp.Body, maxSize+1)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to read image data"})
		return
	}

	if len(body) > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image is too large. Maximum size is 10MB"})
		return
	}

	// Return the image with appropriate content type
	c.Data(http.StatusOK, contentType, body)
}
