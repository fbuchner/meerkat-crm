package services

import (
	"bytes"
	"encoding/xml"
	"io"
	"meerkat/models"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serves the remote's address book with its ETags rewritten by etag, as servers that do not quote
// them or that send weak ETags do
func proxyWithETags(t *testing.T, remote *davRemote, etag func(opaque string) string) string {
	t.Helper()
	target, err := url.Parse(remote.directURL)
	require.NoError(t, err)
	quoted := regexp.MustCompile(`(<[^>/]*getetag[^>]*>)(?:"|&#34;)([^"&<]*)(?:"|&#34;)(</)`)
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ModifyResponse = func(resp *http.Response) error {
		if h := resp.Header.Get("ETag"); h != "" {
			resp.Header.Set("ETag", etag(strings.Trim(h, `"`)))
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		body = quoted.ReplaceAllFunc(body, func(m []byte) []byte {
			parts := quoted.FindSubmatch(m)
			var out bytes.Buffer
			_ = xml.EscapeText(&out, []byte(etag(string(parts[2]))))
			return []byte(string(parts[1]) + out.String() + string(parts[3]))
		})
		resp.Body = io.NopCloser(bytes.NewReader(body))
		resp.ContentLength = int64(len(body))
		resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
		return nil
	}
	srv := httptest.NewServer(proxy)
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestContactSyncWorksWithServersThatSendUnusualETags(t *testing.T) {
	cases := map[string]func(string) string{
		"unquoted": func(o string) string { return o },
		"weak":     func(o string) string { return `W/"` + o + `"` },
	}
	for name, etag := range cases {
		t.Run(name, func(t *testing.T) {
			remote := newMeerkatRemote(t)
			remote.putContact(t, "uid-remote", "Rita", "Remote", "")
			env := setupContactSyncEnv(t, models.CardDAVDirectionTwoWay, remote)
			env.conn.BaseURL = proxyWithETags(t, remote, etag)
			env.createContact(t, "Lou", "Local", "")

			stats := env.sync(t)

			assert.Equal(t, 1, stats.PulledCreated)
			assert.Equal(t, 1, stats.PushedCreated)
			assert.Zero(t, stats.Errors, env.conn.LastSyncError)
			assert.Equal(t, 2, remote.count(t))

			again := env.sync(t)
			assert.Zero(t, again.PulledUpdated+again.PushedUpdated+again.Conflicts+again.Errors, "a second run changes nothing")
		})
	}
}
