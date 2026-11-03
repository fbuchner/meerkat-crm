package httputil

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// documentationIP is TEST-NET-3 (RFC 5737): public as far as IsPrivateIP is concerned, and
// never dialled by these tests. FetchImageFromURL always dials through the guard, so tests of
// what happens after connecting use fetchImage with a test transport instead.
const documentationIP = "203.0.113.10"

// useStubDNS swaps net.DefaultResolver for one that answers A queries with answer(name, n),
// where n counts the A queries seen so far for that name, so hostname handling can be tested
// without real DNS. Returning nil gives NXDOMAIN; AAAA queries get an empty answer for names
// that answer(name, -1) knows, and NXDOMAIN otherwise. Match names exactly: the resolver also
// tries the host's search domains, and those variants count as separate names. Tests in this
// package must not be parallel.
func useStubDNS(t *testing.T, answer func(name string, query int) []net.IP) {
	t.Helper()

	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	go func() {
		queries := map[string]int{}
		buf := make([]byte, 1500)
		for {
			n, addr, err := conn.ReadFrom(buf)
			if err != nil {
				return
			}
			name, qtype, questionEnd, ok := parseDNSQuestion(buf[:n])
			if !ok {
				continue
			}
			var ips []net.IP
			if qtype == 1 { // A
				ips = answer(name, queries[name])
				queries[name]++
			} else if answer(name, -1) != nil {
				ips = []net.IP{} // the name exists, it just has no records of this type
			}
			conn.WriteTo(buildDNSResponse(buf[:questionEnd], ips), addr)
		}
	}()

	original := net.DefaultResolver
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "udp", conn.LocalAddr().String())
		},
	}
	t.Cleanup(func() { net.DefaultResolver = original })
}

// parseDNSQuestion reads the single question of a query. questionEnd is the offset just past it.
func parseDNSQuestion(msg []byte) (name string, qtype uint16, questionEnd int, ok bool) {
	if len(msg) < 12 {
		return "", 0, 0, false
	}
	var labels []string
	i := 12
	for i < len(msg) && msg[i] != 0 {
		l := int(msg[i])
		if i+1+l > len(msg) {
			return "", 0, 0, false
		}
		labels = append(labels, string(msg[i+1:i+1+l]))
		i += 1 + l
	}
	if i+5 > len(msg) {
		return "", 0, 0, false
	}
	return strings.ToLower(strings.Join(labels, ".")), binary.BigEndian.Uint16(msg[i+1:]), i + 5, true
}

// buildDNSResponse answers query (header and question): nil ips is NXDOMAIN, otherwise one A
// record for each IP.
func buildDNSResponse(query []byte, ips []net.IP) []byte {
	resp := append([]byte(nil), query...)
	resp[2] = 0x84 | (query[2] & 0x01) // response, authoritative, keep RD
	resp[3] = 0x80                     // recursion available
	if ips == nil {
		resp[3] |= 3 // NXDOMAIN
	}
	binary.BigEndian.PutUint16(resp[6:], uint16(len(ips))) // ANCOUNT
	binary.BigEndian.PutUint16(resp[8:], 0)                // NSCOUNT
	// The Go resolver sends an EDNS OPT record after the question; it is not echoed back.
	binary.BigEndian.PutUint16(resp[10:], 0) // ARCOUNT
	for _, ip := range ips {
		resp = append(resp, 0xc0, 0x0c) // pointer to the question name
		resp = append(resp, 0, 1, 0, 1) // type A, class IN
		resp = append(resp, 0, 0, 0, 1) // TTL
		resp = append(resp, 0, 4)       // RDLENGTH
		resp = append(resp, ip.To4()...)
	}
	return resp
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip      string
		private bool
	}{
		{"127.0.0.1", true},
		{"127.5.5.5", true},
		{"::1", true},
		{"::ffff:127.0.0.1", true}, // IPv4-mapped form must not slip past the IPv4 checks
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"192.168.1.1", true},
		{"::ffff:192.168.1.1", true},
		{"fc00::1", true},
		{"fd00:ec2::254", true}, // AWS IPv6 metadata endpoint
		{"169.254.169.254", true},
		{"::ffff:169.254.169.254", true},
		{"fe80::1", true},
		{"224.0.0.1", true},
		{"ff02::1", true},
		{"0.0.0.0", true},
		{"::", true},
		{"100.64.0.1", true},
		{"100.100.100.200", true}, // Alibaba metadata endpoint
		{"100.127.255.255", true},

		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"172.15.255.255", false},
		{"172.32.0.1", false},
		{"100.63.255.255", false},
		{"100.128.0.1", false},
		{"2606:4700:4700::1111", false},
		{"64:ff9b::808:808", false}, // NAT64 embedding 8.8.8.8, which DNS64 networks resolve to
		{"2002:808:808::", false},   // 6to4 embedding 8.8.8.8
	}
	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			require.NotNil(t, ip)
			assert.Equal(t, tt.private, IsPrivateIP(ip))
		})
	}

	t.Run("nil is treated as private", func(t *testing.T) {
		assert.True(t, IsPrivateIP(nil))
	})
}

func TestIsPrivateIP_ReservedRangesThatReachInternalHosts(t *testing.T) {
	for _, s := range []string{
		"64:ff9b::a9fe:a9fe", // NAT64 embedding 169.254.169.254
		"64:ff9b::7f00:1",    // NAT64 embedding 127.0.0.1
		"2002:7f00:1::",      // 6to4 embedding 127.0.0.1
		"::7f00:1",           // IPv4-compatible embedding 127.0.0.1
		"::ffff:0:a00:1",     // IPv4-translated embedding 10.0.0.1
		"fec0::1",            // site-local
		"0.0.0.1",            // "this network"
		"192.0.0.192",        // IETF protocol assignments
		"198.18.0.1",         // benchmarking
		"240.0.0.1",          // reserved
		"255.255.255.255",    // broadcast
		"239.1.1.1",          // multicast
	} {
		assert.True(t, IsPrivateIP(net.ParseIP(s)), s)
	}
}

func TestValidateURLForSSRF_RejectsBeforeAnyLookup(t *testing.T) {
	useStubDNS(t, func(name string, query int) []net.IP {
		t.Errorf("unexpected DNS lookup for %q", name)
		return nil
	})

	tests := []struct {
		name    string
		url     string
		wantErr string
	}{
		{"unparseable", "http://bad host:%zz/", "invalid URL format"},
		{"file scheme", "file:///etc/passwd", "only http and https"},
		{"ftp scheme", "ftp://example.com/a.png", "only http and https"},
		{"gopher scheme", "gopher://example.com/", "only http and https"},
		{"javascript scheme", "javascript:alert(1)", "only http and https"},
		{"no scheme", "example.com/a.png", "only http and https"},
		{"no host", "http:///a.png", "URL must have a host"},
		{"port only", "http://:8080/a.png", "URL must have a host"},
		{"localhost", "http://localhost/a.png", "internal hosts"},
		{"localhost in capitals", "http://LOCALHOST/a.png", "internal hosts"},
		{"localhost with port", "http://localhost:8080/a.png", "internal hosts"},
		{"credentials before localhost", "http://public.example@localhost/a.png", "internal hosts"},
		{"loopback literal", "http://127.0.0.1/a.png", "internal hosts"},
		{"unspecified literal", "http://0.0.0.0/a.png", "internal hosts"},
		{"IPv6 loopback literal", "http://[::1]/a.png", "internal hosts"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := validateURLForSSRF(tt.url)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assert.Nil(t, parsed)
		})
	}
}

func TestValidateURLForSSRF_RejectsInternalIPLiterals(t *testing.T) {
	for _, u := range []string{
		"http://10.0.0.1/a.png",
		"http://172.16.0.1/a.png",
		"http://192.168.1.1/a.png",
		"http://169.254.169.254/latest/meta-data/",
		"http://100.100.100.200/latest/meta-data/",
		"http://127.0.0.2/a.png",
		"http://[::ffff:127.0.0.1]/a.png",
		"http://[::ffff:169.254.169.254]/a.png",
		"http://[fd00:ec2::254]/a.png",
		"http://[fe80::1%25en0]/a.png",
	} {
		t.Run(u, func(t *testing.T) {
			_, err := validateURLForSSRF(u)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "internal IP")
		})
	}
}

func TestValidateURLForSSRF_ChecksEveryResolvedAddress(t *testing.T) {
	useStubDNS(t, func(name string, query int) []net.IP {
		switch name {
		case "public.example.test":
			return []net.IP{net.ParseIP(documentationIP)}
		case "internal.example.test":
			return []net.IP{net.ParseIP("10.0.0.5")}
		case "metadata.example.test":
			return []net.IP{net.ParseIP("169.254.169.254")}
		case "mixed.example.test":
			return []net.IP{net.ParseIP(documentationIP), net.ParseIP("192.168.0.1")}
		}
		return nil
	})

	t.Run("public name is allowed and keeps its path", func(t *testing.T) {
		parsed, err := validateURLForSSRF("https://public.example.test/a.png?size=1")
		require.NoError(t, err)
		assert.Equal(t, "public.example.test", parsed.Hostname())
		assert.Equal(t, "/a.png", parsed.Path)
	})

	for _, tt := range []struct{ name, host string }{
		{"name resolving to a private address", "internal.example.test"},
		{"name resolving to the metadata address", "metadata.example.test"},
		{"public address alongside a private one", "mixed.example.test"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateURLForSSRF("https://" + tt.host + "/a.png")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "internal IP")
		})
	}

	t.Run("unresolvable name", func(t *testing.T) {
		_, err := validateURLForSSRF("https://nothing.example.test/a.png")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve")
	})
}

func TestFetchImageFromURL_DoesNotContactLoopbackServer(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("png"))
	}))
	defer server.Close()

	// httptest binds loopback, which is exactly what the guard must refuse.
	body, contentType, err := FetchImageFromURL(server.URL + "/a.png")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "internal hosts")
	assert.Nil(t, body)
	assert.Empty(t, contentType)
	assert.Zero(t, hits.Load())
}

func TestFetchImageFromURL_StripsWhitespaceBeforeValidating(t *testing.T) {
	// Google's VCF exports fold URLs across lines. The cleanup must happen before the
	// guard runs, or whitespace could be used to disguise an internal host.
	for _, u := range []string{
		"http://local host/a.png",
		"http://local\nhost/a.png",
		"http://local\r\nhost/a.png",
		" http://localhost/a.png ",
	} {
		t.Run(u, func(t *testing.T) {
			_, _, err := FetchImageFromURL(u)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "internal hosts")
		})
	}
}

func TestFetchImageFromURL_RejectsDisallowedSchemes(t *testing.T) {
	_, _, err := FetchImageFromURL("file:///etc/passwd")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only http and https")
}

// The first lookup passes validation and the connection-time lookup returns something else.
// The check at dial time is what stops a DNS rebinding attack, because validation alone
// would have let the request through.
func TestFetchImageFromURL_BlocksDNSRebindingAtDialTime(t *testing.T) {
	useStubDNS(t, func(name string, query int) []net.IP {
		if name != "rebind.example.test" {
			return nil
		}
		if query <= 0 {
			return []net.IP{net.ParseIP(documentationIP)}
		}
		return []net.IP{net.ParseIP("127.0.0.1")}
	})

	body, _, err := FetchImageFromURL("http://rebind.example.test/a.png")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "internal IP")
	assert.Nil(t, body)
}

func TestFetchImageFromURL_FailsWhenNameStopsResolvingAtDialTime(t *testing.T) {
	useStubDNS(t, func(name string, query int) []net.IP {
		if name == "vanish.example.test" && query <= 0 {
			return []net.IP{net.ParseIP(documentationIP)}
		}
		return nil
	})

	_, _, err := FetchImageFromURL("http://vanish.example.test/a.png")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve")
}

// roundTripFunc lets the client seam serve responses without a network or the
// dial-time guard, so the response handling can be tested on its own.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func imageResponse(req *http.Request, status int, contentType string, body io.Reader) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     http.Header{"Content-Type": {contentType}},
		Body:       io.NopCloser(body),
		Request:    req,
	}
}

func redirectResponse(req *http.Request, location string) *http.Response {
	resp := imageResponse(req, http.StatusFound, "text/plain", strings.NewReader(""))
	resp.Header.Set("Location", location)
	return resp
}

func TestFetchImage_ResponseHandling(t *testing.T) {
	const maxSize = 10 * 1024 * 1024
	tests := []struct {
		name        string
		status      int
		contentType string
		size        int
		wantErr     string
	}{
		{name: "returns the image and its type", status: http.StatusOK, contentType: "image/png", size: 16},
		{name: "accepts exactly the size limit", status: http.StatusOK, contentType: "image/png", size: maxSize},
		{name: "rejects one byte over the limit", status: http.StatusOK, contentType: "image/png", size: maxSize + 1, wantErr: "too large"},
		{name: "rejects a non-image", status: http.StatusOK, contentType: "text/html", size: 16, wantErr: "does not point to an image"},
		{name: "rejects a failed response", status: http.StatusNotFound, contentType: "image/png", size: 16, wantErr: "remote server returned"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return imageResponse(req, tt.status, tt.contentType, bytes.NewReader(make([]byte, tt.size))), nil
			})}

			body, contentType, err := fetchImage(client, "http://93.184.216.34/photo.png")

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, body)
				return
			}
			require.NoError(t, err)
			assert.Len(t, body, tt.size)
			assert.Equal(t, tt.contentType, contentType)
		})
	}
}

func TestFetchImage_SendsUserAgent(t *testing.T) {
	var userAgent string
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		userAgent = req.Header.Get("User-Agent")
		return imageResponse(req, http.StatusOK, "image/png", strings.NewReader("x")), nil
	})}

	_, _, err := fetchImage(client, "http://93.184.216.34/photo.png")

	require.NoError(t, err)
	assert.Contains(t, userAgent, "MeerkatCRM")
}

func TestFetchImage_Redirects(t *testing.T) {
	t.Run("follows a redirect to a public address", func(t *testing.T) {
		client := &http.Client{
			CheckRedirect: checkImageRedirect,
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Path == "/start" {
					return redirectResponse(req, "http://93.184.216.35/photo.png"), nil
				}
				return imageResponse(req, http.StatusOK, "image/png", strings.NewReader("image")), nil
			}),
		}

		body, _, err := fetchImage(client, "http://93.184.216.34/start")

		require.NoError(t, err)
		assert.Equal(t, "image", string(body))
	})

	for _, target := range []string{
		"http://169.254.169.254/latest/meta-data/",
		"http://127.0.0.1/admin",
		"http://localhost/admin",
		"http://10.0.0.1/",
		"file:///etc/passwd",
	} {
		t.Run("refuses a redirect to "+target, func(t *testing.T) {
			var requested []string
			client := &http.Client{
				CheckRedirect: checkImageRedirect,
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					requested = append(requested, req.URL.String())
					return redirectResponse(req, target), nil
				}),
			}

			_, _, err := fetchImage(client, "http://93.184.216.34/start")

			require.ErrorContains(t, err, "redirect to disallowed location")
			assert.Equal(t, []string{"http://93.184.216.34/start"}, requested)
		})
	}

	t.Run("stops after three redirects", func(t *testing.T) {
		hops := 0
		client := &http.Client{
			CheckRedirect: checkImageRedirect,
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				hops++
				return redirectResponse(req, "http://93.184.216.34/next"), nil
			}),
		}

		_, _, err := fetchImage(client, "http://93.184.216.34/start")

		require.ErrorContains(t, err, "too many redirects")
		assert.Equal(t, 3, hops)
	})
}

func TestFetchImage_TimesOutOnASilentServer(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	defer server.Close()
	defer close(release)

	_, _, err := fetchImage(&http.Client{Timeout: 50 * time.Millisecond}, server.URL)

	var netErr net.Error
	require.ErrorAs(t, err, &netErr)
	assert.True(t, netErr.Timeout())
}

func TestNewSafeClient_GuardsAreWiredIn(t *testing.T) {
	client := newSafeClient()

	assert.Equal(t, 15*time.Second, client.Timeout)
	require.NotNil(t, client.CheckRedirect)
	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	_, err := transport.DialContext(context.Background(), "tcp", "127.0.0.1:80")
	assert.ErrorContains(t, err, "internal IP")
}
