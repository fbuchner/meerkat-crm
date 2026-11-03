package httputil

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func mustCIDRs(cidrs ...string) []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(err)
		}
		nets = append(nets, ipNet)
	}
	return nets
}

// reservedRanges are special-purpose ranges that Go's Is* helpers leave out. The documentation
// ranges are not listed: they are not routable, so they cannot reach anything.
var reservedRanges = mustCIDRs(
	"0.0.0.0/8",       // "this network"
	"100.64.0.0/10",   // shared address space (RFC 6598), used by cloud providers for metadata endpoints
	"192.0.0.0/24",    // IETF protocol assignments
	"198.18.0.0/15",   // benchmarking
	"240.0.0.0/4",     // reserved, including broadcast
	"64:ff9b:1::/48",  // local-use NAT64
	"::ffff:0:0:0/96", // IPv4-translated (SIIT); not ::ffff:0:0/96, which is every mapped IPv4 address
	"fec0::/10",       // deprecated site-local, which Go's IsPrivate leaves out
	"100::/64",        // discard-only
	"2001::/32",       // Teredo, which tunnels to an address the prefix does not reveal
)

// embeddedIPv4 returns the IPv4 address an IPv6 transition address forwards to, or nil.
func embeddedIPv4(ip net.IP) net.IP {
	ip16 := ip.To16()
	if ip16 == nil || ip.To4() != nil {
		return nil
	}
	switch {
	case ip16[0] == 0x20 && ip16[1] == 0x02: // 6to4, 2002::/16
		return net.IPv4(ip16[2], ip16[3], ip16[4], ip16[5])
	case nat64Prefix.Contains(ip16), ipv4CompatiblePrefix.Contains(ip16):
		return net.IPv4(ip16[12], ip16[13], ip16[14], ip16[15])
	}
	return nil
}

var (
	nat64Prefix          = mustCIDRs("64:ff9b::/96")[0]
	ipv4CompatiblePrefix = mustCIDRs("::/96")[0]
)

// IsPrivateIP checks if an IP address is in a private/reserved range
func IsPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true
	}

	// Check for loopback
	if ip.IsLoopback() {
		return true
	}

	// Check for link-local (includes cloud metadata endpoint 169.254.169.254) and multicast
	if ip.IsLinkLocalUnicast() || ip.IsMulticast() {
		return true
	}

	// Check for private ranges
	if ip.IsPrivate() {
		return true
	}

	// Check for unspecified (0.0.0.0 or ::)
	if ip.IsUnspecified() {
		return true
	}

	for _, reserved := range reservedRanges {
		if reserved.Contains(ip) {
			return true
		}
	}

	// A transition address reaches whatever IPv4 address it embeds
	if embedded := embeddedIPv4(ip); embedded != nil {
		return IsPrivateIP(embedded)
	}

	return false
}

// validateURLForSSRF checks if a URL is safe to fetch (not pointing to internal resources)
func validateURLForSSRF(rawURL string) (*url.URL, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, errors.New("invalid URL format")
	}

	// Only allow http and https schemes
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, errors.New("only http and https URLs are allowed")
	}

	host := parsedURL.Hostname()
	if host == "" {
		return nil, errors.New("URL must have a host")
	}

	// Block common internal hostnames
	lowerHost := strings.ToLower(host)
	blockedHosts := []string{"localhost", "127.0.0.1", "0.0.0.0", "::1", "[::1]"}
	for _, blocked := range blockedHosts {
		if lowerHost == blocked {
			return nil, errors.New("access to internal hosts is not allowed")
		}
	}

	// Resolve the hostname to IP addresses
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, errors.New("failed to resolve hostname")
	}

	// Check all resolved IPs
	for _, ip := range ips {
		if IsPrivateIP(ip) {
			return nil, errors.New("access to internal IP addresses is not allowed")
		}
	}

	return parsedURL, nil
}

// FetchImageFromURL fetches an image from a URL with SSRF protection.
// Returns the image data, content type, and any error.
// The URL is sanitized to remove whitespace (handles Google VCF format).
func FetchImageFromURL(imageURL string) ([]byte, string, error) {
	// Clean URL - remove spaces and newlines (Google VCF format may have these)
	cleanURL := strings.ReplaceAll(imageURL, " ", "")
	cleanURL = strings.ReplaceAll(cleanURL, "\n", "")
	cleanURL = strings.ReplaceAll(cleanURL, "\r", "")

	// Validate the URL format and scheme
	parsedURL, err := validateURLForSSRF(cleanURL)
	if err != nil {
		return nil, "", err
	}

	return fetchImage(newSafeClient(), parsedURL.String())
}

// newSafeClient returns an HTTP client that refuses internal addresses, both at
// connection time and on redirects.
func newSafeClient() *http.Client {
	// Create a custom dialer that validates IP addresses at connection time
	// This prevents DNS rebinding/TOCTOU attacks
	safeDialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 10 * time.Second,
	}

	safeDialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		// Extract host from addr (format is host:port)
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		// Resolve the hostname to IP addresses
		ips, err := net.LookupIP(host)
		if err != nil {
			return nil, errors.New("failed to resolve hostname")
		}

		// Find a safe IP to connect to
		var safeIP net.IP
		for _, ip := range ips {
			if !IsPrivateIP(ip) {
				safeIP = ip
				break
			}
		}

		if safeIP == nil {
			return nil, errors.New("access to internal IP addresses is not allowed")
		}

		// Connect using the validated IP address directly
		safeAddr := net.JoinHostPort(safeIP.String(), port)
		return safeDialer.DialContext(ctx, network, safeAddr)
	}

	// Create HTTP client with custom transport that validates IPs at connection time
	return &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			DialContext: safeDialContext,
		},
		CheckRedirect: checkImageRedirect,
	}
}

// checkImageRedirect validates each redirect target and limits the number of hops.
func checkImageRedirect(req *http.Request, via []*http.Request) error {
	// Validate redirect target URL format
	_, err := validateURLForSSRF(req.URL.String())
	if err != nil {
		return errors.New("redirect to disallowed location")
	}
	if len(via) >= 3 {
		return errors.New("too many redirects")
	}
	return nil
}

// fetchImage downloads an already validated URL with the given client.
func fetchImage(client *http.Client, imageURL string) ([]byte, string, error) {
	// Fetch the image using the validated URL
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return nil, "", err
	}

	// Set a user agent to avoid being blocked by some servers
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MeerkatCRM/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", errors.New("failed to fetch image: remote server returned " + resp.Status)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return nil, "", errors.New("URL does not point to an image")
	}

	// Limit response size (10MB)
	const maxSize = 10 * 1024 * 1024
	limitedReader := io.LimitReader(resp.Body, maxSize+1)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, "", err
	}

	if len(body) > maxSize {
		return nil, "", errors.New("image is too large, maximum size is 10MB")
	}

	return body, contentType, nil
}
