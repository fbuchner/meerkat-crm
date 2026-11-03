package controllers

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"meerkat/config"
	"meerkat/middleware"
	"meerkat/models"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// A minimal real HEIC image, from the test data of github.com/gen2brain/heic (MIT).
const tinyHEICBase64 = "AAAAHGZ0eXBoZWljAAAAAG1pZjFoZWljbWlhZgAAAY5tZXRhAAAAAAAAACFoZGxyAAAAAAAAAABwaWN0AAAAAAAAAAAAAAAAAAAAAA5waXRtAAAAAAABAAAANGlsb2MAAAAAREAAAgABAAAAAAGyAAEAAAAAAAAB1AACAAAAAAOGAAEAAAAAAAAA5AAAADhpaW5mAAAAAAACAAAAFWluZmUCAAAAAAEAAGh2YzEAAAAAFWluZmUCAAABAAIAAEV4aWYAAAAAzWlwcnAAAACuaXBjbwAAAHlodmNDAQNwAAAAAAAAAAAAWvAA/P34+AAADwNgAAEAGEABDAH//wNwAAADAJAAAAMAAAMAWroCQGEAAQAsQgEBA3AAAAMAkAAAAwAAAwBaoAUCAeFlupJKa5uAhoMCAAADADIAAAMAAhBiAAEAB0QBwXKwYkAAAAAUaXNwZQAAAAAAAAKAAAAB4AAAABBwaXhpAAAAAAMICAgAAAAJaXJvdAMAAAAXaXBtYQAAAAAAAAABAAEEgQIDhAAAABppcmVmAAAAAAAAAA5jZHNjAAIAAQABAAACwG1kYXQAAAHQKAGvExB7DZ2ufw1k5TKPE//K+YnsdKChs9tP9NRKDeHdR62JE7rBlMqcWFV8SfFPnuCGhviCUvtAd8ZvhLPJ9bLynm9+beFUnrRf354QX0OrRr8ArMJ1fxeAAbLu0VkoI8vAPtKjDIpjCsIC6aulFw6/BFJLAp+J+oueLdF5W7Ld14VzcYejal/pF6oLQxQBEwPoAAADAAADADXgqLFxxDLFToGyXIAlkg25bSijcf+rh9IO04Te79jOlkkRCxTJ5ODQ3mziXw15CIvPfQYKbUfEHoqhWcHAmtFuGLCkZNwotC2m+xbwOIdycWk4afqgRg2X8rzLx6AAAAMAAAMAAAMAAAMAHNAb4h3ClqEPq2GMdJEDvzFT/ciuf3hCSR8fwvx5pOlx1JF0FI2Fv9IaSBrxKLMIqIAC6oAe7AQFAS0AmeBG0CPpJN/5+QRrrheOQ68HxtVVq9L1de4kDTrJykpwnBrZ5TNfjIMXFZEEYfSPRbeKib5Mmm7Os8ytfd4tQeua+s9Zb2m+aoBUTl9FZcZM5fimyYD1BCsl15Ib5DnaexxRaySFnbcy2AAB3AAAAwAAAwAABmyF4lqV8pR+Cks2lPbJ1RXKiAAAAwAAGVAAAAAGRXhpZgAATU0AKgAAAAgACAEPAAIAAAAIAAAAbgEQAAIAAAAJAAAAdgESAAMAAAABAAYAAAEaAAUAAAABAAAAgAEbAAUAAAABAAAAiAEoAAMAAAABAAEAAAITAAMAAAABAAEAAIdpAAQAAAABAAAAkAAAAABUZXN0Q2FtAE1vZGVsMTIzAAAAAAABAAAAAQAAAAEAAAABAAWCnQAFAAAAAQAAANKIJwADAAAAAQMgAACQAAAHAAAABDAyMzKRAQAHAAAABAECAwCgAQADAAAAAf//AAAAAAAAAAAAHAAAAAU="

type photoFixture struct {
	db      *gorm.DB
	router  *gin.Engine
	dir     string
	user    models.User
	contact models.Contact
	other   models.Contact
}

func setupPhotoFixture(t *testing.T) photoFixture {
	t.Helper()
	db := newTestDB(t)
	user := newTestUser(t, db, "owner", false)
	stranger := newTestUser(t, db, "stranger", false)
	dir := t.TempDir()
	cfg := &config.Config{ProfilePhotoDir: dir}

	router := routerAs(db, user.ID, *cfg)
	router.POST("/contacts/:id/profile_picture", func(c *gin.Context) { AddPhotoToContact(c, cfg) })
	router.GET("/contacts/:id/profile_picture", func(c *gin.Context) { GetProfilePicture(c, cfg) })

	return photoFixture{
		db: db, router: router, dir: dir, user: user,
		contact: newTestContact(t, db, user.ID, "Mine"),
		other:   newTestContact(t, db, stranger.ID, "Theirs"),
	}
}

func (f photoFixture) picturePath(id uint) string {
	return fmt.Sprintf("/contacts/%d/profile_picture", id)
}

func (f photoFixture) reload(t *testing.T) models.Contact {
	t.Helper()
	var contact models.Contact
	require.NoError(t, f.db.First(&contact, f.contact.ID).Error)
	return contact
}

func (f photoFixture) files(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(f.dir)
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

// sidesAndCenterImage is red with a green centered band, so cropping off-center is visible.
func sidesAndCenterImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			c := color.RGBA{R: 255, A: 255}
			if x > width/3 && x < 2*width/3 {
				c = color.RGBA{G: 255, A: 255}
			}
			img.Set(x, y, c)
		}
	}
	return img
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, nil))
	return buf.Bytes()
}

func TestAddPhoto_PNGIsCroppedResizedAndStoredAsJPEG(t *testing.T) {
	f := setupPhotoFixture(t)

	w := doUpload(t, f.router, f.picturePath(f.contact.ID), "photo", "me.png", encodePNG(t, sidesAndCenterImage(900, 450)))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	contact := f.reload(t)
	assert.True(t, strings.HasSuffix(contact.Photo, "_photo.jpg"), contact.Photo)
	assert.Equal(t, []string{contact.Photo}, f.files(t))
	assert.True(t, strings.HasPrefix(contact.PhotoThumbnail, "data:image/jpeg;base64,"))

	file, err := os.Open(filepath.Join(f.dir, contact.Photo))
	require.NoError(t, err)
	defer file.Close()
	saved, err := jpeg.Decode(file)
	require.NoError(t, err, "the stored photo must be a JPEG whatever was uploaded")
	assert.Equal(t, image.Pt(400, 400), saved.Bounds().Size(), "cropped to a square, then capped at 400px")

	r, g, _, _ := saved.At(200, 200).RGBA()
	assert.Greater(t, g, r, "the crop must keep the center of the image")
}

func TestAddPhoto_SmallSquareJPEGIsNotUpscaled(t *testing.T) {
	f := setupPhotoFixture(t)

	w := doUpload(t, f.router, f.picturePath(f.contact.ID), "photo", "me.jpg", encodeJPEG(t, sidesAndCenterImage(60, 60)))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	file, err := os.Open(filepath.Join(f.dir, f.reload(t).Photo))
	require.NoError(t, err)
	defer file.Close()
	config, err := jpeg.DecodeConfig(file)
	require.NoError(t, err)
	assert.Equal(t, 60, config.Width)
	assert.Equal(t, 60, config.Height)
}

func TestAddPhoto_ReplacingRemovesTheOldFile(t *testing.T) {
	f := setupPhotoFixture(t)
	path := f.picturePath(f.contact.ID)

	require.Equal(t, http.StatusOK, doUpload(t, f.router, path, "photo", "a.png", encodePNG(t, sidesAndCenterImage(50, 50))).Code)
	first := f.reload(t).Photo
	require.Equal(t, http.StatusOK, doUpload(t, f.router, path, "photo", "b.png", encodePNG(t, sidesAndCenterImage(50, 50))).Code)
	second := f.reload(t).Photo

	assert.NotEqual(t, first, second)
	assert.Equal(t, []string{second}, f.files(t), "the replaced photo must not stay on disk")
}

func TestAddPhoto_HEICIsConvertedToJPEG(t *testing.T) {
	f := setupPhotoFixture(t)
	heicBytes, err := base64.StdEncoding.DecodeString(tinyHEICBase64)
	require.NoError(t, err)

	w := doUpload(t, f.router, f.picturePath(f.contact.ID), "photo", "iphone.heic", heicBytes)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	file, err := os.Open(filepath.Join(f.dir, f.reload(t).Photo))
	require.NoError(t, err)
	defer file.Close()
	_, err = jpeg.Decode(file)
	assert.NoError(t, err, "HEIC uploads must be stored as JPEG")
}

func TestAddPhoto_RejectsContentThatIsNotASupportedImage(t *testing.T) {
	gif := append([]byte("GIF89a"), bytes.Repeat([]byte{0}, 64)...)
	truncatedPNG := encodePNG(t, sidesAndCenterImage(30, 30))[:40]
	truncatedHEIC := append([]byte("\x00\x00\x00\x18ftypheic"), bytes.Repeat([]byte{0}, 64)...)

	for name, content := range map[string][]byte{
		"plain text with an image name": []byte("this is not an image"),
		"gif":                           gif,
		"executable":                    []byte("#!/bin/sh\nrm -rf /\n"),
		"corrupt png":                   truncatedPNG,
		"corrupt heic":                  truncatedHEIC,
		"empty":                         {},
	} {
		t.Run(name, func(t *testing.T) {
			f := setupPhotoFixture(t)

			w := doUpload(t, f.router, f.picturePath(f.contact.ID), "photo", "photo.jpg", content)

			assert.GreaterOrEqual(t, w.Code, 400, w.Body.String())
			assert.Empty(t, f.reload(t).Photo)
			assert.Empty(t, f.reload(t).PhotoThumbnail)
			assert.Empty(t, f.files(t), "nothing may be written for a rejected upload")
		})
	}
}

func TestAddPhoto_WrongFileTypeIsAClientError(t *testing.T) {
	skipKnownBug(t, "photo_controller.go:159 maps every processing failure, including an unsupported format, to a 500")
	f := setupPhotoFixture(t)

	w := doUpload(t, f.router, f.picturePath(f.contact.ID), "photo", "notes.txt", []byte("this is not an image"))

	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

func TestAddPhoto_OversizeFileIsRejected(t *testing.T) {
	f := setupPhotoFixture(t)

	w := doUpload(t, f.router, f.picturePath(f.contact.ID), "photo", "huge.jpg", make([]byte, 10*1024*1024+1))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "too large")
	assert.Empty(t, f.files(t))
	assert.Empty(t, f.reload(t).Photo)
}

func TestAddPhoto_OversizeFileBehindTheBodyLimitIsNotReportedAsSuccess(t *testing.T) {
	skipKnownBug(t, "photo_controller.go:146 treats any FormFile error as no file, so main.go's 10MB body limit turns an oversize upload into a 200")
	f := setupPhotoFixture(t)
	cfg := &config.Config{ProfilePhotoDir: f.dir}
	router := routerAs(f.db, f.user.ID, *cfg)
	router.Use(middleware.DefaultBodySizeLimitMiddleware())
	router.POST("/contacts/:id/profile_picture", func(c *gin.Context) { AddPhotoToContact(c, cfg) })

	w := doUpload(t, router, f.picturePath(f.contact.ID), "photo", "huge.jpg", make([]byte, 11*1024*1024))

	assert.GreaterOrEqual(t, w.Code, 400, "the photo was not stored, so the client must not be told it succeeded")
}

func TestAddPhoto_AnotherUsersContactIsNotFound(t *testing.T) {
	f := setupPhotoFixture(t)

	w := doUpload(t, f.router, f.picturePath(f.other.ID), "photo", "me.png", encodePNG(t, sidesAndCenterImage(50, 50)))

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Empty(t, f.files(t))
	var other models.Contact
	require.NoError(t, f.db.First(&other, f.other.ID).Error)
	assert.Empty(t, other.Photo)
}

func TestAddPhoto_InvalidContactID(t *testing.T) {
	f := setupPhotoFixture(t)
	w := doUpload(t, f.router, "/contacts/abc/profile_picture", "photo", "me.png", encodePNG(t, sidesAndCenterImage(50, 50)))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, f.files(t))
}

func TestAddPhoto_DisabledInDemoMode(t *testing.T) {
	t.Setenv("DEMO_MODE", "true")
	f := setupPhotoFixture(t)

	w := doUpload(t, f.router, f.picturePath(f.contact.ID), "photo", "me.png", encodePNG(t, sidesAndCenterImage(50, 50)))

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Empty(t, f.files(t))
	assert.Empty(t, f.reload(t).Photo)
}

func TestAddPhoto_WithoutAFileLeavesThePhotoAlone(t *testing.T) {
	f := setupPhotoFixture(t)
	require.NoError(t, f.db.Model(&models.Contact{}).Where("id = ?", f.contact.ID).Update("photo", "kept.jpg").Error)

	doRequest(f.router, http.MethodPost, f.picturePath(f.contact.ID), nil, "")

	assert.Equal(t, "kept.jpg", f.reload(t).Photo)
}

func TestGetProfilePicture_ServesTheFullPhoto(t *testing.T) {
	f := setupPhotoFixture(t)
	require.NoError(t, os.WriteFile(filepath.Join(f.dir, "mine.jpg"), []byte("jpeg-bytes"), 0o600))
	require.NoError(t, f.db.Model(&models.Contact{}).Where("id = ?", f.contact.ID).Update("photo", "mine.jpg").Error)

	w := doRequest(f.router, http.MethodGet, f.picturePath(f.contact.ID), nil, "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "jpeg-bytes", w.Body.String())
}

func TestGetProfilePicture_RefusesPathsOutsideThePhotoDir(t *testing.T) {
	parent := t.TempDir()
	secret := filepath.Join(parent, "secret.txt")
	require.NoError(t, os.WriteFile(secret, []byte("top-secret"), 0o600))

	for name, photo := range map[string]string{
		"parent traversal": "../secret.txt",
		"nested traversal": "a/../../secret.txt",
		"absolute path":    secret,
	} {
		t.Run(name, func(t *testing.T) {
			f := setupPhotoFixture(t)
			dir := filepath.Join(parent, "photos")
			require.NoError(t, os.MkdirAll(dir, 0o700))
			cfg := &config.Config{ProfilePhotoDir: dir}
			router := routerAs(f.db, f.user.ID, *cfg)
			router.GET("/contacts/:id/profile_picture", func(c *gin.Context) { GetProfilePicture(c, cfg) })
			require.NoError(t, f.db.Model(&models.Contact{}).Where("id = ?", f.contact.ID).Update("photo", photo).Error)

			w := doRequest(router, http.MethodGet, f.picturePath(f.contact.ID), nil, "")

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.NotContains(t, w.Body.String(), "top-secret")
		})
	}
}

func TestGetProfilePicture_Thumbnail(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("thumb-bytes"))

	for name, tc := range map[string]struct {
		thumbnail string
		status    int
	}{
		"valid data url":   {"data:image/jpeg;base64," + encoded, http.StatusOK},
		"none stored":      {"", http.StatusNotFound},
		"legacy file path": {"legacy_thumb.jpg", http.StatusNotFound},
		"no payload":       {"data:image/jpeg;base64", http.StatusInternalServerError},
		"corrupt base64":   {"data:image/jpeg;base64,!!!", http.StatusInternalServerError},
	} {
		t.Run(name, func(t *testing.T) {
			f := setupPhotoFixture(t)
			require.NoError(t, f.db.Model(&models.Contact{}).Where("id = ?", f.contact.ID).Update("photo_thumbnail", tc.thumbnail).Error)

			w := doRequest(f.router, http.MethodGet, f.picturePath(f.contact.ID)+"?thumbnail=true", nil, "")

			assert.Equal(t, tc.status, w.Code)
			if tc.status == http.StatusOK {
				assert.Equal(t, "image/jpeg", w.Header().Get("Content-Type"))
				assert.Equal(t, "thumb-bytes", w.Body.String())
			}
		})
	}
}

func TestGetProfilePicture_NotFoundCases(t *testing.T) {
	f := setupPhotoFixture(t)

	assert.Equal(t, http.StatusBadRequest, doRequest(f.router, http.MethodGet, "/contacts/abc/profile_picture", nil, "").Code)
	assert.Equal(t, http.StatusNotFound, doRequest(f.router, http.MethodGet, f.picturePath(f.contact.ID), nil, "").Code, "no photo set")

	require.NoError(t, f.db.Model(&models.Contact{}).Where("id = ?", f.contact.ID).Update("photo", "gone.jpg").Error)
	assert.Equal(t, http.StatusNotFound, doRequest(f.router, http.MethodGet, f.picturePath(f.contact.ID), nil, "").Code, "file missing on disk")
}

func TestGetProfilePicture_AnotherUsersContactIsNotServed(t *testing.T) {
	f := setupPhotoFixture(t)
	require.NoError(t, os.WriteFile(filepath.Join(f.dir, "theirs.jpg"), []byte("their-photo"), 0o600))
	thumbnail := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString([]byte("their-thumb"))
	require.NoError(t, f.db.Model(&models.Contact{}).Where("id = ?", f.other.ID).
		Updates(map[string]any{"photo": "theirs.jpg", "photo_thumbnail": thumbnail}).Error)

	for _, suffix := range []string{"", "?thumbnail=true"} {
		w := doRequest(f.router, http.MethodGet, f.picturePath(f.other.ID)+suffix, nil, "")
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.NotContains(t, w.Body.String(), "their-")
	}
}

func TestProxyImage_RequiresURL(t *testing.T) {
	db := newTestDB(t)
	router := routerAs(db, 1, config.Config{})
	router.GET("/proxy/image", ProxyImage)

	assert.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodGet, "/proxy/image", nil, "").Code)
}

func TestProxyImage_DoesNotFetchInternalAddresses(t *testing.T) {
	var hits atomic.Int32
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("internal-secret"))
	}))
	defer internal.Close()

	db := newTestDB(t)
	router := routerAs(db, 1, config.Config{})
	router.GET("/proxy/image", ProxyImage)

	w := doRequest(router, http.MethodGet, "/proxy/image?url="+url.QueryEscape(internal.URL+"/x.png"), nil, "")

	assert.GreaterOrEqual(t, w.Code, 400)
	assert.NotContains(t, w.Body.String(), "internal-secret")
	assert.Zero(t, hits.Load(), "a loopback URL must never be requested")
}
