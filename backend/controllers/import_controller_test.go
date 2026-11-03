package controllers

import (
	"bytes"
	"encoding/base64"
	"image/jpeg"
	"meerkat/config"
	"meerkat/middleware"
	"meerkat/models"
	"meerkat/services"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	testVCF = "BEGIN:VCARD\r\nVERSION:3.0\r\nUID:grace-1\r\nFN:Grace Hopper\r\nN:Hopper;Grace;;;\r\nEMAIL:grace@example.com\r\nEND:VCARD\r\n" +
		"BEGIN:VCARD\r\nVERSION:3.0\r\nUID:alan-1\r\nFN:Alan Turing\r\nN:Turing;Alan;;;\r\nEND:VCARD\r\n"
	testCSV = "First Name,Last Name,Email\nAda,Lovelace,ada@example.com\nCharles,Babbage,charles@example.com\n"
)

type importFixture struct {
	db     *gorm.DB
	dir    string
	user   models.User
	router *gin.Engine
	// strangerRouter acts as a second user against the same data and sessions.
	strangerRouter *gin.Engine
	stranger       models.User
}

func setupImport(t *testing.T) importFixture {
	t.Helper()
	db := newTestDB(t)
	dir := t.TempDir()
	cfg := &config.Config{ProfilePhotoDir: dir}
	user := newTestUser(t, db, "owner", false)
	stranger := newTestUser(t, db, "stranger", false)

	build := func(userID uint) *gin.Engine {
		router := routerAs(db, userID, *cfg)
		confirm := middleware.ValidateJSONMiddleware(&models.ImportConfirmRequest{})
		router.POST("/import/upload", UploadCSVForImport)
		router.POST("/import/preview", middleware.ValidateJSONMiddleware(&models.ImportPreviewRequest{}), PreviewImport)
		router.POST("/import/confirm", confirm, ConfirmImport)
		router.POST("/import/vcf/upload", func(c *gin.Context) { UploadVCFForImport(c, cfg) })
		router.POST("/import/vcf/confirm", confirm, func(c *gin.Context) { ConfirmVCFImport(c, cfg) })
		return router
	}
	return importFixture{db: db, dir: dir, user: user, stranger: stranger, router: build(user.ID), strangerRouter: build(stranger.ID)}
}

func (f importFixture) contactsOf(t *testing.T, userID uint) []models.Contact {
	t.Helper()
	var contacts []models.Contact
	require.NoError(t, f.db.Where("user_id = ?", userID).Order("id").Find(&contacts).Error)
	return contacts
}

func addAll(rows int) []models.RowImportAction {
	actions := make([]models.RowImportAction, rows)
	for i := range actions {
		actions[i] = models.RowImportAction{RowIndex: i, Action: "add"}
	}
	return actions
}

func TestImportUpload_RejectsBadFiles(t *testing.T) {
	for name, tc := range map[string]struct {
		path, field, filename string
		content               []byte
		reason                string
	}{
		"csv without file":    {"/import/upload", "wrong", "a.csv", []byte(testCSV), "No file uploaded"},
		"csv wrong extension": {"/import/upload", "file", "a.txt", []byte(testCSV), "CSV"},
		"csv oversize":        {"/import/upload", "file", "a.csv", make([]byte, services.MaxCSVSize+1), "too large"},
		"csv empty":           {"/import/upload", "file", "a.csv", nil, ""},
		"vcf without file":    {"/import/vcf/upload", "wrong", "a.vcf", []byte(testVCF), "No file uploaded"},
		"vcf wrong extension": {"/import/vcf/upload", "file", "a.txt", []byte(testVCF), "VCF"},
		"vcf oversize":        {"/import/vcf/upload", "file", "a.vcf", make([]byte, services.MaxVCFSize+1), "too large"},
	} {
		t.Run(name, func(t *testing.T) {
			f := setupImport(t)

			w := doUpload(t, f.router, tc.path, tc.field, tc.filename, tc.content)

			assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			assert.Contains(t, w.Body.String(), tc.reason)
			assert.Empty(t, f.contactsOf(t, f.user.ID))
		})
	}
}

func TestVCFImport_GarbageImportsNothing(t *testing.T) {
	f := setupImport(t)

	w := doUpload(t, f.router, "/import/vcf/upload", "file", "a.vcf", []byte("this is not a vcard\nat all\n"))

	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	assert.Empty(t, f.contactsOf(t, f.user.ID))
}

func TestVCFImport_SuccessfulFlow(t *testing.T) {
	f := setupImport(t)

	w := doUpload(t, f.router, "/import/vcf/upload", "file", "contacts.VCF", []byte(testVCF))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	preview := decodeJSON[models.ImportPreviewResponse](t, w)
	assert.NotEmpty(t, preview.SessionID)
	assert.Equal(t, 2, preview.TotalRows)
	assert.Equal(t, 2, preview.ValidRows)
	assert.Empty(t, f.contactsOf(t, f.user.ID), "the upload only previews")

	w = doJSON(t, f.router, http.MethodPost, "/import/vcf/confirm", models.ImportConfirmRequest{
		SessionID: preview.SessionID,
		Actions:   []models.RowImportAction{{RowIndex: 0, Action: "add"}, {RowIndex: 1, Action: "skip"}},
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	result := decodeJSON[models.ImportResult](t, w)
	assert.Equal(t, 1, result.Created)
	assert.Equal(t, 1, result.Skipped)
	assert.Empty(t, result.Errors)

	contacts := f.contactsOf(t, f.user.ID)
	require.Len(t, contacts, 1)
	assert.Equal(t, "Grace", contacts[0].Firstname)
	assert.Equal(t, "Hopper", contacts[0].Lastname)
}

func TestVCFImport_DuplicateIsDetectedAndMerged(t *testing.T) {
	f := setupImport(t)
	uid := "grace-1"
	existing := models.Contact{UserID: f.user.ID, Firstname: "Grace", Lastname: "Hopper", VCardUID: uid}
	require.NoError(t, f.db.Create(&existing).Error)

	w := doUpload(t, f.router, "/import/vcf/upload", "file", "a.vcf", []byte(testVCF))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	preview := decodeJSON[models.ImportPreviewResponse](t, w)
	assert.Equal(t, 1, preview.DuplicateCount)

	w = doJSON(t, f.router, http.MethodPost, "/import/vcf/confirm", models.ImportConfirmRequest{
		SessionID: preview.SessionID,
		Actions:   []models.RowImportAction{{RowIndex: 0, Action: "update"}},
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	result := decodeJSON[models.ImportResult](t, w)
	assert.Equal(t, 1, result.Updated)
	assert.Len(t, f.contactsOf(t, f.user.ID), 1, "merging must not add a second contact")
}

func TestVCFImport_ContactOfAnotherUserIsNotADuplicate(t *testing.T) {
	f := setupImport(t)
	uid := "grace-1"
	theirs := models.Contact{UserID: f.stranger.ID, Firstname: "Grace", Lastname: "Hopper", VCardUID: uid}
	require.NoError(t, f.db.Create(&theirs).Error)

	w := doUpload(t, f.router, "/import/vcf/upload", "file", "a.vcf", []byte(testVCF))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Zero(t, decodeJSON[models.ImportPreviewResponse](t, w).DuplicateCount)
}

func TestVCFImport_EmbeddedPhotoIsSaved(t *testing.T) {
	f := setupImport(t)
	var photo bytes.Buffer
	require.NoError(t, jpeg.Encode(&photo, sidesAndCenterImage(40, 40), nil))
	card := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Pic Ture\r\nN:Ture;Pic;;;\r\nPHOTO;ENCODING=b;TYPE=JPEG:" +
		base64.StdEncoding.EncodeToString(photo.Bytes()) + "\r\nEND:VCARD\r\n"

	w := doUpload(t, f.router, "/import/vcf/upload", "file", "a.vcf", []byte(card))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	preview := decodeJSON[models.ImportPreviewResponse](t, w)

	w = doJSON(t, f.router, http.MethodPost, "/import/vcf/confirm", models.ImportConfirmRequest{SessionID: preview.SessionID, Actions: addAll(1)})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	contacts := f.contactsOf(t, f.user.ID)
	require.Len(t, contacts, 1)
	require.NotEmpty(t, contacts[0].Photo)
	assert.True(t, strings.HasPrefix(contacts[0].PhotoThumbnail, "data:image/"))
	_, err := os.Stat(filepath.Join(f.dir, contacts[0].Photo))
	assert.NoError(t, err, "the photo must land in the configured directory")
}

func TestVCFImport_SessionOfAnotherUserCannotBeConfirmed(t *testing.T) {
	f := setupImport(t)
	w := doUpload(t, f.router, "/import/vcf/upload", "file", "a.vcf", []byte(testVCF))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	sessionID := decodeJSON[models.ImportPreviewResponse](t, w).SessionID

	w = doJSON(t, f.strangerRouter, http.MethodPost, "/import/vcf/confirm", models.ImportConfirmRequest{SessionID: sessionID, Actions: addAll(2)})

	assert.GreaterOrEqual(t, w.Code, 400)
	assert.Less(t, w.Code, 500)
	assert.NotContains(t, w.Body.String(), "Grace")
	assert.Empty(t, f.contactsOf(t, f.stranger.ID))
	assert.Empty(t, f.contactsOf(t, f.user.ID))
}

func TestVCFImport_UnknownSession(t *testing.T) {
	f := setupImport(t)
	w := doJSON(t, f.router, http.MethodPost, "/import/vcf/confirm", models.ImportConfirmRequest{SessionID: "nope", Actions: addAll(1)})
	assert.GreaterOrEqual(t, w.Code, 400)
	assert.Less(t, w.Code, 500)
}

func uploadCSVAndMap(t *testing.T, f importFixture) (string, []models.ColumnMapping) {
	t.Helper()
	w := doUpload(t, f.router, "/import/upload", "file", "people.CSV", []byte(testCSV))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	upload := decodeJSON[models.ImportUploadResponse](t, w)
	assert.Equal(t, []string{"First Name", "Last Name", "Email"}, upload.Headers)
	assert.Equal(t, 2, upload.RowCount)
	assert.Equal(t, []string{"Ada", "Lovelace", "ada@example.com"}, upload.SampleData[0])
	return upload.SessionID, upload.SuggestedMappings
}

func TestCSVImport_SuccessfulFlow(t *testing.T) {
	f := setupImport(t)
	sessionID, mappings := uploadCSVAndMap(t, f)

	w := doJSON(t, f.router, http.MethodPost, "/import/preview", models.ImportPreviewRequest{SessionID: sessionID, Mappings: mappings})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	preview := decodeJSON[models.ImportPreviewResponse](t, w)
	assert.Equal(t, 2, preview.TotalRows)
	assert.Equal(t, 2, preview.ValidRows)

	w = doJSON(t, f.router, http.MethodPost, "/import/confirm", models.ImportConfirmRequest{SessionID: sessionID, Actions: addAll(2)})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 2, decodeJSON[models.ImportResult](t, w).Created)

	contacts := f.contactsOf(t, f.user.ID)
	require.Len(t, contacts, 2)
	assert.Equal(t, "Ada", contacts[0].Firstname)
	assert.Equal(t, "Babbage", contacts[1].Lastname)
}

func TestCSVImport_ConfirmNeedsAPreview(t *testing.T) {
	f := setupImport(t)
	sessionID, _ := uploadCSVAndMap(t, f)

	w := doJSON(t, f.router, http.MethodPost, "/import/confirm", models.ImportConfirmRequest{SessionID: sessionID, Actions: addAll(2)})

	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	assert.Empty(t, f.contactsOf(t, f.user.ID))
}

func TestCSVImport_SessionOfAnotherUserIsOffLimits(t *testing.T) {
	f := setupImport(t)
	sessionID, mappings := uploadCSVAndMap(t, f)

	w := doJSON(t, f.strangerRouter, http.MethodPost, "/import/preview", models.ImportPreviewRequest{SessionID: sessionID, Mappings: mappings})
	assert.GreaterOrEqual(t, w.Code, 400)
	assert.NotContains(t, w.Body.String(), "Lovelace")

	w = doJSON(t, f.router, http.MethodPost, "/import/preview", models.ImportPreviewRequest{SessionID: sessionID, Mappings: mappings})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = doJSON(t, f.strangerRouter, http.MethodPost, "/import/confirm", models.ImportConfirmRequest{SessionID: sessionID, Actions: addAll(2)})
	assert.GreaterOrEqual(t, w.Code, 400)
	assert.Empty(t, f.contactsOf(t, f.stranger.ID))
	assert.Empty(t, f.contactsOf(t, f.user.ID))
}

func TestImport_RejectsMalformedRequests(t *testing.T) {
	f := setupImport(t)

	assert.Equal(t, http.StatusBadRequest, doJSON(t, f.router, http.MethodPost, "/import/preview", gin.H{"mappings": []any{}}).Code, "session_id is required")
	assert.Equal(t, http.StatusBadRequest, doJSON(t, f.router, http.MethodPost, "/import/confirm", gin.H{"session_id": "x"}).Code, "actions are required")
	assert.Equal(t, http.StatusBadRequest, doJSON(t, f.router, http.MethodPost, "/import/vcf/confirm", gin.H{
		"session_id": "x", "actions": []gin.H{{"row_index": 0, "action": "explode"}},
	}).Code, "unknown action")
}
