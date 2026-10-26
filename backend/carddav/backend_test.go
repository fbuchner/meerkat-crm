package carddav

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"meerkat/models"

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
	_, ctx, db := setupCircleFilterBackend(t, models.CardDAVCircleModeInclude, []string{"Family"})
	h := NewHandler(db, t.TempDir())
	serve := func(method, uid string) int {
		req := httptest.NewRequest(method, "/carddav/addressbooks/tester/contacts/"+uid+".vcf", nil).WithContext(ctx)
		w := httptest.NewRecorder()
		h.handler.ServeHTTP(w, req)
		return w.Code
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
