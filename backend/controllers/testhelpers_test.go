package controllers

import (
	"bytes"
	"encoding/json"
	"io"
	"meerkat/config"
	"meerkat/models"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const testJWTSecret = "test-jwt-secret-key-with-32-chars!!"

// newTestDB returns an isolated in-memory database with every table the handlers under test touch.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	require.NoError(t, db.AutoMigrate(
		&models.User{}, &models.Contact{}, &models.Activity{}, &models.Note{}, &models.Relationship{},
		&models.Reminder{}, &models.ReminderCompletion{}, &models.EmploymentHistory{}, &models.ApiToken{},
		&models.Webhook{}, &models.WebhookDelivery{}, &models.JobExecution{},
		&models.CalendarSubscription{}, &models.CalendarEventLink{},
		&models.CardDAVConnection{}, &models.CardDAVContactLink{}, &models.CardDAVSync{},
		&models.ContactDuplicateDismissal{},
	))
	return db
}

func newTestUser(t *testing.T, db *gorm.DB, name string, isAdmin bool) models.User {
	t.Helper()
	user := models.User{Username: name, Email: name + "@example.com", Password: "x", IsAdmin: isAdmin}
	require.NoError(t, db.Create(&user).Error)
	return user
}

func newTestContact(t *testing.T, db *gorm.DB, userID uint, firstname string) models.Contact {
	t.Helper()
	contact := models.Contact{UserID: userID, Firstname: firstname}
	require.NoError(t, db.Create(&contact).Error)
	return contact
}

// routerAs builds a router authenticated as userID, wired the way main.go wires db and cfg.
func routerAs(db *gorm.DB, userID uint, cfg config.Config) *gin.Engine {
	r := routerForUser(db, userID)
	r.Use(func(c *gin.Context) {
		c.Set("cfg", cfg)
		c.Next()
	})
	return r
}

func doRequest(router http.Handler, method, path string, body io.Reader, contentType string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func doJSON(t *testing.T, router http.Handler, method, path string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(payload)
	require.NoError(t, err)
	return doRequest(router, method, path, bytes.NewReader(data), "application/json")
}

func doUpload(t *testing.T, router http.Handler, path, field, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(field, filename)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return doRequest(router, http.MethodPost, path, &body, writer.FormDataContentType())
}

func decodeJSON[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out), w.Body.String())
	return out
}
