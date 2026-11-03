package services

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"meerkat/httputil"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// builds a DialContext that refuses connections to private/loopback/link-local addresses (SSRF protection for cloud
// deployments). The caller supplies its feature-specific sentinel errors so
// controllers can map failures without cross-feature coupling.
func newPrivateBlockingDialContext(errUnreachable, errPrivateAddress error) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
		if err != nil {
			return nil, fmt.Errorf("%w: could not resolve host", errUnreachable)
		}
		dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 10 * time.Second}
		// Try each allowed address in turn, as the standard dialer would.
		dialErr := errPrivateAddress
		for _, ip := range ips {
			if httputil.IsPrivateIP(ip) {
				continue
			}
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return conn, nil
			}
			dialErr = err
		}
		return nil, dialErr
	}
}

// limitedBody caps a response body's size, returning errTooLarge once exceeded.
type limitedBody struct {
	inner       io.ReadCloser
	remaining   int64
	errTooLarge error
}

func (b *limitedBody) Read(p []byte) (int, error) {
	if b.remaining <= 0 {
		// The limit was consumed exactly; only fail if more data follows.
		var probe [1]byte
		n, err := b.inner.Read(probe[:])
		if n > 0 {
			return 0, b.errTooLarge
		}
		return 0, err
	}
	if int64(len(p)) > b.remaining {
		p = p[:b.remaining]
	}
	n, err := b.inner.Read(p)
	b.remaining -= int64(n)
	return n, err
}

func (b *limitedBody) Close() error { return b.inner.Close() }

var getETagElement = regexp.MustCompile(`(<(?:[\w.-]+:)?getetag(?:\s[^>]*[^/>])?>)([^<]*)(</(?:[\w.-]+:)?getetag\s*>)`)

// quoteETags rewrites a response's ETags into the quoted form go-webdav insists on. Some servers
// send them unquoted or weak (probably Mailbox.org), and go-webdav then rejects the whole response.
func quoteETags(resp *http.Response) (*http.Response, error) {
	if etag := resp.Header.Get("ETag"); etag != "" {
		resp.Header.Set("ETag", quoteETag(etag))
	}
	if resp.StatusCode != http.StatusMultiStatus {
		return resp, nil
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	body = getETagElement.ReplaceAllFunc(body, func(match []byte) []byte {
		parts := getETagElement.FindSubmatch(match)
		var text struct {
			Value string `xml:",chardata"`
		}
		if xml.Unmarshal([]byte("<x>"+string(parts[2])+"</x>"), &text) != nil || strings.TrimSpace(text.Value) == "" {
			return match
		}
		var out bytes.Buffer
		out.Write(parts[1])
		_ = xml.EscapeText(&out, []byte(quoteETag(text.Value)))
		out.Write(parts[3])
		return out.Bytes()
	})
	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	resp.Header.Del("Content-Length")
	return resp, nil
}

func quoteETag(etag string) string {
	etag = strings.TrimSpace(etag)
	if len(etag) >= 2 && etag[0] == '"' && etag[len(etag)-1] == '"' {
		etag = etag[1 : len(etag)-1]
	}
	return strconv.Quote(etag)
}

// SameCredentialOrigin reports whether a password stored for one URL may be sent to another:
// same scheme, host and port, so it follows neither a new server nor a downgrade to http.
func SameCredentialOrigin(stored, target *url.URL) bool {
	port := func(u *url.URL) string {
		if p := u.Port(); p != "" {
			return p
		}
		return strings.ToLower(u.Scheme)
	}
	return strings.EqualFold(stored.Scheme, target.Scheme) &&
		strings.EqualFold(stored.Hostname(), target.Hostname()) &&
		port(stored) == port(target)
}
