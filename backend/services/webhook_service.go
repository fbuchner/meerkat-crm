package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"meerkat/config"
	"meerkat/logger"
	"meerkat/models"
	"net/http"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	maxDeliveryAttempts    = 3
	webhookDeliveryTimeout = 15 * time.Second

	// Stored on the delivery and shown to its owner. Deliberately generic: the raw transport
	// error says whether a port is closed, filtered or unresolvable, which would turn the
	// test endpoint into a port scanner for any signed-in user. The detail goes to the log.
	webhookErrInvalidURL     = "webhook URL is invalid"
	webhookErrDeliveryFailed = "could not connect to the webhook URL"
	webhookErrPrivateBlocked = "webhook URL resolves to a private or loopback address"
)

var (
	errWebhookUnreachable    = errors.New("webhook host unreachable")
	errWebhookPrivateAddress = errors.New("webhook host resolves to a private address")

	deliveryClient = &http.Client{Timeout: webhookDeliveryTimeout}
	// guardedDeliveryClient checks the address at dial time, so the check also covers
	// redirects and cannot be bypassed by a DNS answer that changes after validation.
	guardedDeliveryClient = &http.Client{
		Timeout: webhookDeliveryTimeout,
		Transport: &http.Transport{
			DialContext: newPrivateBlockingDialContext(errWebhookUnreachable, errWebhookPrivateAddress),
		},
	}
	// semaphore limits concurrent outbound webhook HTTP calls
	deliverySem = make(chan struct{}, 10)
)

var retryDelays = []time.Duration{5 * time.Minute, 15 * time.Minute}

type webhookPayload struct {
	ID        string      `json:"id"`
	Event     string      `json:"event"`
	Timestamp string      `json:"timestamp"`
	Data      interface{} `json:"data"`
}

func buildPayloadBody(eventType string, data interface{}) ([]byte, error) {
	payload := webhookPayload{
		ID:        uuid.New().String(),
		Event:     eventType,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data:      data,
	}
	return json.Marshal(payload)
}

// TriggerWebhooks fires webhooks for all active subscriptions matching eventType for the user.
// Runs each delivery in its own goroutine (non-blocking).
func TriggerWebhooks(db *gorm.DB, cfg config.Config, userID uint, eventType string, data interface{}) {
	var webhooks []models.Webhook
	if err := db.Where("user_id = ? AND is_active = ? AND deleted_at IS NULL", userID, true).Find(&webhooks).Error; err != nil {
		logger.Error().Err(err).Uint("user_id", userID).Msg("Failed to load webhooks for triggering")
		return
	}

	body, err := buildPayloadBody(eventType, data)
	if err != nil {
		logger.Error().Err(err).Str("event", eventType).Msg("Failed to build webhook payload")
		return
	}

	for _, wh := range webhooks {
		subscribed := false
		for _, e := range wh.Events {
			if e == eventType {
				subscribed = true
				break
			}
		}
		if !subscribed {
			continue
		}

		wh := wh
		go func() {
			deliverySem <- struct{}{}
			defer func() { <-deliverySem }()
			deliverWebhook(db, cfg, wh, eventType, body, 1)
		}()
	}
}

// TestWebhookDelivery delivers a test payload directly to the given webhook, ignoring event subscriptions.
func TestWebhookDelivery(db *gorm.DB, cfg config.Config, wh models.Webhook) models.WebhookDelivery {
	testData := map[string]interface{}{
		"message": "This is a test webhook delivery from Meerkat CRM.",
	}
	body, err := buildPayloadBody("test", testData)
	if err != nil {
		errStr := err.Error()
		d := models.WebhookDelivery{WebhookID: wh.ID, EventType: "test", Payload: "{}", Error: &errStr, Attempts: 1}
		db.Create(&d)
		return d
	}
	return deliverWebhook(db, cfg, wh, "test", body, 1)
}

func deliverWebhook(db *gorm.DB, cfg config.Config, wh models.Webhook, eventType string, body []byte, attempt int) models.WebhookDelivery {
	client := deliveryClient
	if cfg.WebhookBlockPrivateURLs {
		client = guardedDeliveryClient
	}

	sig := computeSignature(wh.Secret, body)
	req, err := http.NewRequest("POST", wh.URL, bytes.NewReader(body))
	if err != nil {
		logger.Warn().Err(err).Uint("webhook_id", wh.ID).Msg("Failed to build webhook request")
		errStr := webhookErrInvalidURL
		return saveDelivery(db, wh.ID, eventType, string(body), nil, &errStr, attempt, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", "sha256="+sig)
	req.Header.Set("X-Meerkat-Event", eventType)

	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, errWebhookPrivateAddress) {
			errStr := webhookErrPrivateBlocked
			return saveDelivery(db, wh.ID, eventType, string(body), nil, &errStr, attempt, nil)
		}
		logger.Warn().Err(err).Uint("webhook_id", wh.ID).Msg("Webhook delivery failed")
		errStr := webhookErrDeliveryFailed
		return saveDelivery(db, wh.ID, eventType, string(body), nil, &errStr, attempt, retryAt(attempt))
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	statusCode := resp.StatusCode
	if statusCode >= 200 && statusCode < 300 {
		return saveDelivery(db, wh.ID, eventType, string(body), &statusCode, nil, attempt, nil)
	}
	errStr := fmt.Sprintf("unexpected status %d", statusCode)
	return saveDelivery(db, wh.ID, eventType, string(body), &statusCode, &errStr, attempt, retryAt(attempt))
}

func computeSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return fmt.Sprintf("%x", mac.Sum(nil))
}

func retryAt(attempt int) *time.Time {
	if attempt > len(retryDelays) {
		return nil
	}
	t := time.Now().Add(retryDelays[attempt-1])
	return &t
}

func saveDelivery(db *gorm.DB, webhookID uint, eventType, payload string, statusCode *int, errMsg *string, attempts int, nextRetryAt *time.Time) models.WebhookDelivery {
	d := models.WebhookDelivery{
		WebhookID:   webhookID,
		EventType:   eventType,
		Payload:     payload,
		StatusCode:  statusCode,
		Error:       errMsg,
		Attempts:    attempts,
		NextRetryAt: nextRetryAt,
	}
	if err := db.Create(&d).Error; err != nil {
		logger.Error().Err(err).Uint("webhook_id", webhookID).Msg("Failed to save webhook delivery")
	}
	return d
}

// ProcessWebhookRetries retries failed deliveries where NextRetryAt <= now and Attempts < maxDeliveryAttempts.
func ProcessWebhookRetries(db *gorm.DB, cfg config.Config) {
	now := time.Now()
	var deliveries []models.WebhookDelivery
	if err := db.Where("next_retry_at <= ? AND attempts < ? AND deleted_at IS NULL", now, maxDeliveryAttempts).Find(&deliveries).Error; err != nil {
		logger.Error().Err(err).Msg("Failed to load webhook deliveries for retry")
		return
	}

	for _, d := range deliveries {
		var wh models.Webhook
		if err := db.Where("id = ? AND is_active = ? AND user_id IN (SELECT id FROM users WHERE deleted_at IS NULL)", d.WebhookID, true).First(&wh).Error; err != nil {
			logger.Warn().Err(err).Uint("webhook_id", d.WebhookID).Msg("Webhook not found, inactive or its user deleted, skipping retry")
			db.Model(&d).Update("next_retry_at", nil)
			continue
		}

		// Clear next_retry_at so it won't be picked up again until the new delivery is saved
		if err := db.Model(&d).Update("next_retry_at", nil).Error; err != nil {
			logger.Error().Err(err).Uint("delivery_id", d.ID).Msg("Failed to clear next_retry_at")
			continue
		}

		d, wh := d, wh
		go func() {
			deliverySem <- struct{}{}
			defer func() { <-deliverySem }()
			// Replay the original payload so retries share the same event ID and timestamp
			deliverWebhook(db, cfg, wh, d.EventType, []byte(d.Payload), d.Attempts+1)
		}()
	}
}
