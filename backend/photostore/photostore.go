// Package photostore holds the disk-I/O and encoding helpers for contact
// profile photos: saving an uploaded/synced photo to disk (with cropping,
// resizing, and thumbnail generation), reading it back for serialization,
// fetching a photo from a remote URL, and decoding an embedded photo value
// (raw base64, a data: URI, or a vCard PHOTO field).
//
// Extracted from backend/carddav/vcard_mapper.go per
// docs/fork-plan/50-integration-and-rebrand.md WP-73's photo-bridging
// prerequisite: backend/models needs this logic too (to bridge
// Contact.Photo/PhotoThumbnail <-> contactmodel.Card.Media), but carddav
// already imports models (carddav/auth.go, carddav/backend.go), so models
// importing carddav back would be an import cycle. This package has zero
// dependency on models/carddav/services (only external libraries and the
// existing, equally-independent httputil package), so all three can import
// it without a cycle.
package photostore

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"mycorrhizal/httputil"

	"github.com/emersion/go-vcard"
	"github.com/gen2brain/heic"
	"github.com/google/uuid"
	"github.com/nfnt/resize"
)

// SaveContactPhoto saves photo data to disk and generates a thumbnail.
// Returns the photo filename (relative to photoDir) and a base64 data-URL
// thumbnail suitable for storing directly on Contact.PhotoThumbnail.
func SaveContactPhoto(photoData []byte, mediaType string, photoDir string) (string, string, error) {
	if len(photoData) == 0 {
		return "", "", nil
	}

	// Detect content type if not provided
	if mediaType == "" {
		mediaType = http.DetectContentType(photoData)
	}

	// Decode the image
	var img image.Image
	var err error

	reader := bytes.NewReader(photoData)
	switch {
	case strings.Contains(mediaType, "jpeg") || strings.Contains(mediaType, "jpg"):
		img, err = jpeg.Decode(reader)
	case strings.Contains(mediaType, "png"):
		img, err = png.Decode(reader)
	case strings.Contains(mediaType, "heic") || strings.Contains(mediaType, "heif"):
		img, err = heic.Decode(reader)
	default:
		// Check for HEIC magic bytes if media type is unknown
		// HEIC files have "ftyp" followed by "heic", "heix", "hevc", "hevx", or "mif1" at byte 4
		if len(photoData) >= 12 && string(photoData[4:8]) == "ftyp" {
			brand := string(photoData[8:12])
			if brand == "heic" || brand == "heix" || brand == "hevc" || brand == "hevx" || brand == "mif1" {
				img, err = heic.Decode(reader)
			}
		}
		// Try to decode as JPEG first, then PNG, then HEIC
		if img == nil {
			img, err = jpeg.Decode(reader)
			if err != nil {
				reader.Seek(0, 0)
				img, err = png.Decode(reader)
				if err != nil {
					reader.Seek(0, 0)
					img, err = heic.Decode(reader)
				}
			}
		}
	}
	if err != nil {
		return "", "", err
	}

	// Crop to centered square if rectangular
	img = cropToSquare(img)

	// Generate unique filename
	baseFilename := uuid.New().String()
	photoPath := baseFilename + "_photo.jpg"

	// Ensure directory exists
	if err := os.MkdirAll(photoDir, os.ModePerm); err != nil {
		return "", "", err
	}

	// Save photo (max 400x400, only downscale - smaller images keep their size)
	const maxPhotoSize = 400
	bounds := img.Bounds()
	photoImg := img
	if bounds.Dx() > maxPhotoSize || bounds.Dy() > maxPhotoSize {
		photoImg = resize.Resize(maxPhotoSize, maxPhotoSize, img, resize.Lanczos3)
	}
	fullPhotoPath := filepath.Join(photoDir, photoPath)
	outFile, err := os.Create(fullPhotoPath)
	if err != nil {
		return "", "", err
	}
	defer outFile.Close()

	if err := jpeg.Encode(outFile, photoImg, &jpeg.Options{Quality: 85}); err != nil {
		return "", "", err
	}

	// Create thumbnail and encode as base64 data URL (48x48)
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

// FetchPhotoFromURL fetches a photo from a URL with SSRF protection.
// Returns the photo data, media type, and any error.
func FetchPhotoFromURL(photoURL string) ([]byte, string, error) {
	return httputil.FetchImageFromURL(photoURL)
}

// ReadContactPhoto reads a contact's photo for serialization: it prefers the
// full-resolution photo on disk (photoPath, resolved under photoDir) and
// falls back to the base64 data-URL thumbnail (photoThumbnail) if the
// on-disk file is unavailable. Returns raw base64 data (no "data:" prefix)
// and the resolved media type; both are "" if no photo is available.
//
// Takes the specific fields it needs (rather than *models.Contact) so this
// package has zero dependency on backend/models — see the package doc
// comment.
func ReadContactPhoto(photoPath, photoThumbnail, photoDir string) (string, string) {
	// Try to read full photo from disk
	if photoPath != "" && photoDir != "" {
		fullPath := filepath.Join(photoDir, photoPath)
		data, err := os.ReadFile(fullPath)
		if err == nil {
			mediaType := http.DetectContentType(data)
			return base64.StdEncoding.EncodeToString(data), mediaType
		}
	}

	// Fall back to thumbnail (already base64)
	if photoThumbnail != "" && strings.HasPrefix(photoThumbnail, "data:") {
		// Parse data URL: data:image/jpeg;base64,<data>
		parts := strings.SplitN(photoThumbnail, ",", 2)
		if len(parts) == 2 {
			// Extract media type from first part
			mediaType := "image/jpeg"
			if strings.Contains(parts[0], "image/png") {
				mediaType = "image/png"
			}
			return parts[1], mediaType
		}
	}

	return "", ""
}

// ReadContactPhotoDataURI is ReadContactPhoto, but returns the result as a
// single "data:<mediaType>;base64,<data>" URI (plus the resolved media type)
// rather than raw base64 data. "" (both return values empty) means no photo
// is available. This is the encoding convention already used by the
// vcard4/vcard3 adapters for contactmodel.Resource{Kind:"photo"}.URI (see
// vcard3/adapter.go's parseDataURI/exportMediaField and vcard4/adapter.go's
// resource import/export, which treat a Resource's URI verbatim as this same
// "data:" shape for embedded PHOTO values) — used by
// models.RecordFromContact to build the Card.Media{Kind:"photo"} entry.
func ReadContactPhotoDataURI(photoPath, photoThumbnail, photoDir string) (uri string, mediaType string) {
	data, mt := ReadContactPhoto(photoPath, photoThumbnail, photoDir)
	if data == "" {
		return "", ""
	}
	if mt == "" {
		mt = "image/jpeg"
	}
	return "data:" + mt + ";base64," + data, mt
}

// DecodePhotoURI decodes a photo property value that may be an http(s) URL,
// a "data:" URI, or raw base64 (with mediaTypeHint supplying the media type
// when the value itself doesn't carry one, e.g. a vCard 3.0 ENCODING=b/TYPE=
// field or a neutral Resource with a separate MediaType field).
//
// Returns:
//   - data: the decoded photo bytes (nil if the value is a remote URL, or if
//     decoding failed)
//   - mediaType: the resolved media type
//   - url: the http(s) URL, if the value was a remote reference rather than
//     embedded data (data is nil in that case; the caller is expected to
//     fetch it separately, e.g. via FetchPhotoFromURL)
func DecodePhotoURI(value, mediaTypeHint string) (data []byte, mediaType string, url string) {
	if value == "" {
		return nil, "", ""
	}

	mediaType = mediaTypeHint

	// Google VCF format may have spaces/newlines in URLs that need to be removed.
	cleanValue := strings.ReplaceAll(value, " ", "")
	cleanValue = strings.ReplaceAll(cleanValue, "\n", "")
	cleanValue = strings.ReplaceAll(cleanValue, "\r", "")

	if strings.HasPrefix(cleanValue, "http://") || strings.HasPrefix(cleanValue, "https://") {
		// It's a URL-based photo, return the URL for later fetching.
		return nil, mediaType, cleanValue
	}

	// Check if it's a data URI.
	if strings.HasPrefix(value, "data:") {
		parts := strings.SplitN(value, ",", 2)
		if len(parts) == 2 {
			if mediaType == "" && strings.Contains(parts[0], "image/") {
				// Extract media type from data URI.
				start := strings.Index(parts[0], "image/")
				end := strings.Index(parts[0][start:], ";")
				if end == -1 {
					mediaType = parts[0][start:]
				} else {
					mediaType = parts[0][start : start+end]
				}
			}
			value = parts[1]
		}
	}

	// Decode base64.
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, "", ""
	}

	return decoded, mediaType, ""
}

// ExtractPhotoData extracts binary photo data from a vCard PHOTO field.
// Returns: photoData (bytes), mediaType (string), photoURL (string if
// URL-based photo).
func ExtractPhotoData(field *vcard.Field) ([]byte, string, string) {
	if field == nil || field.Value == "" {
		return nil, "", ""
	}

	mediaType := ""
	// Check for MEDIATYPE or TYPE parameter.
	if mt := field.Params.Get("MEDIATYPE"); mt != "" {
		mediaType = mt
	} else if t := field.Params.Get("TYPE"); t != "" {
		mediaType = "image/" + strings.ToLower(t)
	}

	return DecodePhotoURI(field.Value, mediaType)
}
