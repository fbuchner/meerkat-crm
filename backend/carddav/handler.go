package carddav

import (
	"net/http"
	"strings"

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

// charsetResponseWriter wraps http.ResponseWriter to ensure text/vcard responses
// include charset=utf-8, preventing iOS from misinterpreting UTF-8 as Latin-1.
type charsetResponseWriter struct {
	http.ResponseWriter
}

func (w *charsetResponseWriter) WriteHeader(statusCode int) {
	w.fixContentType()
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *charsetResponseWriter) Write(b []byte) (int, error) {
	w.fixContentType()
	return w.ResponseWriter.Write(b)
}

func (w *charsetResponseWriter) fixContentType() {
	ct := w.Header().Get("Content-Type")
	if strings.HasPrefix(ct, "text/vcard") && !strings.Contains(ct, "charset") {
		w.Header().Set("Content-Type", ct+"; charset=utf-8")
	}
}

// GinHandler returns a Gin handler function that wraps the CardDAV handler
func (h *Handler) GinHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user info from Gin context (set by BasicAuthMiddleware)
		userID, _ := c.Get("userID")
		username, _ := c.Get("username")

		// Set in request context for the backend
		ctx := ContextWithUser(c.Request.Context(), userID.(uint), username.(string), h.db, h.photoDir)
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

		h.handler.ServeHTTP(&charsetResponseWriter{c.Writer}, c.Request)
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
	c.Redirect(http.StatusPermanentRedirect, "/carddav/")
}
