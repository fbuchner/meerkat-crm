package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"mycorrhizal/config"
	"mycorrhizal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tier 3c item 9 (docs/fork-plan/95-backlog-and-priorities.md), re-dispatching
// Phase 3a of docs/fork-plan/45-test-coverage-closure.md: computeSignature,
// retryAt, saveDelivery, deliverWebhook, TriggerWebhooks and
// TestWebhookDelivery were all at 0% (or near it) coverage. This file covers
// the delivery path itself (signing, backoff, persistence, actual HTTP
// delivery, and the exported entry points). SSRF-guarding behavior
// (isPrivateURL, clientFor, the guarded dialer) is already covered by
// webhook_ssrf_test.go and is intentionally not duplicated here.

// ---- computeSignature ----------------------------------------------------

func TestComputeSignatureMatchesHMACSHA256Hex(t *testing.T) {
	secret := "top-secret"
	body := []byte(`{"hello":"world"}`)

	got := computeSignature(secret, body)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	want := fmt.Sprintf("%x", mac.Sum(nil))

	assert.Equal(t, want, got)
	// Sanity: hex-encoded SHA-256 HMAC is always 64 hex chars.
	assert.Len(t, got, 64)
}

func TestComputeSignatureDiffersOnSecretOrBody(t *testing.T) {
	base := computeSignature("secret-a", []byte("payload"))

	assert.NotEqual(t, base, computeSignature("secret-b", []byte("payload")),
		"signature must depend on the secret")
	assert.NotEqual(t, base, computeSignature("secret-a", []byte("different-payload")),
		"signature must depend on the body")
}

// TestComputeSignatureHandVerification is the required hand-verification for
// security-adjacent code: it recomputes the signature the way a receiving
// webhook consumer would (independently, via stdlib hmac/sha256) and confirms
// it matches computeSignature's output byte-for-byte. If computeSignature's
// secret or hash algorithm silently changed, this test — and the assertion
// used against the live HTTP header in TestDeliverWebhookSendsValidSignature
// below — would fail.
func TestComputeSignatureHandVerification(t *testing.T) {
	secret := "receiver-side-secret"
	body := []byte(`{"id":"abc","event":"contact.created"}`)

	got := computeSignature(secret, body)

	independentMAC := hmac.New(sha256.New, []byte(secret))
	independentMAC.Write(body)
	independentHex := fmt.Sprintf("%x", independentMAC.Sum(nil))

	require.Equal(t, independentHex, got,
		"a receiving webhook consumer verifying HMAC-SHA256 over the raw body with the shared secret must accept this signature")
}

// ---- retryAt --------------------------------------------------------------

func TestRetryAtSchedule(t *testing.T) {
	before := time.Now()
	first := retryAt(1)
	second := retryAt(2)
	third := retryAt(3) // beyond len(retryDelays) == 2 -> no further retry
	after := time.Now()

	require.NotNil(t, first)
	require.NotNil(t, second)
	assert.Nil(t, third, "attempts beyond the configured backoff schedule must not be retried")

	// attempt 1 -> now + 5m
	assert.True(t, !first.Before(before.Add(5*time.Minute)) && !first.After(after.Add(5*time.Minute)),
		"attempt 1 retry time %v must be ~5m from now (window %v..%v)", first, before.Add(5*time.Minute), after.Add(5*time.Minute))

	// attempt 2 -> now + 15m
	assert.True(t, !second.Before(before.Add(15*time.Minute)) && !second.After(after.Add(15*time.Minute)),
		"attempt 2 retry time %v must be ~15m from now (window %v..%v)", second, before.Add(15*time.Minute), after.Add(15*time.Minute))

	// second retry delay must be strictly longer than the first (backoff grows).
	assert.True(t, second.Sub(*first) > 0, "retry delay must grow between attempt 1 and attempt 2")
}

func TestRetryAtNilBeyondMaxAttempts(t *testing.T) {
	// maxDeliveryAttempts is 3; ProcessWebhookRetries only re-queues rows with
	// attempts < maxDeliveryAttempts, and retryAt is the mechanism that stops
	// scheduling further retries once the backoff table is exhausted.
	assert.Nil(t, retryAt(len(retryDelays)+1))
	assert.Nil(t, retryAt(maxDeliveryAttempts))
}

// ---- saveDelivery -----------------------------------------------------

func TestSaveDeliveryPersistsSuccessRecord(t *testing.T) {
	db := setupWebhookRetryTestDB(t)
	statusCode := 200

	d := saveDelivery(db, 42, "contact.created", `{"a":1}`, &statusCode, nil, 1, nil)

	require.NotZero(t, d.ID, "saveDelivery must return the persisted record with its ID populated")

	var loaded models.WebhookDelivery
	require.NoError(t, db.First(&loaded, d.ID).Error)
	assert.EqualValues(t, 42, loaded.WebhookID)
	assert.Equal(t, "contact.created", loaded.EventType)
	assert.Equal(t, `{"a":1}`, loaded.Payload)
	require.NotNil(t, loaded.StatusCode)
	assert.Equal(t, 200, *loaded.StatusCode)
	assert.Nil(t, loaded.Error)
	assert.Equal(t, 1, loaded.Attempts)
	assert.Nil(t, loaded.NextRetryAt)
}

// TestSaveDeliveryLogsAndReturnsOnCreateError covers the db.Create error
// branch: saveDelivery must not panic when persistence fails, and still
// returns the (unsaved) delivery value to its caller.
func TestSaveDeliveryLogsAndReturnsOnCreateError(t *testing.T) {
	db := setupWebhookRetryTestDB(t)
	// Drop the table out from under saveDelivery so db.Create fails.
	require.NoError(t, db.Migrator().DropTable(&models.WebhookDelivery{}))

	var d models.WebhookDelivery
	require.NotPanics(t, func() {
		d = saveDelivery(db, 1, "contact.created", `{}`, nil, nil, 1, nil)
	})
	assert.Zero(t, d.ID, "a failed Create must leave the record unsaved (no ID assigned)")
}

func TestSaveDeliveryPersistsFailureRecordWithRetry(t *testing.T) {
	db := setupWebhookRetryTestDB(t)
	errMsg := "unexpected status 500"
	next := time.Now().Add(5 * time.Minute)

	d := saveDelivery(db, 7, "contact.updated", `{}`, nil, &errMsg, 1, &next)

	var loaded models.WebhookDelivery
	require.NoError(t, db.First(&loaded, d.ID).Error)
	assert.Nil(t, loaded.StatusCode)
	require.NotNil(t, loaded.Error)
	assert.Equal(t, "unexpected status 500", *loaded.Error)
	require.NotNil(t, loaded.NextRetryAt)
	assert.WithinDuration(t, next, *loaded.NextRetryAt, time.Second)
}

// ---- deliverWebhook ---------------------------------------------------

func newTestWebhook(url, secret string) models.Webhook {
	return models.Webhook{
		UserID:   1,
		Name:     "test hook",
		URL:      url,
		Events:   []string{"contact.created"},
		Secret:   secret,
		IsActive: true,
	}
}

// TestDeliverWebhookSendsValidSignature is the primary "actual HTTP delivery"
// test: it stands up a real HTTP server, delivers through deliverWebhook, and
// asserts the request the server actually received carries a correct,
// independently-verifiable signature, the right headers, and the right body
// -- not just that deliverWebhook returned without error.
func TestDeliverWebhookSendsValidSignature(t *testing.T) {
	db := setupWebhookRetryTestDB(t)
	secret := "whsec_abc123"

	var (
		receivedBody      []byte
		receivedSignature string
		receivedEventHdr  string
		receivedContentTy string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		receivedBody = b
		receivedSignature = r.Header.Get("X-Webhook-Signature")
		receivedEventHdr = r.Header.Get("X-Mycorrhizal-Event")
		receivedContentTy = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	wh := newTestWebhook(server.URL, secret)
	require.NoError(t, db.Create(&wh).Error)

	body := []byte(`{"id":"evt_1","event":"contact.created","data":{"name":"Ada"}}`)
	cfg := config.Config{WebhookBlockPrivateURLs: false}

	delivery := deliverWebhook(db, cfg, wh, "contact.created", body, 1)

	require.NotNil(t, delivery.StatusCode)
	assert.Equal(t, 200, *delivery.StatusCode)
	assert.Nil(t, delivery.Error)
	assert.Nil(t, delivery.NextRetryAt, "a successful delivery must not schedule a retry")

	// The server must have received exactly the signed body...
	assert.Equal(t, body, receivedBody)
	assert.Equal(t, "application/json", receivedContentTy)
	assert.Equal(t, "contact.created", receivedEventHdr)

	// ...and the signature header must be independently verifiable by a
	// receiver who only knows the shared secret and the raw body.
	require.True(t, len(receivedSignature) > len("sha256="), "signature header must be present")
	require.Equal(t, "sha256=", receivedSignature[:len("sha256=")])
	gotHex := receivedSignature[len("sha256="):]

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(receivedBody)
	wantHex := fmt.Sprintf("%x", mac.Sum(nil))
	assert.Equal(t, wantHex, gotHex, "receiver-side HMAC verification of the delivered request must succeed")
}

func TestDeliverWebhookNon2xxSchedulesRetryAndRecordsStatus(t *testing.T) {
	db := setupWebhookRetryTestDB(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	wh := newTestWebhook(server.URL, "secret")
	require.NoError(t, db.Create(&wh).Error)
	cfg := config.Config{}

	before := time.Now()
	delivery := deliverWebhook(db, cfg, wh, "contact.created", []byte(`{}`), 1)
	after := time.Now()

	require.NotNil(t, delivery.StatusCode)
	assert.Equal(t, 500, *delivery.StatusCode)
	require.NotNil(t, delivery.Error)
	assert.Equal(t, "unexpected status 500", *delivery.Error)
	require.NotNil(t, delivery.NextRetryAt, "a failed delivery on attempt 1 must be scheduled for retry")
	assert.True(t, !delivery.NextRetryAt.Before(before.Add(5*time.Minute)) && !delivery.NextRetryAt.After(after.Add(5*time.Minute)))
	assert.Equal(t, 1, delivery.Attempts)
}

func TestDeliverWebhookFinalAttemptDoesNotScheduleRetry(t *testing.T) {
	db := setupWebhookRetryTestDB(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	wh := newTestWebhook(server.URL, "secret")
	require.NoError(t, db.Create(&wh).Error)
	cfg := config.Config{}

	// attempt 3 exceeds len(retryDelays) == 2, so no further retry is scheduled
	// even though the request itself still fails.
	delivery := deliverWebhook(db, cfg, wh, "contact.created", []byte(`{}`), maxDeliveryAttempts)

	require.NotNil(t, delivery.StatusCode)
	assert.Equal(t, 500, *delivery.StatusCode)
	assert.Nil(t, delivery.NextRetryAt, "exhausted retry budget must not schedule another retry")
	assert.Equal(t, maxDeliveryAttempts, delivery.Attempts)
}

func TestDeliverWebhookConnectionErrorSchedulesRetry(t *testing.T) {
	db := setupWebhookRetryTestDB(t)

	// Port 1 on loopback: nothing listens there, so the client.Do call fails
	// with a connection error rather than returning a response.
	wh := newTestWebhook("http://127.0.0.1:1/hook", "secret")
	require.NoError(t, db.Create(&wh).Error)
	cfg := config.Config{WebhookBlockPrivateURLs: false}

	delivery := deliverWebhook(db, cfg, wh, "contact.created", []byte(`{}`), 1)

	assert.Nil(t, delivery.StatusCode, "a transport-level failure never gets a status code")
	require.NotNil(t, delivery.Error)
	require.NotNil(t, delivery.NextRetryAt)
}

func TestDeliverWebhookMalformedURLSchedulesRetry(t *testing.T) {
	db := setupWebhookRetryTestDB(t)

	// Malformed enough that http.NewRequest itself fails (invalid IPv6
	// literal, unbalanced bracket) while cfg.WebhookBlockPrivateURLs is off so
	// the isPrivateURL pre-check is skipped and this exercises the
	// http.NewRequest error branch specifically.
	wh := newTestWebhook("http://[::1", "secret")
	require.NoError(t, db.Create(&wh).Error)
	cfg := config.Config{WebhookBlockPrivateURLs: false}

	delivery := deliverWebhook(db, cfg, wh, "contact.created", []byte(`{}`), 1)

	assert.Nil(t, delivery.StatusCode)
	require.NotNil(t, delivery.Error)
	require.NotNil(t, delivery.NextRetryAt)
}

func TestDeliverWebhookBlocksPrivateURLWhenConfigured(t *testing.T) {
	db := setupWebhookRetryTestDB(t)

	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// httptest servers listen on 127.0.0.1, a loopback address, so with the
	// guard enabled this must be rejected before any request is sent.
	wh := newTestWebhook(server.URL, "secret")
	require.NoError(t, db.Create(&wh).Error)
	cfg := config.Config{WebhookBlockPrivateURLs: true}

	delivery := deliverWebhook(db, cfg, wh, "contact.created", []byte(`{}`), 1)

	assert.Equal(t, int32(0), atomic.LoadInt32(&hits), "the guarded delivery must never reach the destination server")
	assert.Nil(t, delivery.StatusCode)
	require.NotNil(t, delivery.Error)
	assert.Equal(t, "webhook URL resolves to a private or loopback address", *delivery.Error)
	// This is the pre-flight branch in deliverWebhook (distinct from the
	// dialer-level guard); it does not call retryAt at all, so no retry is
	// scheduled here even though it's attempt 1.
	assert.Nil(t, delivery.NextRetryAt)
}

// ---- TriggerWebhooks ----------------------------------------------------

func TestTriggerWebhooksFansOutToSubscribedActiveWebhooksOnly(t *testing.T) {
	db := setupWebhookRetryTestDB(t)

	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	const userID = uint(1)

	subscribedActive := models.Webhook{UserID: userID, Name: "match", URL: server.URL, Events: []string{"contact.created"}, Secret: "s1", IsActive: true}
	require.NoError(t, db.Create(&subscribedActive).Error)

	notSubscribed := models.Webhook{UserID: userID, Name: "no-match", URL: server.URL, Events: []string{"contact.updated"}, Secret: "s2", IsActive: true}
	require.NoError(t, db.Create(&notSubscribed).Error)

	inactive := models.Webhook{UserID: userID, Name: "inactive", URL: server.URL, Events: []string{"contact.created"}, Secret: "s3", IsActive: false}
	require.NoError(t, db.Create(&inactive).Error)
	// models.Webhook.IsActive carries `gorm:"default:true"`; GORM treats the Go
	// zero value (false) on Create as "unset" and silently applies the SQL
	// default instead, so an explicit UpdateColumn is needed to actually
	// persist IsActive=false here.
	require.NoError(t, db.Model(&inactive).UpdateColumn("is_active", false).Error)

	otherUser := models.Webhook{UserID: userID + 1, Name: "other-user", URL: server.URL, Events: []string{"contact.created"}, Secret: "s4", IsActive: true}
	require.NoError(t, db.Create(&otherUser).Error)

	softDeleted := models.Webhook{UserID: userID, Name: "deleted", URL: server.URL, Events: []string{"contact.created"}, Secret: "s5", IsActive: true}
	require.NoError(t, db.Create(&softDeleted).Error)
	require.NoError(t, db.Delete(&softDeleted).Error)

	cfg := config.Config{}
	TriggerWebhooks(db, cfg, userID, "contact.created", map[string]string{"name": "Ada"})

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&hits) >= 1
	}, 3*time.Second, 10*time.Millisecond, "the subscribed active webhook must be delivered to")

	// Give any accidental extra deliveries a moment to land, then confirm
	// exactly one delivery was made and it's attributed to the right webhook.
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, int32(1), atomic.LoadInt32(&hits), "only the subscribed, active, non-deleted webhook for this user must fire")

	var deliveries []models.WebhookDelivery
	require.NoError(t, db.Find(&deliveries).Error)
	require.Len(t, deliveries, 1)
	assert.Equal(t, subscribedActive.ID, deliveries[0].WebhookID)
	assert.Equal(t, "contact.created", deliveries[0].EventType)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(deliveries[0].Payload), &payload))
	assert.Equal(t, "contact.created", payload["event"])
}

func TestTriggerWebhooksNoActiveSubscriptionsDeliversNothing(t *testing.T) {
	db := setupWebhookRetryTestDB(t)
	cfg := config.Config{}

	assert.NotPanics(t, func() {
		TriggerWebhooks(db, cfg, 999, "contact.created", map[string]string{"x": "y"})
	})

	time.Sleep(50 * time.Millisecond)
	var deliveries []models.WebhookDelivery
	require.NoError(t, db.Find(&deliveries).Error)
	assert.Empty(t, deliveries)
}

// TestTriggerWebhooksDBQueryErrorIsNoop covers the webhooks-lookup error
// branch: if the initial query fails, TriggerWebhooks must log and return
// without attempting any delivery, rather than panicking on a nil/empty slice.
func TestTriggerWebhooksDBQueryErrorIsNoop(t *testing.T) {
	db := setupWebhookRetryTestDB(t)
	// Drop the table out from under the lookup query so it errors.
	require.NoError(t, db.Migrator().DropTable(&models.Webhook{}))

	assert.NotPanics(t, func() {
		TriggerWebhooks(db, config.Config{}, 1, "contact.created", map[string]string{"x": "y"})
	})
}

// TestTriggerWebhooksPayloadMarshalFailureIsNoop covers the buildPayloadBody
// error branch: TriggerWebhooks must log and return without dispatching to
// any webhook rather than panicking or sending a broken payload.
func TestTriggerWebhooksPayloadMarshalFailureIsNoop(t *testing.T) {
	db := setupWebhookRetryTestDB(t)

	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	const userID = uint(1)
	wh := models.Webhook{UserID: userID, Name: "match", URL: server.URL, Events: []string{"contact.created"}, Secret: "s1", IsActive: true}
	require.NoError(t, db.Create(&wh).Error)

	// channels are not JSON-marshalable, so buildPayloadBody must fail here.
	unmarshalable := make(chan int)

	assert.NotPanics(t, func() {
		TriggerWebhooks(db, config.Config{}, userID, "contact.created", unmarshalable)
	})

	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, int32(0), atomic.LoadInt32(&hits), "a payload that fails to marshal must never reach a webhook")

	var deliveries []models.WebhookDelivery
	require.NoError(t, db.Find(&deliveries).Error)
	assert.Empty(t, deliveries)
}

// ---- TestWebhookDelivery ---------------------------------------------------

func TestTestWebhookDeliverySendsTestPayloadAndPersists(t *testing.T) {
	db := setupWebhookRetryTestDB(t)

	var receivedEventHdr string
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		receivedBody = b
		receivedEventHdr = r.Header.Get("X-Mycorrhizal-Event")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	wh := models.Webhook{UserID: 1, Name: "test", URL: server.URL, Events: []string{"contact.created"}, Secret: "s1", IsActive: true}
	require.NoError(t, db.Create(&wh).Error)

	delivery := TestWebhookDelivery(db, config.Config{}, wh)

	require.NotNil(t, delivery.StatusCode)
	assert.Equal(t, 200, *delivery.StatusCode)
	assert.Nil(t, delivery.Error)
	assert.Equal(t, "test", delivery.EventType)
	assert.Equal(t, "test", receivedEventHdr)
	// Ignores event subscriptions entirely: this webhook is only subscribed to
	// contact.created, yet the test delivery still goes out.

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(receivedBody, &payload))
	assert.Equal(t, "test", payload["event"])
	data, ok := payload["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, data["message"], "test webhook delivery")

	// Persisted, findable by ID.
	var loaded models.WebhookDelivery
	require.NoError(t, db.First(&loaded, delivery.ID).Error)
	assert.Equal(t, delivery.WebhookID, loaded.WebhookID)
}

func TestTestWebhookDeliveryRecordsFailureOnUnreachableURL(t *testing.T) {
	db := setupWebhookRetryTestDB(t)

	wh := models.Webhook{UserID: 1, Name: "test", URL: "http://127.0.0.1:1/hook", Events: []string{}, Secret: "s1", IsActive: true}
	require.NoError(t, db.Create(&wh).Error)

	delivery := TestWebhookDelivery(db, config.Config{}, wh)

	assert.Nil(t, delivery.StatusCode)
	require.NotNil(t, delivery.Error)
	assert.Equal(t, "test", delivery.EventType)
}
