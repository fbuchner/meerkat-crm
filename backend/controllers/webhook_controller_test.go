package controllers

import (
	"bytes"
	"encoding/json"
	"meerkat/models"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func seedWebhook(db *gorm.DB, userID uint, url string) models.Webhook {
	wh := models.Webhook{
		UserID:   userID,
		Name:     "Test Hook",
		URL:      url,
		Events:   []string{"contact.created"},
		Secret:   "testsecret",
		IsActive: true,
	}
	db.Create(&wh)
	return wh
}

func routerForUser(db *gorm.DB, userID uint) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("userID", userID)
		c.Next()
	})
	return r
}

func TestListWebhooks(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	router := routerForUser(db, user.ID)
	router.GET("/webhooks", ListWebhooks)

	seedWebhook(db, user.ID, "https://example.com/a")
	seedWebhook(db, user.ID, "https://example.com/b")

	req, _ := http.NewRequest("GET", "/webhooks", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	assert.Len(t, body["webhooks"], 2)
}

func TestCreateWebhook(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	router := routerForUser(db, user.ID)
	router.POST("/webhooks", withValidated(func() any { return &models.WebhookInput{} }), CreateWebhook)

	input := models.WebhookInput{
		Name:     "My Hook",
		URL:      "https://example.com/hook",
		Events:   []string{"contact.created", "note.created"},
		IsActive: true,
	}
	body, _ := json.Marshal(input)

	req, _ := http.NewRequest("POST", "/webhooks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "My Hook", resp["name"])
	assert.Equal(t, "https://example.com/hook", resp["url"])
	assert.Len(t, resp["events"], 2)
	assert.NotNil(t, resp["id"])

	var count int64
	db.Model(&models.Webhook{}).Where("user_id = ?", user.ID).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestCreateWebhookLimit(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	router := routerForUser(db, user.ID)
	router.POST("/webhooks", withValidated(func() any { return &models.WebhookInput{} }), CreateWebhook)

	for i := 0; i < maxWebhooksPerUser; i++ {
		db.Create(&models.Webhook{
			UserID:   user.ID,
			Name:     "Hook " + strconv.Itoa(i),
			URL:      "https://example.com/" + strconv.Itoa(i),
			Events:   []string{"contact.created"},
			Secret:   "secret",
			IsActive: true,
		})
	}

	input := models.WebhookInput{Name: "One Too Many", URL: "https://example.com/extra", Events: []string{"contact.created"}}
	body, _ := json.Marshal(input)
	req, _ := http.NewRequest("POST", "/webhooks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestGetWebhook(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	router := routerForUser(db, user.ID)
	router.GET("/webhooks/:id", GetWebhook)

	wh := seedWebhook(db, user.ID, "https://example.com/hook")

	req, _ := http.NewRequest("GET", "/webhooks/"+strconv.Itoa(int(wh.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp models.WebhookResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, wh.ID, resp.ID)
	assert.Equal(t, wh.Name, resp.Name)
}

func TestGetWebhookNotFound(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	router := routerForUser(db, user.ID)
	router.GET("/webhooks/:id", GetWebhook)

	req, _ := http.NewRequest("GET", "/webhooks/99999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateWebhook(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	router := routerForUser(db, user.ID)
	router.PUT("/webhooks/:id", withValidated(func() any { return &models.WebhookInput{} }), UpdateWebhook)

	wh := seedWebhook(db, user.ID, "https://example.com/hook")

	update := models.WebhookInput{
		Name:     "Renamed Hook",
		URL:      "https://example.com/new",
		Events:   []string{"activity.created"},
		IsActive: false,
	}
	body, _ := json.Marshal(update)

	req, _ := http.NewRequest("PUT", "/webhooks/"+strconv.Itoa(int(wh.ID)), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp models.WebhookResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Renamed Hook", resp.Name)
	assert.Equal(t, "https://example.com/new", resp.URL)
	assert.False(t, resp.IsActive)
}

func TestDeleteWebhook(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	router := routerForUser(db, user.ID)
	router.DELETE("/webhooks/:id", DeleteWebhook)

	wh := seedWebhook(db, user.ID, "https://example.com/hook")

	req, _ := http.NewRequest("DELETE", "/webhooks/"+strconv.Itoa(int(wh.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var deleted models.Webhook
	result := db.Unscoped().First(&deleted, wh.ID)
	assert.NoError(t, result.Error)
	assert.NotNil(t, deleted.DeletedAt)
}

func TestTestWebhook(t *testing.T) {
	received := make(chan struct{}, 1)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.NotEmpty(t, r.Header.Get("X-Webhook-Signature"))
		assert.Equal(t, "test", r.Header.Get("X-Meerkat-Event"))
		received <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	router := routerForUser(db, user.ID)
	router.POST("/webhooks/:id/test", TestWebhook)

	wh := seedWebhook(db, user.ID, target.URL)

	req, _ := http.NewRequest("POST", "/webhooks/"+strconv.Itoa(int(wh.ID))+"/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	assert.NotNil(t, body["delivery"])
	assert.Len(t, received, 1, "target server should have received exactly one request")
}

func TestGetWebhookDeliveries(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	router := routerForUser(db, user.ID)
	router.GET("/webhooks/:id/deliveries", GetWebhookDeliveries)

	wh := seedWebhook(db, user.ID, "https://example.com/hook")
	code := 200
	db.Create(&models.WebhookDelivery{WebhookID: wh.ID, EventType: "contact.created", Payload: "{}", StatusCode: &code, Attempts: 1})
	db.Create(&models.WebhookDelivery{WebhookID: wh.ID, EventType: "note.created", Payload: "{}", StatusCode: &code, Attempts: 1})

	req, _ := http.NewRequest("GET", "/webhooks/"+strconv.Itoa(int(wh.ID))+"/deliveries", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	assert.Len(t, body["deliveries"], 2)
}

func TestWebhookUserIsolation(t *testing.T) {
	db, _ := setupRouter()
	var user1 models.User
	db.First(&user1)

	user2 := models.User{Username: "other", Password: "pass", Email: "other@example.com"}
	db.Create(&user2)

	wh := seedWebhook(db, user1.ID, "https://example.com/hook")

	router := routerForUser(db, user2.ID)
	router.GET("/webhooks/:id", GetWebhook)
	router.PUT("/webhooks/:id", withValidated(func() any { return &models.WebhookInput{} }), UpdateWebhook)
	router.DELETE("/webhooks/:id", DeleteWebhook)

	id := strconv.Itoa(int(wh.ID))
	for _, tc := range []struct {
		method string
		path   string
		body   []byte
	}{
		{"GET", "/webhooks/" + id, nil},
		{"PUT", "/webhooks/" + id, func() []byte { b, _ := json.Marshal(models.WebhookInput{Name: "x", URL: "https://x.com", Events: []string{"contact.created"}}); return b }()},
		{"DELETE", "/webhooks/" + id, nil},
	} {
		req, _ := http.NewRequest(tc.method, tc.path, bytes.NewBuffer(tc.body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code, "method %s should return 404 for wrong user", tc.method)
	}
}
