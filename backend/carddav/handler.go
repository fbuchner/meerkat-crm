package carddav

import (
	"net/http"

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

// GinHandler returns a Gin handler function that wraps the CardDAV handler
func (h *Handler) GinHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user info from Gin context (set by BasicAuthMiddleware)
		userID, _ := c.Get("userID")
		username, _ := c.Get("username")

		// Set in request context for the backend
		ctx := ContextWithUser(c.Request.Context(), userID.(uint), username.(string), h.db, h.photoDir)
		c.Request = c.Request.WithContext(ctx)

		h.handler.ServeHTTP(c.Writer, c.Request)
	}
}

// WellKnownRedirect handles the /.well-known/carddav discovery redirect
func WellKnownRedirect(c *gin.Context) {
	// Redirect to the CardDAV root
	c.Redirect(http.StatusPermanentRedirect, "/carddav/")
}
