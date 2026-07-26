package controllers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"meerkat/config"
	"meerkat/middleware"
	"meerkat/models"
	"meerkat/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// --- shared fixtures / helpers -------------------------------------------------------

// testPNGBase64ImportCtrl is the same 1x1 transparent PNG fixture used across
// the codebase (photostore, services/contact_sync_service_test.go,
// services/import_session_test.go, controllers/export_controller_test.go) --
// small enough to embed inline, decodes as a real image.
const testPNGBase64ImportCtrl = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="

// registerImportRoutes wires all six import handlers onto router exactly the
// way routes.go does, including the cfg closures for the two handlers that
// take *config.Config directly (UploadVCFForImport/ConfirmVCFImport) rather
// than through currentConfig(c).
func registerImportRoutes(router *gin.Engine, cfg *config.Config) {
	router.POST("/contacts/import/upload", UploadCSVForImport)
	router.POST("/contacts/import/preview", middleware.ValidateJSONMiddleware(&models.ImportPreviewRequest{}), PreviewImport)
	router.POST("/contacts/import/confirm", middleware.ValidateJSONMiddleware(&models.ImportConfirmRequest{}), ConfirmImport)
	router.POST("/contacts/import/vcf/upload", func(c *gin.Context) {
		UploadVCFForImport(c, cfg)
	})
	router.POST("/contacts/import/vcf/confirm", middleware.ValidateJSONMiddleware(&models.ImportConfirmRequest{}), func(c *gin.Context) {
		ConfirmVCFImport(c, cfg)
	})
	router.POST("/contacts/import/jscontact/upload", UploadJSContactForImport)
}

// newFileUploadRequest builds a multipart/form-data POST with a single "file"
// part, mirroring how the real frontend/clients call these upload endpoints.
func newFileUploadRequest(t *testing.T, url, filename string, content []byte) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, url, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

// newFileUploadRequestNoFile builds a multipart request with no "file" part
// at all, to exercise the "missing file" branch of each upload handler.
func newFileUploadRequestNoFile(t *testing.T, url string) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.WriteField("note", "no file here"))
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, url, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

// newJSONRequest builds a standard application/json POST request.
func newJSONRequest(t *testing.T, url string, payload interface{}) *http.Request {
	t.Helper()
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// errorEnvelope mirrors errors.ErrorResponse's wire shape. Note that
// apperrors.ErrInvalidInput's top-level Message is always the generic
// "Invalid value for field '<field>'" -- the actual human-readable reason
// (e.g. "File too large...", "File must be a CSV file") is carried in
// Details["reason"] (see errors/errors.go's ErrInvalidInput), so tests that
// want to assert on the specific reason text must look there, not at Message.
type errorEnvelope struct {
	Error struct {
		Code    string                 `json:"code"`
		Message string                 `json:"message"`
		Details map[string]interface{} `json:"details"`
	} `json:"error"`
}

func decodeError(t *testing.T, w *httptest.ResponseRecorder) errorEnvelope {
	t.Helper()
	var env errorEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "response body: %s", w.Body.String())
	return env
}

// errorReason extracts the human-readable "reason" detail from an
// ErrInvalidInput-shaped error response.
func errorReason(env errorEnvelope) string {
	if r, ok := env.Error.Details["reason"].(string); ok {
		return r
	}
	return ""
}

// csvFixture builds a minimal valid CSV upload body with the given data rows
// appended after a fixed three-column header.
func csvFixture(rows ...[3]string) []byte {
	var b strings.Builder
	b.WriteString("First Name,Last Name,Email\n")
	for _, r := range rows {
		b.WriteString(fmt.Sprintf("%s,%s,%s\n", r[0], r[1], r[2]))
	}
	return []byte(b.String())
}

// vcfFixture builds a minimal valid single-contact vCard 4.0 body. If
// photoDataURI is non-empty, an embedded PHOTO property (the
// "PHOTO:data:image/...;base64,..." form vcard4's Import path copies
// verbatim into Card.Media) is included.
func vcfFixture(firstname, lastname, email, photoDataURI string) []byte {
	var b strings.Builder
	b.WriteString("BEGIN:VCARD\r\n")
	b.WriteString("VERSION:4.0\r\n")
	b.WriteString(fmt.Sprintf("FN:%s %s\r\n", firstname, lastname))
	b.WriteString(fmt.Sprintf("N:%s;%s;;;\r\n", lastname, firstname))
	if email != "" {
		b.WriteString(fmt.Sprintf("EMAIL:%s\r\n", email))
	}
	if photoDataURI != "" {
		b.WriteString(fmt.Sprintf("PHOTO:%s\r\n", photoDataURI))
	}
	b.WriteString("END:VCARD\r\n")
	return []byte(b.String())
}

// jsContactFixture builds a minimal valid JSContact Card JSON body whose
// name.components set both given/surname (so ApplyRecordToContact populates
// Contact.Firstname/Lastname and ValidateImportedContact's "First name is
// required" check passes).
func jsContactFixture(uid, given, surname, email string) []byte {
	card := map[string]interface{}{
		"@type":   "Card",
		"version": "1.0",
		"uid":     uid,
		"name": map[string]interface{}{
			"@type": "Name",
			"full":  given + " " + surname,
			"components": []map[string]string{
				{"kind": "given", "value": given},
				{"kind": "surname", "value": surname},
			},
		},
	}
	if email != "" {
		card["emails"] = map[string]interface{}{
			"E1": map[string]interface{}{"@type": "EmailAddress", "address": email},
		}
	}
	raw, _ := json.Marshal(card)
	return raw
}

func decodeTestPNGImportCtrl(t *testing.T) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(testPNGBase64ImportCtrl)
	require.NoError(t, err)
	return data
}

// routerWithoutAuth carries a "db" but deliberately never sets "userID",
// exercising currentUserID's `!ok` branch (helpers.go) that every one of the
// six import handlers checks first.
func routerWithoutAuth(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})
	return r
}

// =====================================================================================
// UploadCSVForImport
// =====================================================================================

func TestUploadCSVForImport_Success(t *testing.T) {
	db, router := setupRouter()
	registerImportRoutes(router, &config.Config{})
	var user models.User
	db.First(&user)
	_ = user

	content := csvFixture([3]string{"Alice", "Smith", "alice@example.com"}, [3]string{"Bob", "Jones", "bob@example.com"})
	req := newFileUploadRequest(t, "/contacts/import/upload", "contacts.csv", content)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp models.ImportUploadResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.NotEmpty(t, resp.SessionID)
	assert.Equal(t, []string{"First Name", "Last Name", "Email"}, resp.Headers)
	assert.Equal(t, 2, resp.RowCount)
	require.Len(t, resp.SampleData, 2)
	assert.Equal(t, []string{"Alice", "Smith", "alice@example.com"}, resp.SampleData[0])

	require.Len(t, resp.SuggestedMappings, 3)
	assert.Equal(t, "firstname", resp.SuggestedMappings[0].ContactField)
	assert.Equal(t, "lastname", resp.SuggestedMappings[1].ContactField)
	assert.Equal(t, "email", resp.SuggestedMappings[2].ContactField)
}

func TestUploadCSVForImport_MissingFile(t *testing.T) {
	_, router := setupRouter()
	registerImportRoutes(router, &config.Config{})

	req := newFileUploadRequestNoFile(t, "/contacts/import/upload")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, "INVALID_INPUT", env.Error.Code)
}

func TestUploadCSVForImport_OversizedFile(t *testing.T) {
	_, router := setupRouter()
	registerImportRoutes(router, &config.Config{})

	big := make([]byte, services.MaxCSVSize+1024)
	for i := range big {
		big[i] = 'a'
	}
	req := newFileUploadRequest(t, "/contacts/import/upload", "big.csv", big)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, "INVALID_INPUT", env.Error.Code)
	assert.Contains(t, errorReason(env), "too large")
}

func TestUploadCSVForImport_WrongExtension(t *testing.T) {
	_, router := setupRouter()
	registerImportRoutes(router, &config.Config{})

	content := csvFixture([3]string{"Alice", "Smith", "alice@example.com"})
	req := newFileUploadRequest(t, "/contacts/import/upload", "contacts.txt", content)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, "INVALID_INPUT", env.Error.Code)
	assert.Contains(t, errorReason(env), "CSV")
}

func TestUploadCSVForImport_MalformedCSV_Returns400NotPanic(t *testing.T) {
	_, router := setupRouter()
	registerImportRoutes(router, &config.Config{})

	// Headers-only, no data rows: a genuine ParseCSV error ("CSV file has no
	// data rows"), proving parse failures surface as a clean 400, not a 500
	// or panic.
	content := []byte("First Name,Last Name,Email\n")
	req := newFileUploadRequest(t, "/contacts/import/upload", "empty.csv", content)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, "INVALID_INPUT", env.Error.Code)
}

// =====================================================================================
// UploadVCFForImport
// =====================================================================================

func TestUploadVCFForImport_Success(t *testing.T) {
	db, router := setupRouter()
	registerImportRoutes(router, &config.Config{})
	var user models.User
	db.First(&user)

	content := vcfFixture("Vera", "Card", "vera@example.com", "")
	req := newFileUploadRequest(t, "/contacts/import/vcf/upload", "contacts.vcf", content)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp models.ImportPreviewResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.NotEmpty(t, resp.SessionID)
	assert.Equal(t, 1, resp.TotalRows)
	assert.Equal(t, 1, resp.ValidRows)
	assert.Equal(t, 0, resp.DuplicateCount)
	assert.Equal(t, 0, resp.ErrorCount)
}

func TestUploadVCFForImport_DuplicateDetected(t *testing.T) {
	db, router := setupRouter()
	registerImportRoutes(router, &config.Config{})
	var user models.User
	db.First(&user)
	require.NoError(t, db.Create(&models.Contact{UserID: user.ID, Firstname: "Vera", Lastname: "Card", Email: "vera-dup@example.com"}).Error)

	content := vcfFixture("Vera", "Card", "vera-dup@example.com", "")
	req := newFileUploadRequest(t, "/contacts/import/vcf/upload", "contacts.vcf", content)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp models.ImportPreviewResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.Equal(t, 1, resp.DuplicateCount)
	require.Len(t, resp.Rows, 1)
	require.NotNil(t, resp.Rows[0].DuplicateMatch)
	assert.Equal(t, "email", resp.Rows[0].DuplicateMatch.MatchReason)
}

func TestUploadVCFForImport_MissingFile(t *testing.T) {
	_, router := setupRouter()
	registerImportRoutes(router, &config.Config{})

	req := newFileUploadRequestNoFile(t, "/contacts/import/vcf/upload")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "INVALID_INPUT", decodeError(t, w).Error.Code)
}

func TestUploadVCFForImport_OversizedFile(t *testing.T) {
	_, router := setupRouter()
	registerImportRoutes(router, &config.Config{})

	big := make([]byte, services.MaxVCFSize+1024)
	for i := range big {
		big[i] = 'a'
	}
	req := newFileUploadRequest(t, "/contacts/import/vcf/upload", "big.vcf", big)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, "INVALID_INPUT", env.Error.Code)
	assert.Contains(t, errorReason(env), "too large")
}

func TestUploadVCFForImport_WrongExtension(t *testing.T) {
	_, router := setupRouter()
	registerImportRoutes(router, &config.Config{})

	content := vcfFixture("Vera", "Card", "vera@example.com", "")
	req := newFileUploadRequest(t, "/contacts/import/vcf/upload", "contacts.txt", content)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, "INVALID_INPUT", env.Error.Code)
	assert.Contains(t, errorReason(env), "VCF")
}

func TestUploadVCFForImport_Malformed_Returns400NotPanic(t *testing.T) {
	_, router := setupRouter()
	registerImportRoutes(router, &config.Config{})

	content := []byte("this is not a vCard at all\nno BEGIN:VCARD block here\n")
	req := newFileUploadRequest(t, "/contacts/import/vcf/upload", "bad.vcf", content)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, "INVALID_INPUT", env.Error.Code)
}

// =====================================================================================
// UploadJSContactForImport
// =====================================================================================

func TestUploadJSContactForImport_Success(t *testing.T) {
	db, router := setupRouter()
	registerImportRoutes(router, &config.Config{})
	var user models.User
	db.First(&user)

	content := jsContactFixture("jscontact-ctrl-success", "Jamie", "Fixture", "jamie@example.com")
	req := newFileUploadRequest(t, "/contacts/import/jscontact/upload", "contacts.json", content)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp models.ImportPreviewResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.NotEmpty(t, resp.SessionID)
	assert.Equal(t, 1, resp.TotalRows)
	assert.Equal(t, 1, resp.ValidRows)
	require.Len(t, resp.Rows, 1)
	assert.Equal(t, "Jamie", resp.Rows[0].ParsedContact["firstname"])
}

func TestUploadJSContactForImport_WrongExtension(t *testing.T) {
	_, router := setupRouter()
	registerImportRoutes(router, &config.Config{})

	content := jsContactFixture("jscontact-ctrl-wrongext", "Jamie", "Fixture", "jamie@example.com")
	req := newFileUploadRequest(t, "/contacts/import/jscontact/upload", "contacts.vcf", content)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, "INVALID_INPUT", env.Error.Code)
	assert.Contains(t, errorReason(env), "JSON")
}

func TestUploadJSContactForImport_MalformedJSON_Returns400NotPanic(t *testing.T) {
	_, router := setupRouter()
	registerImportRoutes(router, &config.Config{})

	content := []byte("{ this is not valid json at all !!! ")
	req := newFileUploadRequest(t, "/contacts/import/jscontact/upload", "bad.json", content)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, "INVALID_INPUT", env.Error.Code)
}

// TestUploadJSContactForImport_ConfirmViaVCFRoute_CreatesRealContact proves
// UploadJSContactForImport's session -- created via CreateVCFSession exactly
// like UploadVCFForImport's, per the handler's own doc comment -- really is
// format-agnostic: confirming it through /contacts/import/vcf/confirm (the
// only confirm route JSContact imports are wired to, see routes.go) must
// produce a genuine persisted Contact row, exercising ConfirmVCF's real
// tx.Create/photo pipeline end-to-end for a JSContact-sourced session.
func TestUploadJSContactForImport_ConfirmViaVCFRoute_CreatesRealContact(t *testing.T) {
	db, router := setupRouter()
	cfg := &config.Config{ProfilePhotoDir: t.TempDir()}
	registerImportRoutes(router, cfg)
	var user models.User
	db.First(&user)

	content := jsContactFixture("jscontact-ctrl-confirm", "Jordan", "Card", "jordan@example.com")
	uploadReq := newFileUploadRequest(t, "/contacts/import/jscontact/upload", "contacts.json", content)
	uploadW := httptest.NewRecorder()
	router.ServeHTTP(uploadW, uploadReq)
	require.Equal(t, http.StatusOK, uploadW.Code, uploadW.Body.String())

	var uploadResp models.ImportPreviewResponse
	require.NoError(t, json.Unmarshal(uploadW.Body.Bytes(), &uploadResp))
	require.Equal(t, 1, uploadResp.ValidRows)

	confirmReq := newJSONRequest(t, "/contacts/import/vcf/confirm", models.ImportConfirmRequest{
		SessionID: uploadResp.SessionID,
		Actions:   []models.RowImportAction{{RowIndex: 0, Action: "add"}},
	})
	confirmW := httptest.NewRecorder()
	router.ServeHTTP(confirmW, confirmReq)
	require.Equal(t, http.StatusOK, confirmW.Code, confirmW.Body.String())

	var result models.ImportResult
	require.NoError(t, json.Unmarshal(confirmW.Body.Bytes(), &result))
	assert.Equal(t, 1, result.Created)

	var persisted models.Contact
	require.NoError(t, db.Where("email = ?", "jordan@example.com").First(&persisted).Error)
	assert.Equal(t, "Jordan", persisted.Firstname)
	assert.Equal(t, user.ID, persisted.UserID)
}

// =====================================================================================
// PreviewImport
// =====================================================================================

func TestPreviewImport_Success(t *testing.T) {
	db, router := setupRouter()
	registerImportRoutes(router, &config.Config{})
	var user models.User
	db.First(&user)

	content := csvFixture([3]string{"Alice", "Smith", "alice-preview@example.com"})
	uploadReq := newFileUploadRequest(t, "/contacts/import/upload", "contacts.csv", content)
	uploadW := httptest.NewRecorder()
	router.ServeHTTP(uploadW, uploadReq)
	require.Equal(t, http.StatusOK, uploadW.Code)

	var uploadResp models.ImportUploadResponse
	require.NoError(t, json.Unmarshal(uploadW.Body.Bytes(), &uploadResp))

	previewReq := newJSONRequest(t, "/contacts/import/preview", models.ImportPreviewRequest{
		SessionID: uploadResp.SessionID,
		Mappings: []models.ColumnMapping{
			{CSVColumn: "First Name", ContactField: "firstname"},
			{CSVColumn: "Last Name", ContactField: "lastname"},
			{CSVColumn: "Email", ContactField: "email"},
		},
	})
	previewW := httptest.NewRecorder()
	router.ServeHTTP(previewW, previewReq)

	require.Equal(t, http.StatusOK, previewW.Code, previewW.Body.String())
	var resp models.ImportPreviewResponse
	require.NoError(t, json.Unmarshal(previewW.Body.Bytes(), &resp))
	assert.Equal(t, uploadResp.SessionID, resp.SessionID)
	assert.Equal(t, 1, resp.TotalRows)
	assert.Equal(t, 1, resp.ValidRows)
}

func TestPreviewImport_UnknownSessionID_NotFound(t *testing.T) {
	_, router := setupRouter()
	registerImportRoutes(router, &config.Config{})

	req := newJSONRequest(t, "/contacts/import/preview", models.ImportPreviewRequest{
		SessionID: "does-not-exist-at-all",
		Mappings:  []models.ColumnMapping{{CSVColumn: "First Name", ContactField: "firstname"}},
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, "NOT_FOUND", env.Error.Code)
}

func TestPreviewImport_WrongUser_Unauthorized(t *testing.T) {
	db, router := setupRouter()
	registerImportRoutes(router, &config.Config{})
	var user models.User
	db.First(&user)
	otherUser := models.User{Username: "other-preview", Password: "password456", Email: "other-preview@example.com"}
	require.NoError(t, db.Create(&otherUser).Error)

	content := csvFixture([3]string{"Alice", "Smith", "alice-wronguser@example.com"})
	uploadReq := newFileUploadRequest(t, "/contacts/import/upload", "contacts.csv", content)
	uploadW := httptest.NewRecorder()
	router.ServeHTTP(uploadW, uploadReq)
	require.Equal(t, http.StatusOK, uploadW.Code)

	var uploadResp models.ImportUploadResponse
	require.NoError(t, json.Unmarshal(uploadW.Body.Bytes(), &uploadResp))

	// Second router, same db, but authenticated as the other user.
	otherRouter := routerForUser(db, otherUser.ID)
	registerImportRoutes(otherRouter, &config.Config{})

	previewReq := newJSONRequest(t, "/contacts/import/preview", models.ImportPreviewRequest{
		SessionID: uploadResp.SessionID,
		Mappings:  []models.ColumnMapping{{CSVColumn: "First Name", ContactField: "firstname"}},
	})
	previewW := httptest.NewRecorder()
	otherRouter.ServeHTTP(previewW, previewReq)

	assert.Equal(t, http.StatusUnauthorized, previewW.Code)
	env := decodeError(t, previewW)
	assert.Equal(t, "UNAUTHORIZED", env.Error.Code)
}

// =====================================================================================
// ConfirmImport (CSV)
// =====================================================================================

func TestConfirmImport_FullCSVHappyPath_PersistsRealRow(t *testing.T) {
	db, router := setupRouter()
	registerImportRoutes(router, &config.Config{})
	var user models.User
	db.First(&user)

	content := csvFixture([3]string{"Casey", "Confirmed", "casey-confirm@example.com"})
	uploadReq := newFileUploadRequest(t, "/contacts/import/upload", "contacts.csv", content)
	uploadW := httptest.NewRecorder()
	router.ServeHTTP(uploadW, uploadReq)
	require.Equal(t, http.StatusOK, uploadW.Code)
	var uploadResp models.ImportUploadResponse
	require.NoError(t, json.Unmarshal(uploadW.Body.Bytes(), &uploadResp))

	previewReq := newJSONRequest(t, "/contacts/import/preview", models.ImportPreviewRequest{
		SessionID: uploadResp.SessionID,
		Mappings: []models.ColumnMapping{
			{CSVColumn: "First Name", ContactField: "firstname"},
			{CSVColumn: "Last Name", ContactField: "lastname"},
			{CSVColumn: "Email", ContactField: "email"},
		},
	})
	previewW := httptest.NewRecorder()
	router.ServeHTTP(previewW, previewReq)
	require.Equal(t, http.StatusOK, previewW.Code)

	confirmReq := newJSONRequest(t, "/contacts/import/confirm", models.ImportConfirmRequest{
		SessionID: uploadResp.SessionID,
		Actions:   []models.RowImportAction{{RowIndex: 0, Action: "add"}},
	})
	confirmW := httptest.NewRecorder()
	router.ServeHTTP(confirmW, confirmReq)
	require.Equal(t, http.StatusOK, confirmW.Code, confirmW.Body.String())

	var result models.ImportResult
	require.NoError(t, json.Unmarshal(confirmW.Body.Bytes(), &result))
	assert.Equal(t, 1, result.Created)
	assert.Empty(t, result.Errors)

	var persisted models.Contact
	require.NoError(t, db.Where("email = ?", "casey-confirm@example.com").First(&persisted).Error)
	assert.Equal(t, "Casey", persisted.Firstname)
	assert.Equal(t, user.ID, persisted.UserID)
}

func TestConfirmImport_NoPriorPreview_InvalidInput(t *testing.T) {
	db, router := setupRouter()
	registerImportRoutes(router, &config.Config{})
	var user models.User
	db.First(&user)

	content := csvFixture([3]string{"NoPreview", "Person", "nopreview@example.com"})
	uploadReq := newFileUploadRequest(t, "/contacts/import/upload", "contacts.csv", content)
	uploadW := httptest.NewRecorder()
	router.ServeHTTP(uploadW, uploadReq)
	require.Equal(t, http.StatusOK, uploadW.Code)
	var uploadResp models.ImportUploadResponse
	require.NoError(t, json.Unmarshal(uploadW.Body.Bytes(), &uploadResp))

	// Confirm without ever calling /preview first.
	confirmReq := newJSONRequest(t, "/contacts/import/confirm", models.ImportConfirmRequest{
		SessionID: uploadResp.SessionID,
		Actions:   []models.RowImportAction{{RowIndex: 0, Action: "add"}},
	})
	confirmW := httptest.NewRecorder()
	router.ServeHTTP(confirmW, confirmReq)

	assert.Equal(t, http.StatusBadRequest, confirmW.Code)
	env := decodeError(t, confirmW)
	assert.Equal(t, "INVALID_INPUT", env.Error.Code)

	var count int64
	require.NoError(t, db.Model(&models.Contact{}).Where("user_id = ?", user.ID).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

// =====================================================================================
// ConfirmVCFImport (VCF, including embedded photo)
// =====================================================================================

func TestConfirmVCFImport_FullHappyPath_WithEmbeddedPhoto(t *testing.T) {
	db, router := setupRouter()
	cfg := &config.Config{ProfilePhotoDir: t.TempDir()}
	registerImportRoutes(router, cfg)
	var user models.User
	db.First(&user)

	photoBytes := decodeTestPNGImportCtrl(t)
	photoDataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(photoBytes)
	content := vcfFixture("Photo", "Haver", "photo-haver@example.com", photoDataURI)

	uploadReq := newFileUploadRequest(t, "/contacts/import/vcf/upload", "contacts.vcf", content)
	uploadW := httptest.NewRecorder()
	router.ServeHTTP(uploadW, uploadReq)
	require.Equal(t, http.StatusOK, uploadW.Code, uploadW.Body.String())

	var uploadResp models.ImportPreviewResponse
	require.NoError(t, json.Unmarshal(uploadW.Body.Bytes(), &uploadResp))
	require.Equal(t, 1, uploadResp.ValidRows)

	confirmReq := newJSONRequest(t, "/contacts/import/vcf/confirm", models.ImportConfirmRequest{
		SessionID: uploadResp.SessionID,
		Actions:   []models.RowImportAction{{RowIndex: 0, Action: "add"}},
	})
	confirmW := httptest.NewRecorder()
	router.ServeHTTP(confirmW, confirmReq)
	require.Equal(t, http.StatusOK, confirmW.Code, confirmW.Body.String())

	var result models.ImportResult
	require.NoError(t, json.Unmarshal(confirmW.Body.Bytes(), &result))
	assert.Equal(t, 1, result.Created)
	assert.Empty(t, result.Errors)

	var persisted models.Contact
	require.NoError(t, db.Where("email = ?", "photo-haver@example.com").First(&persisted).Error)
	assert.Equal(t, "Photo", persisted.Firstname)
	assert.Equal(t, user.ID, persisted.UserID)

	// The photo must have actually landed on disk (via cfg.ProfilePhotoDir,
	// wired through the closure in registerImportRoutes exactly like
	// routes.go's own `func(c *gin.Context) { controllers.ConfirmVCFImport(c, cfg) }`)
	// and the thumbnail must be populated.
	assert.NotEmpty(t, persisted.Photo, "photo path should be populated via photostore.SaveContactPhoto")
	assert.Contains(t, persisted.Photo, "_photo.jpg")
	assert.NotEmpty(t, persisted.PhotoThumbnail)
	assert.Contains(t, persisted.PhotoThumbnail, "data:image/jpeg;base64,")

	// Same GORM/AfterSave regression class as TC-1.1 (import_session_test.go's
	// TestConfirmVCF_Add_EmbeddedPhoto_PersistsPhotoAndValidETag): the ETag
	// must be well-formed and keyed to the real row's ID, proving the deferred
	// photo write went through db.First+Model(&loadedRow) rather than a bulk
	// Where(...).Updates(map) call against a zero-value receiver.
	require.NotEmpty(t, persisted.ETag)
	expectedPrefix := fmt.Sprintf("e-%d-", persisted.ID)
	assert.True(t, strings.HasPrefix(persisted.ETag, expectedPrefix),
		"ETag %q should start with %q", persisted.ETag, expectedPrefix)
}

// =====================================================================================
// Cross-cutting: import-type confusion between the CSV and VCF confirm routes
// =====================================================================================

// TestConfirmVCFImport_CSVSessionRejected proves that a CSV-created session
// ID handed to the VCF-only confirm route is rejected cleanly (400
// INVALID_INPUT), not a panic from a nil vcfContacts slice access --
// ConfirmVCF explicitly checks sessionData.importType != "vcf" before
// touching sessionData.vcfContacts (import_session.go).
func TestConfirmVCFImport_CSVSessionRejected(t *testing.T) {
	db, router := setupRouter()
	cfg := &config.Config{ProfilePhotoDir: t.TempDir()}
	registerImportRoutes(router, cfg)
	var user models.User
	db.First(&user)
	_ = user

	content := csvFixture([3]string{"Cross", "Format", "crossformat@example.com"})
	uploadReq := newFileUploadRequest(t, "/contacts/import/upload", "contacts.csv", content)
	uploadW := httptest.NewRecorder()
	router.ServeHTTP(uploadW, uploadReq)
	require.Equal(t, http.StatusOK, uploadW.Code)
	var uploadResp models.ImportUploadResponse
	require.NoError(t, json.Unmarshal(uploadW.Body.Bytes(), &uploadResp))

	previewReq := newJSONRequest(t, "/contacts/import/preview", models.ImportPreviewRequest{
		SessionID: uploadResp.SessionID,
		Mappings: []models.ColumnMapping{
			{CSVColumn: "First Name", ContactField: "firstname"},
			{CSVColumn: "Last Name", ContactField: "lastname"},
			{CSVColumn: "Email", ContactField: "email"},
		},
	})
	previewW := httptest.NewRecorder()
	router.ServeHTTP(previewW, previewReq)
	require.Equal(t, http.StatusOK, previewW.Code)

	// Hand the CSV session's ID to the VCF-only confirm route.
	confirmReq := newJSONRequest(t, "/contacts/import/vcf/confirm", models.ImportConfirmRequest{
		SessionID: uploadResp.SessionID,
		Actions:   []models.RowImportAction{{RowIndex: 0, Action: "add"}},
	})
	confirmW := httptest.NewRecorder()
	router.ServeHTTP(confirmW, confirmReq)

	assert.Equal(t, http.StatusBadRequest, confirmW.Code, confirmW.Body.String())
	env := decodeError(t, confirmW)
	assert.Equal(t, "INVALID_INPUT", env.Error.Code)

	var count int64
	require.NoError(t, db.Model(&models.Contact{}).Count(&count).Error)
	assert.Equal(t, int64(0), count, "the mis-routed confirm must not have created anything")
}

// TestConfirmImport_VCFSessionAccepted_NoPanic documents the real (verified)
// behavior of the generic /contacts/import/confirm route when handed a
// VCF-created session ID: unlike ConfirmVCFImport, Confirm() (import_session.go)
// does not reject on importType -- it branches on an internal isVCFImport
// flag and reads from sessionData.vcfContacts instead of sessionData.csvContacts,
// so the request completes successfully (just without VCF's photo-processing
// step, which only ConfirmVCF performs) rather than panicking on a nil slice
// access. This nails down actual behavior rather than assuming symmetry with
// the CSV-into-VCF-route case above, which the code does reject explicitly.
func TestConfirmImport_VCFSessionAccepted_NoPanic(t *testing.T) {
	db, router := setupRouter()
	cfg := &config.Config{ProfilePhotoDir: t.TempDir()}
	registerImportRoutes(router, cfg)
	var user models.User
	db.First(&user)
	_ = user

	content := vcfFixture("Vice", "Versa", "vice-versa@example.com", "")
	uploadReq := newFileUploadRequest(t, "/contacts/import/vcf/upload", "contacts.vcf", content)
	uploadW := httptest.NewRecorder()
	router.ServeHTTP(uploadW, uploadReq)
	require.Equal(t, http.StatusOK, uploadW.Code)
	var uploadResp models.ImportPreviewResponse
	require.NoError(t, json.Unmarshal(uploadW.Body.Bytes(), &uploadResp))

	// Hand the VCF session's ID to the generic (CSV/shared) confirm route.
	confirmReq := newJSONRequest(t, "/contacts/import/confirm", models.ImportConfirmRequest{
		SessionID: uploadResp.SessionID,
		Actions:   []models.RowImportAction{{RowIndex: 0, Action: "add"}},
	})
	confirmW := httptest.NewRecorder()
	require.NotPanics(t, func() {
		router.ServeHTTP(confirmW, confirmReq)
	})

	require.Equal(t, http.StatusOK, confirmW.Code, confirmW.Body.String())
	var result models.ImportResult
	require.NoError(t, json.Unmarshal(confirmW.Body.Bytes(), &result))
	assert.Equal(t, 1, result.Created)

	var persisted models.Contact
	require.NoError(t, db.Where("email = ?", "vice-versa@example.com").First(&persisted).Error)
	assert.Equal(t, "Vice", persisted.Firstname)
}

// =====================================================================================
// currentUserID `!ok` branch -- every handler checks auth first and returns
// immediately (currentUserID itself already wrote the 401 response).
// =====================================================================================

func TestUploadCSVForImport_NoAuth_Unauthorized(t *testing.T) {
	db, _ := setupRouter()
	router := routerWithoutAuth(db)
	registerImportRoutes(router, &config.Config{})

	req := newFileUploadRequest(t, "/contacts/import/upload", "contacts.csv", csvFixture([3]string{"A", "B", "a@example.com"}))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "UNAUTHORIZED", decodeError(t, w).Error.Code)
}

func TestUploadVCFForImport_NoAuth_Unauthorized(t *testing.T) {
	db, _ := setupRouter()
	router := routerWithoutAuth(db)
	registerImportRoutes(router, &config.Config{})

	req := newFileUploadRequest(t, "/contacts/import/vcf/upload", "contacts.vcf", vcfFixture("A", "B", "a@example.com", ""))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "UNAUTHORIZED", decodeError(t, w).Error.Code)
}

func TestUploadJSContactForImport_NoAuth_Unauthorized(t *testing.T) {
	db, _ := setupRouter()
	router := routerWithoutAuth(db)
	registerImportRoutes(router, &config.Config{})

	req := newFileUploadRequest(t, "/contacts/import/jscontact/upload", "contacts.json", jsContactFixture("noauth", "A", "B", "a@example.com"))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "UNAUTHORIZED", decodeError(t, w).Error.Code)
}

func TestPreviewImport_NoAuth_Unauthorized(t *testing.T) {
	db, _ := setupRouter()
	router := routerWithoutAuth(db)
	registerImportRoutes(router, &config.Config{})

	req := newJSONRequest(t, "/contacts/import/preview", models.ImportPreviewRequest{
		SessionID: "irrelevant",
		Mappings:  []models.ColumnMapping{{CSVColumn: "First Name", ContactField: "firstname"}},
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "UNAUTHORIZED", decodeError(t, w).Error.Code)
}

func TestConfirmImport_NoAuth_Unauthorized(t *testing.T) {
	db, _ := setupRouter()
	router := routerWithoutAuth(db)
	registerImportRoutes(router, &config.Config{})

	req := newJSONRequest(t, "/contacts/import/confirm", models.ImportConfirmRequest{
		SessionID: "irrelevant",
		Actions:   []models.RowImportAction{{RowIndex: 0, Action: "add"}},
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "UNAUTHORIZED", decodeError(t, w).Error.Code)
}

func TestConfirmVCFImport_NoAuth_Unauthorized(t *testing.T) {
	db, _ := setupRouter()
	router := routerWithoutAuth(db)
	registerImportRoutes(router, &config.Config{})

	req := newJSONRequest(t, "/contacts/import/vcf/confirm", models.ImportConfirmRequest{
		SessionID: "irrelevant",
		Actions:   []models.RowImportAction{{RowIndex: 0, Action: "add"}},
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "UNAUTHORIZED", decodeError(t, w).Error.Code)
}

// =====================================================================================
// UploadJSContactForImport: missing-file / oversized-file (mirrors the CSV
// and VCF upload tests above; not explicitly called out by name in the WP but
// cheap to add and closes the remaining branch coverage on this handler).
// =====================================================================================

func TestUploadJSContactForImport_MissingFile(t *testing.T) {
	_, router := setupRouter()
	registerImportRoutes(router, &config.Config{})

	req := newFileUploadRequestNoFile(t, "/contacts/import/jscontact/upload")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "INVALID_INPUT", decodeError(t, w).Error.Code)
}

func TestUploadJSContactForImport_OversizedFile(t *testing.T) {
	_, router := setupRouter()
	registerImportRoutes(router, &config.Config{})

	big := make([]byte, services.MaxVCFSize+1024)
	for i := range big {
		big[i] = 'a'
	}
	req := newFileUploadRequest(t, "/contacts/import/jscontact/upload", "big.json", big)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, "INVALID_INPUT", env.Error.Code)
	assert.Contains(t, errorReason(env), "too large")
}
