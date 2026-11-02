package services

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"meerkat/config"
	"meerkat/models"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupWebhookTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.WebhookDelivery{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func TestDeliverWebhook_GuardBlocksLoopbackAtDialTime(t *testing.T) {
	hit := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true }))
	defer target.Close()

	d := deliverWebhook(setupWebhookTestDB(t), config.Config{WebhookBlockPrivateURLs: true},
		models.Webhook{URL: target.URL, Secret: "s"}, "test", []byte("{}"), 1)

	assert.False(t, hit)
	if assert.NotNil(t, d.Error) {
		assert.Equal(t, webhookErrPrivateBlocked, *d.Error)
	}
	assert.Nil(t, d.NextRetryAt)
}

func TestGuardedDeliveryClient_RefusesLoopback(t *testing.T) {
	resp, err := guardedDeliveryClient.Get("http://127.0.0.1:1/")
	if resp != nil {
		_ = resp.Body.Close()
	}
	assert.ErrorIs(t, err, errWebhookPrivateAddress)
}

func TestDeliverWebhook_HidesTransportErrorDetail(t *testing.T) {
	target := httptest.NewServer(http.NotFoundHandler())
	url := target.URL
	target.Close() // nothing listens on this port any more

	d := deliverWebhook(setupWebhookTestDB(t), config.Config{},
		models.Webhook{URL: url, Secret: "s"}, "test", []byte("{}"), 1)

	if assert.NotNil(t, d.Error) {
		assert.Equal(t, webhookErrDeliveryFailed, *d.Error)
		assert.False(t, strings.Contains(*d.Error, "refused"))
	}
	assert.NotNil(t, d.NextRetryAt)
}

func TestGuardedDeliveryClient_RefusesCarrierGradeNAT(t *testing.T) {
	resp, err := guardedDeliveryClient.Get("http://100.100.100.200/")
	if resp != nil {
		_ = resp.Body.Close()
	}
	assert.ErrorIs(t, err, errWebhookPrivateAddress)
}
