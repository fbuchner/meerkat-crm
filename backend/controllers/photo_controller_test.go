package controllers

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"mycorrhizal/config"
	"mycorrhizal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- fixtures ---

func newPNGBytes(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 3 % 256), G: uint8(y * 5 % 256), B: 100, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func newJPEGBytes(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, nil))
	return buf.Bytes()
}

// newTestFileHeader builds a real *multipart.FileHeader by round-tripping
// through an actual multipart-encoded HTTP request body, which is what both
// AddPhotoToContact (via the router) and processAndSavePhoto (called
// directly here) actually receive in production.
func newTestFileHeader(t *testing.T, fieldName, filename string, data []byte) *multipart.FileHeader {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fieldName, filename)
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	require.NoError(t, req.ParseMultipartForm(32<<20))
	return req.MultipartForm.File[fieldName][0]
}

// --- processAndSavePhoto ---

func TestProcessAndSavePhoto_PNGSuccess(t *testing.T) {
	dir := t.TempDir()
	fh := newTestFileHeader(t, "photo", "photo.png", newPNGBytes(t, 100, 100))

	photoPath, thumbnail, err := processAndSavePhoto(fh, dir)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(photoPath, "_photo.jpg"))
	assert.True(t, strings.HasPrefix(thumbnail, "data:image/jpeg;base64,"))

	fullPath := filepath.Join(dir, photoPath)
	data, err := os.ReadFile(fullPath)
	require.NoError(t, err)
	decodedImg, err := jpeg.Decode(bytes.NewReader(data))
	require.NoError(t, err, "saved photo must be a valid JPEG")
	assert.Equal(t, 100, decodedImg.Bounds().Dx())
	assert.Equal(t, 100, decodedImg.Bounds().Dy())

	thumbData, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(thumbnail, "data:image/jpeg;base64,"))
	require.NoError(t, err)
	thumbImg, err := jpeg.Decode(bytes.NewReader(thumbData))
	require.NoError(t, err, "thumbnail must be a valid JPEG")
	assert.Equal(t, 48, thumbImg.Bounds().Dx())
	assert.Equal(t, 48, thumbImg.Bounds().Dy())
}

func TestProcessAndSavePhoto_JPEGSuccess(t *testing.T) {
	dir := t.TempDir()
	fh := newTestFileHeader(t, "photo", "photo.jpg", newJPEGBytes(t, 60, 60))

	photoPath, thumbnail, err := processAndSavePhoto(fh, dir)
	require.NoError(t, err)
	assert.NotEmpty(t, photoPath)
	assert.NotEmpty(t, thumbnail)

	_, err = os.Stat(filepath.Join(dir, photoPath))
	require.NoError(t, err, "photo file must be written to disk")
}

func TestProcessAndSavePhoto_RectangularImageIsCroppedToSquare(t *testing.T) {
	dir := t.TempDir()
	fh := newTestFileHeader(t, "photo", "wide.png", newPNGBytes(t, 40, 20))

	photoPath, _, err := processAndSavePhoto(fh, dir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, photoPath))
	require.NoError(t, err)
	decodedImg, err := jpeg.Decode(bytes.NewReader(data))
	require.NoError(t, err)
	// The smaller dimension (20) determines the cropped square's size.
	assert.Equal(t, 20, decodedImg.Bounds().Dx())
	assert.Equal(t, 20, decodedImg.Bounds().Dy())
}

func TestProcessAndSavePhoto_LargeImageIsDownscaledTo400(t *testing.T) {
	dir := t.TempDir()
	fh := newTestFileHeader(t, "photo", "big.png", newPNGBytes(t, 500, 500))

	photoPath, _, err := processAndSavePhoto(fh, dir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, photoPath))
	require.NoError(t, err)
	decodedImg, err := jpeg.Decode(bytes.NewReader(data))
	require.NoError(t, err)
	assert.Equal(t, 400, decodedImg.Bounds().Dx())
	assert.Equal(t, 400, decodedImg.Bounds().Dy())
}

func TestProcessAndSavePhoto_UnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	fh := newTestFileHeader(t, "photo", "notes.txt", []byte("this is plain text, not an image, padded out past the sniff window so DetectContentType has plenty to look at and still correctly calls it text/plain; charset=utf-8"))

	_, _, err := processAndSavePhoto(fh, dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported file format")
}

func TestProcessAndSavePhoto_EmptyFileReadError(t *testing.T) {
	dir := t.TempDir()
	fh := newTestFileHeader(t, "photo", "empty.png", []byte{})

	_, _, err := processAndSavePhoto(fh, dir)
	require.Error(t, err)
}

func TestProcessAndSavePhoto_CorruptPNGDecodeError(t *testing.T) {
	dir := t.TempDir()
	// Valid PNG signature (so DetectContentType sniffs "image/png") followed
	// by garbage instead of real chunks, so png.Decode itself must fail.
	corrupt := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, bytes.Repeat([]byte{0xFF}, 64)...)
	fh := newTestFileHeader(t, "photo", "corrupt.png", corrupt)

	_, _, err := processAndSavePhoto(fh, dir)
	require.Error(t, err)
}

func TestProcessAndSavePhoto_HEICMagicBytesDetected(t *testing.T) {
	dir := t.TempDir()
	// http.DetectContentType doesn't recognize HEIC natively; the controller
	// sniffs the ftyp/heic brand itself (photo_controller.go lines ~213-220).
	// Real HEIC encoding isn't practical to fabricate in a unit test, so this
	// proves the sniff-and-route-to-heic.Decode branch is reached (and fails
	// cleanly on non-decodable content) rather than falling through to
	// "unsupported file format".
	heicBytes := make([]byte, 32)
	copy(heicBytes[4:8], []byte("ftyp"))
	copy(heicBytes[8:12], []byte("heic"))
	fh := newTestFileHeader(t, "photo", "photo.heic", heicBytes)

	_, _, err := processAndSavePhoto(fh, dir)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "unsupported file format", "HEIC magic bytes must route to heic.Decode, not the unsupported-format branch")
}

// --- cropToSquare ---

func TestCropToSquare_AlreadySquareIsUnchanged(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 30, 30))
	result := cropToSquare(img)
	assert.Same(t, img, result)
}

func TestCropToSquare_WideImageCropsToHeight(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 40, 20))
	// Mark distinguishing pixels so we can verify the crop is actually
	// centered, not just correctly sized.
	img.Set(10, 5, color.RGBA{255, 0, 0, 255}) // inside the centered crop (offsetX=10)
	img.Set(0, 5, color.RGBA{0, 255, 0, 255})  // outside the centered crop

	result := cropToSquare(img)
	bounds := result.Bounds()
	assert.Equal(t, 20, bounds.Dx())
	assert.Equal(t, 20, bounds.Dy())

	r, g, _, _ := result.At(0, 5).RGBA()
	_ = g
	// Pixel at local (0,5) in the cropped image maps to source (10,5), which
	// was set to red.
	assert.NotZero(t, r)
}

func TestCropToSquare_TallImageCropsToWidth(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 20, 50))
	result := cropToSquare(img)
	bounds := result.Bounds()
	assert.Equal(t, 20, bounds.Dx())
	assert.Equal(t, 20, bounds.Dy())
}

// --- saveImage ---

func TestSaveImage_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.jpg")
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))

	require.NoError(t, saveImage(path, img))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	_, err = jpeg.Decode(bytes.NewReader(data))
	require.NoError(t, err)
}

func TestSaveImage_ErrorsWhenParentDirMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent-subdir", "out.jpg")
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))

	err := saveImage(path, img)
	assert.Error(t, err)
}

// --- GetProfilePicture ---

func TestGetProfilePicture_InvalidContactID(t *testing.T) {
	cfg := &config.Config{ProfilePhotoDir: t.TempDir()}
	_, router := setupRouter()
	router.GET("/contacts/:id/photo", func(c *gin.Context) { GetProfilePicture(c, cfg) })

	req, _ := http.NewRequest("GET", "/contacts/not-a-number/photo", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetProfilePicture_ContactNotFound(t *testing.T) {
	cfg := &config.Config{ProfilePhotoDir: t.TempDir()}
	_, router := setupRouter()
	router.GET("/contacts/:id/photo", func(c *gin.Context) { GetProfilePicture(c, cfg) })

	req, _ := http.NewRequest("GET", "/contacts/999/photo", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetProfilePicture_ThumbnailNotSet(t *testing.T) {
	cfg := &config.Config{ProfilePhotoDir: t.TempDir()}
	db, router := setupRouter()
	router.GET("/contacts/:id/photo", func(c *gin.Context) { GetProfilePicture(c, cfg) })

	var user models.User
	require.NoError(t, db.First(&user).Error)
	contact := models.Contact{UserID: user.ID, Firstname: "No", Lastname: "Thumb"}
	require.NoError(t, db.Create(&contact).Error)

	req, _ := http.NewRequest("GET", "/contacts/"+strconv.Itoa(int(contact.ID))+"/photo?thumbnail=true", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetProfilePicture_LegacyFileBasedThumbnailUnsupported(t *testing.T) {
	cfg := &config.Config{ProfilePhotoDir: t.TempDir()}
	db, router := setupRouter()
	router.GET("/contacts/:id/photo", func(c *gin.Context) { GetProfilePicture(c, cfg) })

	var user models.User
	require.NoError(t, db.First(&user).Error)
	contact := models.Contact{UserID: user.ID, Firstname: "Legacy", Lastname: "Thumb", PhotoThumbnail: "some-legacy-filename.jpg"}
	require.NoError(t, db.Create(&contact).Error)

	req, _ := http.NewRequest("GET", "/contacts/"+strconv.Itoa(int(contact.ID))+"/photo?thumbnail=true", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetProfilePicture_MalformedThumbnailDataURL(t *testing.T) {
	cfg := &config.Config{ProfilePhotoDir: t.TempDir()}
	db, router := setupRouter()
	router.GET("/contacts/:id/photo", func(c *gin.Context) { GetProfilePicture(c, cfg) })

	var user models.User
	require.NoError(t, db.First(&user).Error)
	// Has the "data:" prefix but no comma separating header from payload.
	contact := models.Contact{UserID: user.ID, Firstname: "Bad", Lastname: "URL", PhotoThumbnail: "data:image/jpeg;base64"}
	require.NoError(t, db.Create(&contact).Error)

	req, _ := http.NewRequest("GET", "/contacts/"+strconv.Itoa(int(contact.ID))+"/photo?thumbnail=true", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetProfilePicture_InvalidBase64Thumbnail(t *testing.T) {
	cfg := &config.Config{ProfilePhotoDir: t.TempDir()}
	db, router := setupRouter()
	router.GET("/contacts/:id/photo", func(c *gin.Context) { GetProfilePicture(c, cfg) })

	var user models.User
	require.NoError(t, db.First(&user).Error)
	contact := models.Contact{UserID: user.ID, Firstname: "Bad", Lastname: "Base64", PhotoThumbnail: "data:image/jpeg;base64,not-valid-base64!!!"}
	require.NoError(t, db.Create(&contact).Error)

	req, _ := http.NewRequest("GET", "/contacts/"+strconv.Itoa(int(contact.ID))+"/photo?thumbnail=true", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetProfilePicture_ValidThumbnail(t *testing.T) {
	cfg := &config.Config{ProfilePhotoDir: t.TempDir()}
	db, router := setupRouter()
	router.GET("/contacts/:id/photo", func(c *gin.Context) { GetProfilePicture(c, cfg) })

	var user models.User
	require.NoError(t, db.First(&user).Error)
	jpegBytes := newJPEGBytes(t, 10, 10)
	dataURL := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(jpegBytes)
	contact := models.Contact{UserID: user.ID, Firstname: "Good", Lastname: "Thumb", PhotoThumbnail: dataURL}
	require.NoError(t, db.Create(&contact).Error)

	req, _ := http.NewRequest("GET", "/contacts/"+strconv.Itoa(int(contact.ID))+"/photo?thumbnail=true", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/jpeg", w.Header().Get("Content-Type"))
	assert.Equal(t, jpegBytes, w.Body.Bytes())
}

func TestGetProfilePicture_NoPhotoSet(t *testing.T) {
	cfg := &config.Config{ProfilePhotoDir: t.TempDir()}
	db, router := setupRouter()
	router.GET("/contacts/:id/photo", func(c *gin.Context) { GetProfilePicture(c, cfg) })

	var user models.User
	require.NoError(t, db.First(&user).Error)
	contact := models.Contact{UserID: user.ID, Firstname: "No", Lastname: "Photo"}
	require.NoError(t, db.Create(&contact).Error)

	req, _ := http.NewRequest("GET", "/contacts/"+strconv.Itoa(int(contact.ID))+"/photo", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetProfilePicture_PathTraversalRejected(t *testing.T) {
	cfg := &config.Config{ProfilePhotoDir: t.TempDir()}
	db, router := setupRouter()
	router.GET("/contacts/:id/photo", func(c *gin.Context) { GetProfilePicture(c, cfg) })

	var user models.User
	require.NoError(t, db.First(&user).Error)
	contact := models.Contact{UserID: user.ID, Firstname: "Traversal", Lastname: "Attempt", Photo: "../../etc/passwd"}
	require.NoError(t, db.Create(&contact).Error)

	req, _ := http.NewRequest("GET", "/contacts/"+strconv.Itoa(int(contact.ID))+"/photo", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetProfilePicture_AbsolutePathRejected(t *testing.T) {
	cfg := &config.Config{ProfilePhotoDir: t.TempDir()}
	db, router := setupRouter()
	router.GET("/contacts/:id/photo", func(c *gin.Context) { GetProfilePicture(c, cfg) })

	var user models.User
	require.NoError(t, db.First(&user).Error)
	contact := models.Contact{UserID: user.ID, Firstname: "Absolute", Lastname: "Path", Photo: "/etc/passwd"}
	require.NoError(t, db.Create(&contact).Error)

	req, _ := http.NewRequest("GET", "/contacts/"+strconv.Itoa(int(contact.ID))+"/photo", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetProfilePicture_FileMissingOnDisk(t *testing.T) {
	cfg := &config.Config{ProfilePhotoDir: t.TempDir()}
	db, router := setupRouter()
	router.GET("/contacts/:id/photo", func(c *gin.Context) { GetProfilePicture(c, cfg) })

	var user models.User
	require.NoError(t, db.First(&user).Error)
	contact := models.Contact{UserID: user.ID, Firstname: "Missing", Lastname: "File", Photo: "does-not-exist.jpg"}
	require.NoError(t, db.Create(&contact).Error)

	req, _ := http.NewRequest("GET", "/contacts/"+strconv.Itoa(int(contact.ID))+"/photo", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetProfilePicture_ServesFileFromDisk(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{ProfilePhotoDir: dir}
	db, router := setupRouter()
	router.GET("/contacts/:id/photo", func(c *gin.Context) { GetProfilePicture(c, cfg) })

	var user models.User
	require.NoError(t, db.First(&user).Error)
	jpegBytes := newJPEGBytes(t, 10, 10)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "real_photo.jpg"), jpegBytes, 0644))
	contact := models.Contact{UserID: user.ID, Firstname: "Real", Lastname: "File", Photo: "real_photo.jpg"}
	require.NoError(t, db.Create(&contact).Error)

	req, _ := http.NewRequest("GET", "/contacts/"+strconv.Itoa(int(contact.ID))+"/photo", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, jpegBytes, w.Body.Bytes())
}

// --- AddPhotoToContact ---

func newMultipartPhotoRequest(t *testing.T, url, fieldName, filename string, data []byte) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if data != nil {
		part, err := writer.CreateFormFile(fieldName, filename)
		require.NoError(t, err)
		_, err = part.Write(data)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	req, _ := http.NewRequest("POST", url, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestAddPhotoToContact_DemoModeDisabled(t *testing.T) {
	require.NoError(t, os.Setenv("DEMO_MODE", "true"))
	defer os.Unsetenv("DEMO_MODE")

	cfg := &config.Config{ProfilePhotoDir: t.TempDir()}
	_, router := setupRouter()
	router.POST("/contacts/:id/photo", func(c *gin.Context) { AddPhotoToContact(c, cfg) })

	req := newMultipartPhotoRequest(t, "/contacts/1/photo", "photo", "photo.png", newPNGBytes(t, 10, 10))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAddPhotoToContact_InvalidContactID(t *testing.T) {
	cfg := &config.Config{ProfilePhotoDir: t.TempDir()}
	_, router := setupRouter()
	router.POST("/contacts/:id/photo", func(c *gin.Context) { AddPhotoToContact(c, cfg) })

	req := newMultipartPhotoRequest(t, "/contacts/abc/photo", "photo", "photo.png", newPNGBytes(t, 10, 10))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAddPhotoToContact_ContactNotFound(t *testing.T) {
	cfg := &config.Config{ProfilePhotoDir: t.TempDir()}
	_, router := setupRouter()
	router.POST("/contacts/:id/photo", func(c *gin.Context) { AddPhotoToContact(c, cfg) })

	req := newMultipartPhotoRequest(t, "/contacts/999/photo", "photo", "photo.png", newPNGBytes(t, 10, 10))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAddPhotoToContact_NoFileUploadedSavesContactUnchanged(t *testing.T) {
	cfg := &config.Config{ProfilePhotoDir: t.TempDir()}
	db, router := setupRouter()
	router.POST("/contacts/:id/photo", func(c *gin.Context) { AddPhotoToContact(c, cfg) })

	var user models.User
	require.NoError(t, db.First(&user).Error)
	contact := models.Contact{UserID: user.ID, Firstname: "No", Lastname: "Upload"}
	require.NoError(t, db.Create(&contact).Error)

	req := newMultipartPhotoRequest(t, "/contacts/"+strconv.Itoa(int(contact.ID))+"/photo", "", "", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var updated models.Contact
	require.NoError(t, db.First(&updated, contact.ID).Error)
	assert.Empty(t, updated.Photo)
}

func TestAddPhotoToContact_ValidUploadSetsPhotoAndThumbnail(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{ProfilePhotoDir: dir}
	db, router := setupRouter()
	router.POST("/contacts/:id/photo", func(c *gin.Context) { AddPhotoToContact(c, cfg) })

	var user models.User
	require.NoError(t, db.First(&user).Error)
	contact := models.Contact{UserID: user.ID, Firstname: "New", Lastname: "Photo"}
	require.NoError(t, db.Create(&contact).Error)

	req := newMultipartPhotoRequest(t, "/contacts/"+strconv.Itoa(int(contact.ID))+"/photo", "photo", "photo.png", newPNGBytes(t, 50, 50))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var updated models.Contact
	require.NoError(t, db.First(&updated, contact.ID).Error)
	assert.True(t, strings.HasSuffix(updated.Photo, "_photo.jpg"))
	assert.True(t, strings.HasPrefix(updated.PhotoThumbnail, "data:image/jpeg;base64,"))

	_, err := os.Stat(filepath.Join(dir, updated.Photo))
	require.NoError(t, err, "uploaded photo must be written to disk")
}

func TestAddPhotoToContact_ReplacingPhotoDeletesOldFile(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{ProfilePhotoDir: dir}
	db, router := setupRouter()
	router.POST("/contacts/:id/photo", func(c *gin.Context) { AddPhotoToContact(c, cfg) })

	var user models.User
	require.NoError(t, db.First(&user).Error)

	// Seed an existing "old" photo on disk and on the contact record.
	oldPhotoName := "old_photo.jpg"
	require.NoError(t, os.WriteFile(filepath.Join(dir, oldPhotoName), newJPEGBytes(t, 10, 10), 0644))
	contact := models.Contact{UserID: user.ID, Firstname: "Replace", Lastname: "Photo", Photo: oldPhotoName}
	require.NoError(t, db.Create(&contact).Error)

	req := newMultipartPhotoRequest(t, "/contacts/"+strconv.Itoa(int(contact.ID))+"/photo", "photo", "new.png", newPNGBytes(t, 20, 20))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	_, err := os.Stat(filepath.Join(dir, oldPhotoName))
	assert.True(t, os.IsNotExist(err), "old photo file should have been removed after successful replacement")
}

func TestAddPhotoToContact_FileTooLarge(t *testing.T) {
	cfg := &config.Config{ProfilePhotoDir: t.TempDir()}
	db, router := setupRouter()
	router.POST("/contacts/:id/photo", func(c *gin.Context) { AddPhotoToContact(c, cfg) })

	var user models.User
	require.NoError(t, db.First(&user).Error)
	contact := models.Contact{UserID: user.ID, Firstname: "Too", Lastname: "Big"}
	require.NoError(t, db.Create(&contact).Error)

	oversized := bytes.Repeat([]byte{0xAB}, 10*1024*1024+1)
	req := newMultipartPhotoRequest(t, "/contacts/"+strconv.Itoa(int(contact.ID))+"/photo", "photo", "huge.png", oversized)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAddPhotoToContact_UnsupportedFormatFailsProcessing(t *testing.T) {
	cfg := &config.Config{ProfilePhotoDir: t.TempDir()}
	db, router := setupRouter()
	router.POST("/contacts/:id/photo", func(c *gin.Context) { AddPhotoToContact(c, cfg) })

	var user models.User
	require.NoError(t, db.First(&user).Error)
	contact := models.Contact{UserID: user.ID, Firstname: "Bad", Lastname: "Format"}
	require.NoError(t, db.Create(&contact).Error)

	req := newMultipartPhotoRequest(t, "/contacts/"+strconv.Itoa(int(contact.ID))+"/photo", "photo", "notes.txt",
		[]byte("this is plain text, not an image, and is padded out well past the 512-byte content sniffing window so DetectContentType reliably calls it text/plain rather than guessing something else from too little data to go on."))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- ProxyImage ---
//
// httputil.FetchImageFromURL's SSRF protections (blocked hosts, private IPs,
// redirect validation, size limits) are already covered directly in
// httputil/fetch_test.go. What's specific to this handler layer is request
// parsing and error-code mapping, covered below. A real successful fetch
// can't be exercised at this layer without reaching an actual public host,
// since FetchImageFromURL unconditionally refuses loopback/private targets
// (by design) -- there is no local httptest.Server that would pass its
// checks.

func TestProxyImage_MissingURLParam(t *testing.T) {
	_, router := setupRouter()
	router.GET("/proxy", ProxyImage)

	req, _ := http.NewRequest("GET", "/proxy", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProxyImage_BlockedSSRFTarget(t *testing.T) {
	_, router := setupRouter()
	router.GET("/proxy", ProxyImage)

	req, _ := http.NewRequest("GET", "/proxy?url=http://127.0.0.1/photo.jpg", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestProxyImage_RejectsNonHTTPScheme(t *testing.T) {
	_, router := setupRouter()
	router.GET("/proxy", ProxyImage)

	req, _ := http.NewRequest("GET", "/proxy?url=ftp://not-http-or-https.example.com/x.jpg", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
