package services

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/stretchr/testify/require"
)

// newDiscoveryTestProvider builds an OIDCProvider via a real discovery fetch
// (oidc.NewProvider hitting /.well-known/openid-configuration), unlike
// newTestProvider (oidc_userinfo_test.go) which builds the *oidc.Provider
// directly via oidc.ProviderConfig and never populates its rawClaims -- that
// shortcut means end_session_endpoint (and any other field go-oidc doesn't
// explicitly parse) would never be resolvable through that harness.
func newDiscoveryTestProvider(t *testing.T, endSessionEndpoint string) (*OIDCProvider, func()) {
	t.Helper()

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		endSessionJSON := ""
		if endSessionEndpoint != "" {
			endSessionJSON = fmt.Sprintf(`,"end_session_endpoint":%q`, server.URL+endSessionEndpoint)
		}
		fmt.Fprintf(w, `{"issuer":%q,"authorization_endpoint":%q,"token_endpoint":%q,"jwks_uri":%q%s}`,
			server.URL, server.URL+"/auth", server.URL+"/token", server.URL+"/keys", endSessionJSON)
	}))

	provider, err := oidc.NewProvider(context.Background(), server.URL)
	require.NoError(t, err)

	return &OIDCProvider{provider: provider}, server.Close
}

func TestEndSessionEndpoint_ResolvesFromDiscoveryDocument(t *testing.T) {
	p, cleanup := newDiscoveryTestProvider(t, "/end_session")
	defer cleanup()

	endpoint, err := p.EndSessionEndpoint()
	require.NoError(t, err)
	require.Contains(t, endpoint, "/end_session")
}

func TestEndSessionEndpoint_EmptyWhenProviderDoesNotAdvertiseOne(t *testing.T) {
	p, cleanup := newDiscoveryTestProvider(t, "")
	defer cleanup()

	endpoint, err := p.EndSessionEndpoint()
	require.NoError(t, err)
	require.Empty(t, endpoint)
}
