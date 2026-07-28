package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mycorrhizal/config"
	"mycorrhizal/httputil"
	"mycorrhizal/logger"
	"mycorrhizal/models"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const maxDeliveryAttempts = 3

// Sentinels surfaced by the SSRF-guarded dialer. They are returned as ordinary
// dial errors and end up in the stored delivery record's Error field.
var (
	ErrWebhookUnreachable    = errors.New("webhook host could not be resolved")
	ErrWebhookPrivateAddress = errors.New("webhook URL resolves to a private or loopback address")
)

var (
	// deliveryClient is used when WEBHOOK_BLOCK_PRIVATE_URLS is off, which is
	// the default: self-hosted installs legitimately point webhooks at other
	// services on the same box or LAN, so no address filtering is applied.
	deliveryClient = &http.Client{Timeout: 15 * time.Second}

	// guardedDeliveryClient is used when WEBHOOK_BLOCK_PRIVATE_URLS is on. The
	// filtering lives in the dialer rather than in a pre-flight URL check
	// because every connection the transport opens — including ones for
	// redirect targets — must be validated. A pre-flight check alone is
	// bypassable two ways: a 302 to an internal address (the redirect target
	// was never checked) and DNS rebinding (the dial re-resolves and can get a
	// different answer than the check did).
	guardedDeliveryClient = &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			DialContext: httputil.SafeDialContext(ErrWebhookUnreachable, ErrWebhookPrivateAddress),
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("too many redirects")
			}
			return nil
		},
	}

	// semaphore limits concurrent outbound webhook HTTP calls
	deliverySem = make(chan struct{}, 10)
)

// clientFor picks the delivery client matching the configured SSRF policy.
func clientFor(cfg config.Config) *http.Client {
	if cfg.WebhookBlockPrivateURLs {
		return guardedDeliveryClient
	}
	return deliveryClient
}

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

// isPrivateURL reports whether rawURL resolves to any non-public address. It
// is only a fast pre-flight so the stored delivery record carries a clear
// error; the authoritative check is the pinning dialer on
// guardedDeliveryClient, which also covers redirects and DNS rebinding.
//
// Fails closed: an unparseable URL or a failed lookup counts as private, since
// we cannot show it is safe.
func isPrivateURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return true
	}
	ips, err := net.LookupIP(u.Hostname())
	if err != nil {
		return true
	}
	for _, ip := range ips {
		if !httputil.IsPublicIP(ip) {
			return true
		}
	}
	return false
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
		"message": "This is a test webhook delivery from Mycorrhizal CRM.",
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
	if cfg.WebhookBlockPrivateURLs && isPrivateURL(wh.URL) {
		errStr := "webhook URL resolves to a private or loopback address"
		return saveDelivery(db, wh.ID, eventType, string(body), nil, &errStr, attempt, nil)
	}

	sig := computeSignature(wh.Secret, body)
	req, err := http.NewRequest("POST", wh.URL, bytes.NewReader(body))
	if err != nil {
		errStr := err.Error()
		return saveDelivery(db, wh.ID, eventType, string(body), nil, &errStr, attempt, retryAt(attempt))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", "sha256="+sig)
	req.Header.Set("X-Mycorrhizal-Event", eventType)

	resp, err := clientFor(cfg).Do(req)
	if err != nil {
		errStr := err.Error()
		return saveDelivery(db, wh.ID, eventType, string(body), nil, &errStr, attempt, retryAt(attempt))
	}
	defer func() {
		io.Copy(io.Discard, resp.Body) //nolint:errcheck
		resp.Body.Close()
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
		if err := db.Where("id = ? AND is_active = ?", d.WebhookID, true).First(&wh).Error; err != nil {
			logger.Warn().Err(err).Uint("webhook_id", d.WebhookID).Msg("Webhook not found or inactive for retry")
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
