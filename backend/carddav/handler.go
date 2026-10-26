package carddav

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/carddav"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler wraps the go-webdav CardDAV handler for use with Gin
type Handler struct {
	backend  *Backend
	handler  *carddav.Handler
	db       *gorm.DB
	photoDir string
}

// NewHandler creates a new CardDAV handler
func NewHandler(db *gorm.DB, photoDir string) *Handler {
	backend := NewBackend(db, photoDir)
	handler := &carddav.Handler{
		Backend: backend,
		Prefix:  "/carddav",
	}

	return &Handler{
		backend:  backend,
		handler:  handler,
		db:       db,
		photoDir: photoDir,
	}
}

// lazyCircleFilter caches the user's circle filter for one request. It loads on
// first use, so discovery requests that never touch contacts skip the query.
type lazyCircleFilter struct {
	once   sync.Once
	filter *circleFilter
	err    error
}

// GinHandler returns a Gin handler function that wraps the CardDAV handler
func (h *Handler) GinHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user info from Gin context (set by BasicAuthMiddleware)
		userID, _ := c.Get("userID")
		username, _ := c.Get("username")

		// Set in request context for the backend
		ctx := ContextWithUser(c.Request.Context(), userID.(uint), username.(string), h.db, h.photoDir)
		ctx = context.WithValue(ctx, circleFilterKey, &lazyCircleFilter{})
		c.Request = c.Request.WithContext(ctx)

		// Handle principals endpoint specially for proper discovery
		if strings.HasPrefix(c.Request.URL.Path, "/carddav/principals/") {
			h.servePrincipal(c.Writer, c.Request, username.(string))
			return
		}

		// Handle root CardDAV path for discovery
		if c.Request.URL.Path == "/carddav/" || c.Request.URL.Path == "/carddav" {
			h.servePrincipal(c.Writer, c.Request, username.(string))
			return
		}

		h.handler.ServeHTTP(c.Writer, c.Request)
	}
}

// servePrincipal handles PROPFIND requests for principal discovery
func (h *Handler) servePrincipal(w http.ResponseWriter, r *http.Request, username string) {
	principalPath := "/carddav/principals/" + username + "/"
	homeSetPath := "/carddav/addressbooks/" + username + "/"

	webdav.ServePrincipal(w, r, &webdav.ServePrincipalOptions{
		CurrentUserPrincipalPath: principalPath,
		HomeSets: []webdav.BackendSuppliedHomeSet{
			carddav.NewAddressBookHomeSet(homeSetPath),
		},
		Capabilities: []webdav.Capability{
			carddav.CapabilityAddressBook,
		},
	})
}

// WellKnownRedirect handles the /.well-known/carddav discovery redirect
func WellKnownRedirect(c *gin.Context) {
	// Redirect to the CardDAV root
	c.Redirect(http.StatusMovedPermanently, "/carddav/")
}
