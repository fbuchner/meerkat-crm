package carddav

import (
	"bytes"
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

// charsetResponseWriter wraps http.ResponseWriter to fix UTF-8 encoding issues with iOS:
//  1. Adds charset=utf-8 to text/vcard Content-Type headers (GET responses).
//  2. Buffers XML responses (PROPFIND/REPORT) and injects charset=utf-8 into all
//     text/vcard content-type references, so iOS correctly interprets the embedded
//     vCard data as UTF-8 rather than defaulting to Latin-1.
type charsetResponseWriter struct {
	http.ResponseWriter
	xmlBuf bytes.Buffer
	isXML  bool
}

func (w *charsetResponseWriter) WriteHeader(statusCode int) {
	w.fixVCardContentType()
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *charsetResponseWriter) Write(b []byte) (int, error) {
	w.fixVCardContentType()
	ct := w.Header().Get("Content-Type")
	if !w.isXML && (strings.HasPrefix(ct, "application/xml") || strings.HasPrefix(ct, "text/xml")) {
		w.isXML = true
	}
	if w.isXML {
		return w.xmlBuf.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func (w *charsetResponseWriter) fixVCardContentType() {
	ct := w.Header().Get("Content-Type")
	if strings.HasPrefix(ct, "text/vcard") && !strings.Contains(ct, "charset") {
		w.Header().Set("Content-Type", ct+"; charset=utf-8")
	}
}

// flushXML writes the buffered XML response to the underlying writer after injecting charset=utf-8 into all text/vcard content-type references.
// This ensures iOS correctly treats embedded vCard data as UTF-8 in PROPFIND and REPORT responses.
func (w *charsetResponseWriter) flushXML() {
	if !w.isXML || w.xmlBuf.Len() == 0 {
		return
	}
	// Fix text/vcard in element content (e.g. <getcontenttype>text/vcard</getcontenttype>)
	data := bytes.ReplaceAll(w.xmlBuf.Bytes(),
		[]byte(">text/vcard<"),
		[]byte(">text/vcard; charset=utf-8<"))
	// Fix text/vcard in attributes (e.g. content-type="text/vcard" in supported-address-data)
	data = bytes.ReplaceAll(data,
		[]byte(`content-type="text/vcard"`),
		[]byte(`content-type="text/vcard; charset=utf-8"`))
	// Add content-type to address-data response elements so iOS correctly decodes UTF-8.
	// iOS (per RFC 2616) defaults text/* to ISO-8859-1 when no charset is declared.
	// The carddav namespace is re-declared on each address-data element by Go's xml encoder.
	data = bytes.ReplaceAll(data,
		[]byte(`<address-data xmlns="urn:ietf:params:xml:ns:carddav">`),
		[]byte(`<address-data content-type="text/vcard; charset=utf-8" xmlns="urn:ietf:params:xml:ns:carddav">`))
	// Fallback: handle the rare case where the carddav namespace is declared on an ancestor element
	data = bytes.ReplaceAll(data,
		[]byte(`<address-data>`),
		[]byte(`<address-data content-type="text/vcard; charset=utf-8">`))
	w.ResponseWriter.Write(data)
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

		crw := &charsetResponseWriter{ResponseWriter: c.Writer}
		h.handler.ServeHTTP(crw, c.Request)
		crw.flushXML()
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
