// Package httputil provides shared, SSRF-guarded HTTP-fetch helpers (used by
// the photo-proxy and photo-import paths) that refuse to fetch from private/
// loopback/link-local addresses or cloud-metadata endpoints.
package httputil

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

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

	// Reject if *any* answer is non-public, not merely if all of them are: a
	// host that resolves to a mix of public and internal addresses has no
	// legitimate reason to be fetched here.
	for _, ip := range ips {
		if !IsPublicIP(ip) {
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

	// Validate IP addresses at connection time, pinning the resolved address,
	// so DNS rebinding between validation and dial cannot redirect us inward.
	safeDialContext := SafeDialContext(
		errors.New("failed to resolve hostname"),
		errors.New("access to internal IP addresses is not allowed"),
	)

	// Create HTTP client with custom transport that validates IPs at connection time
	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			DialContext: safeDialContext,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Validate redirect target URL format
			_, err := validateURLForSSRF(req.URL.String())
			if err != nil {
				return errors.New("redirect to disallowed location")
			}
			if len(via) >= 3 {
				return errors.New("too many redirects")
			}
			return nil
		},
	}

	// Fetch the image using the validated URL
	req, err := http.NewRequest("GET", parsedURL.String(), nil)
	if err != nil {
		return nil, "", err
	}

	// Set a user agent to avoid being blocked by some servers
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MycorrhizalCRM/1.0)")

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
