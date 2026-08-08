package monica

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

// newTestClient builds a client against a test server with rate limiting off.
func newTestClient(t *testing.T, serverURL string) *Client {
	t.Helper()
	c, err := NewClient(serverURL, "test-token", false)
	assert.NoError(t, err)
	c.limiter = rate.NewLimiter(rate.Inf, 0)
	return c
}

func pagedJSON(items []map[string]any, page, lastPage, total int) []byte {
	body, _ := json.Marshal(map[string]any{
		"data": items,
		"meta": map[string]int{"current_page": page, "last_page": lastPage, "total": total},
	})
	return body
}

func TestNewClientURLNormalization(t *testing.T) {
	cases := map[string]string{
		"https://monica.example.com":      "https://monica.example.com/api",
		"https://monica.example.com/":     "https://monica.example.com/api",
		"https://monica.example.com/api":  "https://monica.example.com/api",
		"https://monica.example.com/api/": "https://monica.example.com/api",
		"http://192.168.1.5:8080":         "http://192.168.1.5:8080/api",
	}
	for input, want := range cases {
		c, err := NewClient(input, "tok", false)
		assert.NoError(t, err, input)
		assert.Equal(t, want, c.baseURL.String(), input)
	}

	for _, invalid := range []string{"", "ftp://monica.example.com", "not a url", "monica.example.com"} {
		_, err := NewClient(invalid, "tok", false)
		assert.ErrorIs(t, err, ErrInvalidURL, invalid)
	}
}

func TestFetchAllContactsPaginationAndAuth(t *testing.T) {
	var sawAuth, sawWith bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/contacts", r.URL.Path)
		sawAuth = r.Header.Get("Authorization") == "Bearer test-token"
		sawWith = r.URL.Query().Get("with") == "contactfields"
		page := r.URL.Query().Get("page")
		switch page {
		case "1":
			w.Write(pagedJSON([]map[string]any{{"id": 1, "first_name": "Ada"}, {"id": 2, "first_name": "Bob"}}, 1, 2, 3))
		case "2":
			w.Write(pagedJSON([]map[string]any{{"id": 3, "first_name": "Cat"}}, 2, 2, 3))
		default:
			t.Errorf("unexpected page %q", page)
		}
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	var progressCalls []int
	contacts, err := c.FetchAllContacts(context.Background(), func(done, total int) {
		progressCalls = append(progressCalls, done)
	})
	assert.NoError(t, err)
	assert.Len(t, contacts, 3)
	assert.Equal(t, "Ada", contacts[0].FirstName)
	assert.Equal(t, 3, contacts[2].ID)
	assert.True(t, sawAuth)
	assert.True(t, sawWith)
	assert.Equal(t, []int{2, 3}, progressCalls)
}

func TestGetRetriesOn429(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write(pagedJSON(nil, 1, 1, 0))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	assert.NoError(t, c.TestConnection(context.Background()))
	assert.Equal(t, 2, attempts)
}

func TestGetGivesUpAfterMaxRetries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	err := c.TestConnection(context.Background())
	assert.ErrorIs(t, err, ErrUnreachable)
	assert.Equal(t, maxRetries+1, attempts)
}

func TestGetUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	assert.ErrorIs(t, c.TestConnection(context.Background()), ErrUnauthorized)
}

func TestGetUnreachableHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close() // connection refused

	c := newTestClient(t, server.URL)
	assert.ErrorIs(t, c.TestConnection(context.Background()), ErrUnreachable)
}

func TestGetInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<html>login page</html>")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	assert.ErrorIs(t, c.TestConnection(context.Background()), ErrInvalidData)
}

func TestCountEntities(t *testing.T) {
	totals := map[string]int{
		"/api/contacts": 12, "/api/activities": 4, "/api/notes": 7, "/api/reminders": 2,
		"/api/calls": 1, "/api/tasks": 0, "/api/gifts": 3, "/api/debts": 5,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		total, ok := totals[r.URL.Path]
		if !ok {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Write(pagedJSON(nil, 1, 1, total))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	counts, err := c.CountEntities(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, EntityCounts{Contacts: 12, Activities: 4, Notes: 7, Reminders: 2, Calls: 1, Tasks: 0, Gifts: 3, Debts: 5}, counts)
}

func TestFetchContactRelationships(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/contacts/7/relationships", r.URL.Path)
		w.Write(pagedJSON([]map[string]any{{
			"relationship_type": map[string]any{"name": "father"},
			"of_contact":        map[string]any{"id": 9, "first_name": "Jane"},
		}}, 1, 1, 1))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	rels, err := c.FetchContactRelationships(context.Background(), 7)
	assert.NoError(t, err)
	assert.Len(t, rels, 1)
	assert.Equal(t, "father", rels[0].RelationshipType.Name)
	assert.Equal(t, 9, rels[0].OfContact.ID)
}

func TestFetchAvatarTokenScoping(t *testing.T) {
	var monicaAuth string
	monicaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		monicaAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write([]byte("jpeg-bytes"))
	}))
	defer monicaServer.Close()

	var thirdPartyAuth string
	thirdParty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		thirdPartyAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("png-bytes"))
	}))
	defer thirdParty.Close()

	c := newTestClient(t, monicaServer.URL)

	data, mediaType, err := c.FetchAvatar(context.Background(), monicaServer.URL+"/storage/avatar.jpg")
	assert.NoError(t, err)
	assert.Equal(t, "jpeg-bytes", string(data))
	assert.Equal(t, "image/jpeg", mediaType)
	assert.Equal(t, "Bearer test-token", monicaAuth)

	_, _, err = c.FetchAvatar(context.Background(), thirdParty.URL+"/avatar.png")
	assert.NoError(t, err)
	assert.Empty(t, thirdPartyAuth)
}

func TestGetRespectsContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := c.TestConnection(ctx)
	assert.True(t, errors.Is(err, context.Canceled) || errors.Is(err, ErrUnreachable))
}

// Finding #5: nested endpoints (and some Monica versions) omit pagination
// meta entirely, which used to truncate the list at the first page.
func TestFetchAllWithoutPaginationMeta(t *testing.T) {
	fullPage := make([]map[string]any, pageSize)
	for i := range fullPage {
		fullPage[i] = map[string]any{"id": i}
	}
	shortPage := []map[string]any{{"id": 999}}

	var requestedPages []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		requestedPages = append(requestedPages, page)
		// last_page = 0 stands in for "no pagination meta at all".
		if page == "1" {
			w.Write(pagedJSON(fullPage, 1, 0, 0))
			return
		}
		w.Write(pagedJSON(shortPage, 2, 0, 0))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	rels, err := c.FetchContactRelationships(context.Background(), 7)
	assert.NoError(t, err)
	assert.Len(t, rels, pageSize+1, "a full page with no meta must not end the walk")
	assert.Equal(t, []string{"1", "2"}, requestedPages)
}

// A short first page still ends the walk immediately.
func TestFetchAllStopsOnShortPageWithoutMeta(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write(pagedJSON([]map[string]any{{"id": 1}}, 1, 0, 0))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	rels, err := c.FetchContactRelationships(context.Background(), 7)
	assert.NoError(t, err)
	assert.Len(t, rels, 1)
	assert.Equal(t, 1, calls)
}

// An endpoint that ignores the page parameter must not loop forever.
func TestFetchAllStopsAtMaxPages(t *testing.T) {
	fullPage := make([]map[string]any, pageSize)
	for i := range fullPage {
		fullPage[i] = map[string]any{"id": i}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(pagedJSON(fullPage, 1, 0, 0))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	rels, err := c.FetchContactRelationships(context.Background(), 7)
	assert.NoError(t, err)
	assert.Len(t, rels, maxPages*pageSize)
}
