package carddav

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"meerkat/models"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupCircleFilterBackend seeds a user with three contacts: one in "Family",
// one in "Work", one in no circle at all.
func setupCircleFilterBackend(t *testing.T, mode string, circles []string) (*Backend, context.Context, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&models.User{}, &models.Contact{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	user := models.User{Username: "tester", Password: "x", Email: "tester@example.com", CardDAVCircleMode: mode, CardDAVCircles: circles}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	for _, c := range []models.Contact{
		{UserID: user.ID, Firstname: "Fam", VCardUID: "fam", Circles: []string{"Family"}},
		{UserID: user.ID, Firstname: "Colleague", VCardUID: "work", Circles: []string{"Work"}},
		{UserID: user.ID, Firstname: "Stranger", VCardUID: "none"},
	} {
		if err := db.Create(&c).Error; err != nil {
			t.Fatalf("seed contact: %v", err)
		}
	}

	backend := NewBackend(db, t.TempDir())
	ctx := ContextWithUser(context.Background(), user.ID, user.Username, db, t.TempDir())
	return backend, ctx, db
}

func listedUIDs(t *testing.T, b *Backend, ctx context.Context) []string {
	t.Helper()
	objects, err := b.ListAddressObjects(ctx, "/carddav/addressbooks/tester/contacts/", nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	uids := make([]string, 0, len(objects))
	for _, o := range objects {
		uids = append(uids, extractUIDFromPath(o.Path))
	}
	sort.Strings(uids)
	return uids
}

func TestCircleFilterListing(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		circles []string
		want    []string
	}{
		{"all", models.CardDAVCircleModeAll, nil, []string{"fam", "none", "work"}},
		{"unset mode shows all", "", nil, []string{"fam", "none", "work"}},
		{"include", models.CardDAVCircleModeInclude, []string{"Family"}, []string{"fam"}},
		{"exclude", models.CardDAVCircleModeExclude, []string{"Work"}, []string{"fam", "none"}},
		{"case-insensitive", models.CardDAVCircleModeInclude, []string{"family"}, []string{"fam"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, ctx, _ := setupCircleFilterBackend(t, tt.mode, tt.circles)
			got := listedUIDs(t, b, ctx)
			if len(got) != len(tt.want) {
				t.Fatalf("listed %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("listed %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestCircleFilterHidesFilteredContact(t *testing.T) {
	_, _, db := setupCircleFilterBackend(t, models.CardDAVCircleModeInclude, []string{"Family"})
	serve := func(method, uid string) int {
		return serveCardDAV(t, db, method, "/carddav/addressbooks/tester/contacts/"+uid+".vcf", "").Code
	}

	if code := serve(http.MethodGet, "work"); code != http.StatusNotFound {
		t.Fatalf("GET filtered contact: got %d, want 404", code)
	}
	// A device that still holds the contact must not be able to delete it.
	if code := serve(http.MethodDelete, "work"); code != http.StatusNotFound {
		t.Fatalf("DELETE filtered contact: got %d, want 404", code)
	}
	var count int64
	db.Model(&models.Contact{}).Where("vcard_uid = ?", "work").Count(&count)
	if count != 1 {
		t.Fatalf("filtered contact was deleted")
	}

	// A visible contact is still served and deletable.
	if code := serve(http.MethodGet, "fam"); code != http.StatusOK {
		t.Fatalf("GET visible contact: got %d, want 200", code)
	}
	if code := serve(http.MethodDelete, "fam"); code != http.StatusNoContent {
		t.Fatalf("DELETE visible contact: got %d, want 204", code)
	}
}

func TestCircleFilterRejectsPutToHiddenContact(t *testing.T) {
	_, _, db := setupCircleFilterBackend(t, models.CardDAVCircleModeInclude, []string{"Family"})
	card := func(uid, name string) string {
		return "BEGIN:VCARD\r\nVERSION:3.0\r\nUID:" + uid + "\r\nFN:" + name + "\r\nN:;" + name + ";;;\r\nEND:VCARD\r\n"
	}
	firstname := func(uid string) string {
		var c models.Contact
		db.Where("vcard_uid = ?", uid).First(&c)
		return c.Firstname
	}

	// A stale device copy of a filtered-out contact must not overwrite it.
	w := serveCardDAV(t, db, http.MethodPut, "/carddav/addressbooks/tester/contacts/work.vcf", card("work", "Stale"))
	if w.Code != http.StatusPreconditionFailed {
		t.Fatalf("PUT hidden contact: got %d, want 412", w.Code)
	}
	if got := firstname("work"); got != "Colleague" {
		t.Fatalf("hidden contact was overwritten: firstname %q", got)
	}

	// Visible contacts and new contacts are still written.
	if w := serveCardDAV(t, db, http.MethodPut, "/carddav/addressbooks/tester/contacts/fam.vcf", card("fam", "Edited")); w.Code >= 300 {
		t.Fatalf("PUT visible contact: got %d", w.Code)
	}
	if got := firstname("fam"); got != "Edited" {
		t.Fatalf("visible contact not updated: firstname %q", got)
	}
	if w := serveCardDAV(t, db, http.MethodPut, "/carddav/addressbooks/tester/contacts/new.vcf", card("new", "Newbie")); w.Code >= 300 {
		t.Fatalf("PUT new contact: got %d", w.Code)
	}
	if got := firstname("new"); got != "Newbie" {
		t.Fatalf("new contact not created: firstname %q", got)
	}
}

func TestCircleFilterMultiget(t *testing.T) {
	_, _, db := setupCircleFilterBackend(t, models.CardDAVCircleModeInclude, []string{"Family"})
	body := `<?xml version="1.0" encoding="utf-8"?>
<C:addressbook-multiget xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:carddav">
  <D:prop><D:getetag/><C:address-data/></D:prop>
  <D:href>/carddav/addressbooks/tester/contacts/fam.vcf</D:href>
  <D:href>/carddav/addressbooks/tester/contacts/work.vcf</D:href>
</C:addressbook-multiget>`

	w := serveCardDAV(t, db, "REPORT", "/carddav/addressbooks/tester/contacts/", body)
	if w.Code != http.StatusMultiStatus {
		t.Fatalf("multiget: got %d, want 207", w.Code)
	}
	resp := w.Body.String()
	if !strings.Contains(resp, "FN:Fam") {
		t.Fatalf("multiget omitted the visible contact:\n%s", resp)
	}
	if strings.Contains(resp, "FN:Colleague") || !strings.Contains(resp, "404") {
		t.Fatalf("multiget should 404 the filtered contact:\n%s", resp)
	}
}

// serveCardDAV sends one request through the Gin handler, as the router does
// after authentication, so the per-request filter loading is covered too.
func serveCardDAV(t *testing.T, db *gorm.DB, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var user models.User
	if err := db.First(&user).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", user.ID)
		c.Set("username", user.Username)
	})
	handler := NewHandler(db, t.TempDir()).GinHandler()
	router.Any("/carddav/*path", handler)
	router.Handle("REPORT", "/carddav/*path", handler)

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		if method == http.MethodPut {
			req.Header.Set("Content-Type", "text/vcard")
		} else {
			req.Header.Set("Content-Type", "application/xml")
		}
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}
