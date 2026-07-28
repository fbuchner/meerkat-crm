package services

import (
	"testing"

	"mycorrhizal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The guard is opt-in: self-hosted installs legitimately deliver webhooks to
// other services on the same host, so the default must stay unfiltered.
func TestClientForRespectsBlockPrivateURLs(t *testing.T) {
	assert.Same(t, deliveryClient, clientFor(config.Config{WebhookBlockPrivateURLs: false}),
		"default config must use the unfiltered client")
	assert.Same(t, guardedDeliveryClient, clientFor(config.Config{WebhookBlockPrivateURLs: true}),
		"WEBHOOK_BLOCK_PRIVATE_URLS must select the guarded client")
}

// Regression test for the original bypass: the guard used to be a pre-flight
// DNS check on the configured URL only, so a webhook pointed at a host that
// redirected to an internal address reached it anyway. Enforcement now lives in
// the transport's dialer, which every connection goes through — including ones
// opened for redirect targets.
func TestGuardedClientBlocksInternalAddresses(t *testing.T) {
	transport := guardedDeliveryClient.Transport
	require.NotNil(t, transport, "guarded client must not fall back to the default transport")

	for _, target := range []string{
		"http://127.0.0.1:1/hook",
		"http://169.254.169.254/latest/meta-data/", // cloud metadata
		"http://10.0.0.1/hook",
		"http://100.64.0.1/hook", // CGNAT
	} {
		t.Run(target, func(t *testing.T) {
			resp, err := guardedDeliveryClient.Get(target)
			if resp != nil {
				resp.Body.Close()
			}
			require.Error(t, err, "guarded client must refuse %s", target)
			assert.ErrorIs(t, err, ErrWebhookPrivateAddress)
		})
	}
}

func TestIsPrivateURLFailsClosed(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		private bool
	}{
		{"loopback", "http://127.0.0.1/hook", true},
		{"metadata", "http://169.254.169.254/", true},
		{"private range", "http://10.1.2.3/hook", true},
		{"CGNAT", "http://100.64.0.1/hook", true},
		{"unparseable URL", "http://[::1", true},
		{"unresolvable host", "http://nonexistent.invalid/hook", true},
		{"public IP literal", "http://8.8.8.8/hook", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.private, isPrivateURL(tt.url))
		})
	}
}
