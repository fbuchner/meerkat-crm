package monica

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

// Sentinel errors; the controller maps these to user-facing apperrors.
var (
	ErrUnauthorized   = errors.New("monica: authentication failed")
	ErrUnreachable    = errors.New("monica: host could not be reached")
	ErrInvalidURL     = errors.New("monica: base URL is invalid")
	ErrPrivateAddress = errors.New("monica: URL resolves to a private address")
	ErrInvalidData    = errors.New("monica: response could not be parsed")
	ErrNoPhoto        = errors.New("monica: contact has no retrievable photo")
)

const (
	requestTimeout   = 30 * time.Second
	maxRetries       = 3
	maxRetryAfter    = 90 * time.Second
	maxResponseBytes = 10 << 20 // per page/avatar
	pageSize         = 100
	// maxPages bounds the paging loop in case an endpoint keeps returning
	// full pages while ignoring the page parameter.
	maxPages = 1000
)

// Client is a rate-limited, read-only Monica API client. Monica enforces
// 60 requests/minute, so requests are throttled to ~1/second.
type Client struct {
	baseURL *url.URL
	token   string
	http    *http.Client
	limiter *rate.Limiter
}

// newLimiter builds the client-side throttle; swapped out by
// DisableRateLimitForTesting so test suites don't wait a second per request.
var newLimiter = func() *rate.Limiter { return rate.NewLimiter(rate.Every(time.Second), 1) }

// DisableRateLimitForTesting removes client-side throttling for all clients
// created afterwards. Test helper only.
func DisableRateLimitForTesting() {
	newLimiter = func() *rate.Limiter { return rate.NewLimiter(rate.Inf, 0) }
}

// NewClient validates and normalizes the instance URL (the "/api" path is
// appended when missing) and builds a client. When blockPrivate is set,
// connections to private/loopback addresses are refused (SSRF protection for
// cloud deployments; self-hosted Monica usually lives on a LAN, so opt-in).
func NewClient(baseURL, token string, blockPrivate bool) (*Client, error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, ErrInvalidURL
	}
	u.Path = strings.TrimRight(u.Path, "/")
	if !strings.HasSuffix(u.Path, "/api") {
		u.Path += "/api"
	}
	u.RawQuery = ""
	u.Fragment = ""

	transport := &http.Transport{}
	if blockPrivate {
		transport.DialContext = privateBlockingDialContext
	}
	return &Client{
		baseURL: u,
		token:   token,
		http: &http.Client{
			Timeout:   requestTimeout,
			Transport: transport,
		},
		limiter: newLimiter(),
	}, nil
}

func privateBlockingDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("%w: could not resolve host", ErrUnreachable)
	}
	var safeIP net.IP
	for _, ip := range ips {
		if !(ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()) {
			safeIP = ip
			break
		}
	}
	if safeIP == nil {
		return nil, ErrPrivateAddress
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 10 * time.Second}
	return dialer.DialContext(ctx, network, net.JoinHostPort(safeIP.String(), port))
}

// get performs one rate-limited GET against the API and decodes the JSON
// response into out, retrying 429s per Retry-After and 5xx once.
func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	u := *c.baseURL
	u.Path = c.baseURL.Path + path
	if query != nil {
		u.RawQuery = query.Encode()
	}

	for attempt := 0; ; attempt++ {
		if err := c.limiter.Wait(ctx); err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidURL, err)
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Accept", "application/json")

		resp, err := c.http.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if errors.Is(err, ErrPrivateAddress) {
				return ErrPrivateAddress
			}
			return fmt.Errorf("%w: %v", ErrUnreachable, err)
		}

		retry, err := c.handleResponse(resp, out)
		if err == nil {
			return nil
		}
		if !retry || attempt >= maxRetries {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryDelay(resp)):
		}
	}
}

// handleResponse decodes a successful response or classifies the failure;
// retry reports whether the request may be attempted again.
func (c *Client) handleResponse(resp *http.Response, out any) (retry bool, err error) {
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return false, ErrUnauthorized
	case resp.StatusCode == http.StatusTooManyRequests:
		return true, fmt.Errorf("%w: rate limited", ErrUnreachable)
	case resp.StatusCode >= 500:
		return true, fmt.Errorf("%w: server error %d", ErrUnreachable, resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return false, fmt.Errorf("%w: unexpected status %d", ErrInvalidData, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return false, fmt.Errorf("%w: %v", ErrInvalidData, err)
	}
	return false, nil
}

func retryDelay(resp *http.Response) time.Duration {
	delay := 2 * time.Second
	if s := resp.Header.Get("Retry-After"); s != "" {
		if secs, err := strconv.Atoi(s); err == nil && secs >= 0 {
			delay = time.Duration(secs) * time.Second
		}
	}
	if delay > maxRetryAfter {
		delay = maxRetryAfter
	}
	return delay
}

// TestConnection verifies the URL and token with a minimal request.
func (c *Client) TestConnection(ctx context.Context) error {
	var page Paged[json.RawMessage]
	return c.get(ctx, "/contacts", url.Values{"limit": {"1"}}, &page)
}

// CountEntities reads meta.total from each list endpoint (one request each).
func (c *Client) CountEntities(ctx context.Context) (EntityCounts, error) {
	counts := EntityCounts{}
	for _, e := range []struct {
		path string
		dst  *int
	}{
		{"/contacts", &counts.Contacts},
		{"/activities", &counts.Activities},
		{"/notes", &counts.Notes},
		{"/reminders", &counts.Reminders},
		{"/calls", &counts.Calls},
		{"/tasks", &counts.Tasks},
		{"/gifts", &counts.Gifts},
		{"/debts", &counts.Debts},
	} {
		var page Paged[json.RawMessage]
		if err := c.get(ctx, e.path, url.Values{"limit": {"1"}}, &page); err != nil {
			return counts, err
		}
		*e.dst = page.Meta.Total
	}
	return counts, nil
}

// fetchAll walks every page of a list endpoint, invoking progress after each
// page with (items fetched, meta total).
func fetchAll[T any](ctx context.Context, c *Client, path string, extra url.Values, progress func(done, total int)) ([]T, error) {
	var all []T
	for page := 1; page <= maxPages; page++ {
		query := url.Values{"limit": {strconv.Itoa(pageSize)}, "page": {strconv.Itoa(page)}}
		for k, vs := range extra {
			query[k] = vs
		}
		var resp Paged[T]
		if err := c.get(ctx, path, query, &resp); err != nil {
			return nil, err
		}
		all = append(all, resp.Data...)
		if progress != nil {
			progress(len(all), resp.Meta.Total)
		}
		if len(resp.Data) == 0 {
			return all, nil
		}
		// Nested endpoints (and some Monica versions) omit pagination meta
		// entirely; a short page is then the only end-of-list signal.
		if resp.Meta.LastPage > 0 {
			if resp.Meta.LastPage <= page {
				return all, nil
			}
		} else if len(resp.Data) < pageSize {
			return all, nil
		}
	}
	return all, nil
}

// FetchAllContacts pulls every contact including its contact fields.
func (c *Client) FetchAllContacts(ctx context.Context, progress func(done, total int)) ([]Contact, error) {
	return fetchAll[Contact](ctx, c, "/contacts", url.Values{"with": {"contactfields"}}, progress)
}

func (c *Client) FetchAllActivities(ctx context.Context, progress func(done, total int)) ([]Activity, error) {
	return fetchAll[Activity](ctx, c, "/activities", nil, progress)
}

func (c *Client) FetchAllNotes(ctx context.Context, progress func(done, total int)) ([]Note, error) {
	return fetchAll[Note](ctx, c, "/notes", nil, progress)
}

func (c *Client) FetchAllReminders(ctx context.Context, progress func(done, total int)) ([]Reminder, error) {
	return fetchAll[Reminder](ctx, c, "/reminders", nil, progress)
}

func (c *Client) FetchAllCalls(ctx context.Context, progress func(done, total int)) ([]Call, error) {
	return fetchAll[Call](ctx, c, "/calls", nil, progress)
}

func (c *Client) FetchAllTasks(ctx context.Context, progress func(done, total int)) ([]Task, error) {
	return fetchAll[Task](ctx, c, "/tasks", nil, progress)
}

func (c *Client) FetchAllGifts(ctx context.Context, progress func(done, total int)) ([]Gift, error) {
	return fetchAll[Gift](ctx, c, "/gifts", nil, progress)
}

func (c *Client) FetchAllDebts(ctx context.Context, progress func(done, total int)) ([]Debt, error) {
	return fetchAll[Debt](ctx, c, "/debts", nil, progress)
}

// FetchContactRelationships lists the relationships of one contact — Monica
// has no account-wide relationships endpoint, so this is called per contact.
func (c *Client) FetchContactRelationships(ctx context.Context, contactID int) ([]Relationship, error) {
	return fetchAll[Relationship](ctx, c, fmt.Sprintf("/contacts/%d/relationships", contactID), nil, nil)
}

// FetchContactPhoto downloads a contact's avatar through the photos endpoint.
// Cannot use Contact.Information.Avatar.Source "photo" (web route, not API)
func (c *Client) FetchContactPhoto(ctx context.Context, contactID int, avatarURL string) (data []byte, mediaType string, err error) {
	photos, err := fetchAll[Photo](ctx, c, fmt.Sprintf("/contacts/%d/photos", contactID), nil, nil)
	if err != nil {
		return nil, "", err
	}
	photo := selectAvatarPhoto(photos, avatarURL)
	if photo == nil {
		return nil, "", fmt.Errorf("%w: %d photo(s) returned, none matching the avatar", ErrNoPhoto, len(photos))
	}
	// Keep the payload of "data:<media type>;base64,<payload>"
	_, payload, found := strings.Cut(photo.DataURL, ",")
	if !found {
		return nil, "", fmt.Errorf("%w: photo is not a data URL", ErrInvalidData)
	}
	data, err = base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, "", fmt.Errorf("%w: photo payload: %v", ErrInvalidData, err)
	}
	return data, photo.MimeType, nil
}

// selectAvatarPhoto picks the record the avatar URL refers to, falling back to
// the sole photo when the filenames do not line up.
func selectAvatarPhoto(photos []Photo, avatarURL string) *Photo {
	// Match on the URL path only: Monica appends cache-busting queries.
	if u, err := url.Parse(avatarURL); err == nil && u.Path != "" {
		base := path.Base(u.Path)
		for i := range photos {
			if strings.HasSuffix(photos[i].NewFilename, base) {
				return &photos[i]
			}
		}
	}
	if len(photos) == 1 {
		return &photos[0]
	}
	return nil
}

// FetchAvatar downloads an avatar straight from its URL, which works for
// third-party hosts such as gravatar. Monica's own /store URLs need
// FetchContactPhoto instead. The bearer token is only sent when the avatar is
// served by the Monica instance itself, never to third-party hosts (gravatar,
// presigned S3 URLs).
func (c *Client) FetchAvatar(ctx context.Context, avatarURL string) (data []byte, mediaType string, err error) {
	u, err := url.Parse(avatarURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, "", ErrInvalidURL
	}
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", ErrInvalidURL
	}
	if u.Host == c.baseURL.Host {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("%w: avatar status %d", ErrUnreachable, resp.StatusCode)
	}
	data, err = io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	return data, resp.Header.Get("Content-Type"), nil
}
