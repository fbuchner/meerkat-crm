package httputil

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- validateURLForSSRF ---
//
// These cases exercise validateURLForSSRF directly (this file is in package
// httputil, so the unexported function is reachable) and are all resolved
// without any real DNS/network access: literal IPs are parsed by
// net.LookupIP without a lookup, and the ".invalid" TLD used for the
// resolution-failure case is reserved by RFC 2606 to never resolve, so the
// local resolver answers NXDOMAIN immediately rather than reaching the
// network.

func TestValidateURLForSSRF(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantErr string // substring expected in the error, empty means no error
	}{
		{"allows a public IPv4 literal", "http://8.8.8.8/photo.jpg", ""},
		{"allows https scheme", "https://8.8.8.8/photo.jpg", ""},

		{"rejects unparsable URL", "http://\x7f", "invalid URL format"},
		{"rejects non-http(s) scheme", "ftp://example.com/photo.jpg", "only http and https URLs are allowed"},
		{"rejects file scheme", "file:///etc/passwd", "only http and https URLs are allowed"},
		{"rejects a URL with no host", "http://", "URL must have a host"},

		{"blocks localhost hostname", "http://localhost/photo.jpg", "internal hosts"},
		{"blocks 127.0.0.1 hostname", "http://127.0.0.1/photo.jpg", "internal hosts"},
		{"blocks 0.0.0.0 hostname", "http://0.0.0.0/photo.jpg", "internal hosts"},
		{"blocks ::1 hostname", "http://[::1]/photo.jpg", "internal hosts"},
		{"blocks LOCALHOST case-insensitively", "http://LOCALHOST/photo.jpg", "internal hosts"},

		{"blocks private LAN address by resolved IP", "http://192.168.1.1/photo.jpg", "internal IP addresses"},
		{"blocks 10/8 address by resolved IP", "http://10.0.0.5/photo.jpg", "internal IP addresses"},
		{"blocks link-local cloud-metadata address", "http://169.254.169.254/latest/meta-data/", "internal IP addresses"},
		{"blocks CGNAT address", "http://100.64.0.1/photo.jpg", "internal IP addresses"},

		{"fails closed when the hostname does not resolve", "http://nonexistent-host-xyz.invalid/photo.jpg", "failed to resolve hostname"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := validateURLForSSRF(tt.rawURL)
			if tt.wantErr == "" {
				require.NoError(t, err)
				require.NotNil(t, parsed)
				assert.Equal(t, tt.rawURL, parsed.String())
				return
			}
			require.Error(t, err)
			assert.Nil(t, parsed)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// --- FetchImageFromURL: SSRF protection ---
//
// FetchImageFromURL delegates to validateURLForSSRF for the initial check and
// additionally pins the resolved IP at dial time via SafeDialContext (see
// safedial.go), so DNS rebinding between validation and connection, and
// redirects to a disallowed location, are both closed too. This mirrors
// photostore's FetchPhotoFromURL SSRF suite (photostore/photostore_test.go),
// adapted to call httputil.FetchImageFromURL directly.

func TestFetchImageFromURL_BlocksLocalhostHostname(t *testing.T) {
	if _, _, err := FetchImageFromURL("http://localhost/photo.jpg"); err == nil {
		t.Error("expected fetching from localhost to be rejected as an SSRF target")
	}
}

func TestFetchImageFromURL_BlocksLoopbackIPv4Literal(t *testing.T) {
	if _, _, err := FetchImageFromURL("http://127.0.0.1/photo.jpg"); err == nil {
		t.Error("expected fetching from 127.0.0.1 to be rejected as an SSRF target")
	}
}

func TestFetchImageFromURL_BlocksLoopbackIPv6Literal(t *testing.T) {
	if _, _, err := FetchImageFromURL("http://[::1]/photo.jpg"); err == nil {
		t.Error("expected fetching from [::1] to be rejected as an SSRF target")
	}
}

func TestFetchImageFromURL_BlocksUnspecifiedAddress(t *testing.T) {
	// 0.0.0.0 is in fetch.go's blockedHosts list but not photostore's -- worth
	// its own case since it is a real SSRF vector (many OSes route 0.0.0.0 to
	// the local host) and is checked by hostname string, not resolved IP.
	if _, _, err := FetchImageFromURL("http://0.0.0.0/photo.jpg"); err == nil {
		t.Error("expected fetching from 0.0.0.0 to be rejected as an SSRF target")
	}
}

func TestFetchImageFromURL_BlocksPrivateLANAddress(t *testing.T) {
	// 192.168.0.0/16 is not in the hardcoded hostname blocklist at all -- this
	// proves the resolved-IP check (IsPublicIP), not just the hostname string
	// list, actually rejects it.
	if _, _, err := FetchImageFromURL("http://192.168.1.1/photo.jpg"); err == nil {
		t.Error("expected fetching from a private LAN address (192.168.1.1) to be rejected")
	}
}

func TestFetchImageFromURL_BlocksLinkLocalMetadataAddress(t *testing.T) {
	// 169.254.169.254 is the classic cloud-metadata SSRF target (AWS/GCP/Azure
	// instance metadata service); IsPublicIP must catch it.
	if _, _, err := FetchImageFromURL("http://169.254.169.254/latest/meta-data/"); err == nil {
		t.Error("expected fetching from the link-local metadata address to be rejected")
	}
}

func TestFetchImageFromURL_RejectsNonHTTPScheme(t *testing.T) {
	if _, _, err := FetchImageFromURL("file:///etc/passwd"); err == nil {
		t.Error("expected a non-http(s) scheme to be rejected")
	}
}

func TestFetchImageFromURL_RejectsUnparsableURL(t *testing.T) {
	// A URL containing a raw control character fails url.Parse itself.
	_, err := url.Parse("http://\x7f")
	if err == nil {
		t.Skip("test environment's url.Parse unexpectedly accepted the malformed URL")
	}
	if _, _, err := FetchImageFromURL("http://\x7f"); err == nil {
		t.Error("expected an unparsable URL to be rejected")
	}
}

func TestFetchImageFromURL_RejectsURLWithNoHost(t *testing.T) {
	if _, _, err := FetchImageFromURL("http://"); err == nil {
		t.Error("expected a URL with no host to be rejected")
	}
}

func TestFetchImageFromURL_SanitizesEmbeddedWhitespaceBeforeValidating(t *testing.T) {
	// Google VCF-exported PHOTO;VALUE=uri fields can arrive line-folded, with
	// embedded "\r\n " sequences splitting the URL. FetchImageFromURL strips
	// spaces/newlines/CRs before parsing (fetch.go lines 64-67). Prove that
	// happens -- and that the SSRF guard still applies to the *cleaned* URL --
	// by embedding a "\r\n" inside a loopback URL that would otherwise fail
	// url.Parse outright (net/url rejects raw control characters). If we see
	// the specific "internal hosts" rejection rather than a generic parse
	// error, the cleanup ran and the result was still correctly blocked.
	dirty := "http://127.0.0.1/pho\r\nto.jpg"
	_, _, err := FetchImageFromURL(dirty)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "internal hosts",
		"expected the whitespace-cleaned URL to be rejected as a loopback target, not fail as an unparsable URL")
}

// --- FetchImageFromURL: proof the request never reaches a listening server ---
//
// The cases above only assert on the returned error, which would also be
// true if the SSRF check merely reordered rather than actually blocked the
// dial. These two spin up a *real* httptest.Server (which, per Go's stdlib,
// only ever binds to a loopback address) and assert the handler is never
// invoked -- if the guard were broken, these would either connect and
// receive the fixture image back, or increment the hit counter.

func TestFetchImageFromURL_LoopbackIPv4LiteralNeverReachesListener(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not actually reached"))
	}))
	defer server.Close()

	// server.URL is already an http://127.0.0.1:<port> URL -- exactly the
	// loopback literal validateURLForSSRF must block.
	require.True(t, strings.HasPrefix(server.URL, "http://127.0.0.1:"), "sanity check: httptest.Server must bind to loopback for this test to prove anything")

	data, contentType, err := FetchImageFromURL(server.URL + "/photo.jpg")
	assert.Error(t, err, "expected the loopback listener to be rejected as an SSRF target")
	assert.Nil(t, data)
	assert.Empty(t, contentType)
	assert.Equal(t, int32(0), atomic.LoadInt32(&hits), "the SSRF guard must prevent the request from ever reaching the listening server")
}

func TestFetchImageFromURL_LocalhostHostnameNeverReachesListener(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not actually reached"))
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	localhostURL := "http://localhost:" + serverURL.Port() + "/photo.jpg"

	data, contentType, err := FetchImageFromURL(localhostURL)
	assert.Error(t, err, "expected the localhost listener to be rejected as an SSRF target")
	assert.Nil(t, data)
	assert.Empty(t, contentType)
	assert.Equal(t, int32(0), atomic.LoadInt32(&hits), "the SSRF guard must prevent the request from ever reaching the listening server")
}
