package photostore

import (
	"bytes"
	"encoding/base64"
	"image/jpeg"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emersion/go-vcard"
)

// testPNGBase64 is a 1x1 transparent PNG, small enough to embed inline in
// tests. Same fixture used by services/contact_sync_service_test.go's
// testCardWithPhoto for the same reason.
const testPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="

func decodeTestPNG(t *testing.T) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(testPNGBase64)
	if err != nil {
		t.Fatalf("failed to decode test fixture PNG: %v", err)
	}
	return data
}

// --- SaveContactPhoto ---

func TestSaveContactPhoto(t *testing.T) {
	dir := t.TempDir()
	data := decodeTestPNG(t)

	photoPath, thumbnail, err := SaveContactPhoto(data, "image/png", dir)
	if err != nil {
		t.Fatalf("SaveContactPhoto returned error: %v", err)
	}
	if photoPath == "" {
		t.Fatal("expected a non-empty photo path")
	}
	if !strings.HasSuffix(photoPath, "_photo.jpg") {
		t.Errorf("photoPath = %q, want a _photo.jpg suffix", photoPath)
	}
	if !strings.HasPrefix(thumbnail, "data:image/jpeg;base64,") {
		t.Errorf("thumbnail = %q, want a data:image/jpeg;base64, prefix", thumbnail)
	}

	// The photo must actually be written to disk under photoDir, and must be
	// a valid re-encoded JPEG (SaveContactPhoto always re-encodes to JPEG
	// regardless of the input format).
	fullPath := filepath.Join(dir, photoPath)
	fileBytes, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("photo was not written to disk at %q: %v", fullPath, err)
	}
	if _, err := jpeg.Decode(bytes.NewReader(fileBytes)); err != nil {
		t.Errorf("saved photo is not a valid JPEG: %v", err)
	}

	// The thumbnail payload must itself decode to a valid JPEG.
	thumbData := strings.TrimPrefix(thumbnail, "data:image/jpeg;base64,")
	decodedThumb, err := base64.StdEncoding.DecodeString(thumbData)
	if err != nil {
		t.Fatalf("thumbnail is not valid base64: %v", err)
	}
	if _, err := jpeg.Decode(bytes.NewReader(decodedThumb)); err != nil {
		t.Errorf("thumbnail is not a valid JPEG: %v", err)
	}
}

func TestSaveContactPhoto_EmptyInputIsNoop(t *testing.T) {
	photoPath, thumbnail, err := SaveContactPhoto(nil, "", t.TempDir())
	if err != nil {
		t.Fatalf("SaveContactPhoto(nil) returned error: %v", err)
	}
	if photoPath != "" || thumbnail != "" {
		t.Errorf("SaveContactPhoto(nil) = (%q, %q), want empty strings", photoPath, thumbnail)
	}
}

func TestSaveContactPhoto_InvalidImageDataErrors(t *testing.T) {
	_, _, err := SaveContactPhoto([]byte("this is not an image"), "", t.TempDir())
	if err == nil {
		t.Error("expected an error decoding non-image bytes, got nil")
	}
}

// --- ReadContactPhoto / ReadContactPhotoDataURI ---

func TestReadContactPhotoRoundTrip(t *testing.T) {
	dir := t.TempDir()
	data := decodeTestPNG(t)

	photoPath, _, err := SaveContactPhoto(data, "image/png", dir)
	if err != nil {
		t.Fatalf("SaveContactPhoto returned error: %v", err)
	}

	onDisk, err := os.ReadFile(filepath.Join(dir, photoPath))
	if err != nil {
		t.Fatalf("failed to read saved photo back directly: %v", err)
	}

	got, mediaType := ReadContactPhoto(photoPath, "", dir)
	if mediaType != "image/jpeg" {
		t.Errorf("mediaType = %q, want image/jpeg", mediaType)
	}
	decoded, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		t.Fatalf("ReadContactPhoto returned invalid base64: %v", err)
	}
	if !bytes.Equal(decoded, onDisk) {
		t.Error("ReadContactPhoto bytes do not match the bytes actually on disk")
	}
}

func TestReadContactPhoto_FallsBackToThumbnailWhenFileMissing(t *testing.T) {
	dir := t.TempDir()
	thumbnail := "data:image/jpeg;base64,Zm9vYmFy" // "foobar"

	got, mediaType := ReadContactPhoto("does-not-exist_photo.jpg", thumbnail, dir)
	if mediaType != "image/jpeg" {
		t.Errorf("mediaType = %q, want image/jpeg", mediaType)
	}
	if got != "Zm9vYmFy" {
		t.Errorf("got = %q, want raw base64 payload from thumbnail", got)
	}
}

func TestReadContactPhoto_NoPhotoAvailable(t *testing.T) {
	got, mediaType := ReadContactPhoto("", "", "")
	if got != "" || mediaType != "" {
		t.Errorf("ReadContactPhoto with nothing set = (%q, %q), want empty strings", got, mediaType)
	}
}

func TestReadContactPhotoDataURI(t *testing.T) {
	dir := t.TempDir()
	data := decodeTestPNG(t)
	photoPath, _, err := SaveContactPhoto(data, "image/png", dir)
	if err != nil {
		t.Fatalf("SaveContactPhoto returned error: %v", err)
	}

	uri, mediaType := ReadContactPhotoDataURI(photoPath, "", dir)
	if mediaType != "image/jpeg" {
		t.Errorf("mediaType = %q, want image/jpeg", mediaType)
	}
	if !strings.HasPrefix(uri, "data:image/jpeg;base64,") {
		t.Errorf("uri = %q, want a data:image/jpeg;base64, prefix", uri)
	}
}

func TestReadContactPhotoDataURI_NoPhotoAvailable(t *testing.T) {
	uri, mediaType := ReadContactPhotoDataURI("", "", "")
	if uri != "" || mediaType != "" {
		t.Errorf("ReadContactPhotoDataURI with nothing set = (%q, %q), want empty strings", uri, mediaType)
	}
}

// --- ExtractPhotoData ---

func TestExtractPhotoData_Nil(t *testing.T) {
	data, mediaType, photoURL := ExtractPhotoData(nil)
	if data != nil || mediaType != "" || photoURL != "" {
		t.Errorf("ExtractPhotoData(nil) = (%v, %q, %q), want all zero values", data, mediaType, photoURL)
	}
}

func TestExtractPhotoData_InlineBase64WithTypeParam(t *testing.T) {
	data := decodeTestPNG(t)
	field := &vcard.Field{
		Value:  base64.StdEncoding.EncodeToString(data),
		Params: vcard.Params{"TYPE": []string{"PNG"}},
	}

	got, mediaType, photoURL := ExtractPhotoData(field)
	if photoURL != "" {
		t.Errorf("photoURL = %q, want empty for inline data", photoURL)
	}
	if mediaType != "image/png" {
		t.Errorf("mediaType = %q, want image/png (derived from TYPE=PNG)", mediaType)
	}
	if !bytes.Equal(got, data) {
		t.Error("decoded photo bytes do not match the original inline base64 payload")
	}
}

func TestExtractPhotoData_DataURIForm(t *testing.T) {
	data := decodeTestPNG(t)
	field := &vcard.Field{
		Value: "data:image/png;base64," + base64.StdEncoding.EncodeToString(data),
	}

	got, mediaType, photoURL := ExtractPhotoData(field)
	if photoURL != "" {
		t.Errorf("photoURL = %q, want empty for a data: URI", photoURL)
	}
	if mediaType != "image/png" {
		t.Errorf("mediaType = %q, want image/png (parsed out of the data: URI)", mediaType)
	}
	if !bytes.Equal(got, data) {
		t.Error("decoded photo bytes do not match the original data: URI payload")
	}
}

func TestExtractPhotoData_ReferencedURI(t *testing.T) {
	field := &vcard.Field{Value: "https://example.com/photo.jpg"}

	data, _, photoURL := ExtractPhotoData(field)
	if data != nil {
		t.Errorf("expected nil data for a referenced (non-embedded) photo URL, got %v", data)
	}
	if photoURL != "https://example.com/photo.jpg" {
		t.Errorf("photoURL = %q, want the original https URL", photoURL)
	}
}

// --- FetchPhotoFromURL: SSRF protection ---
//
// FetchPhotoFromURL delegates to httputil.FetchImageFromURL, which blocks
// loopback/private/link-local targets both by hostname (a fixed blocklist
// checked before any DNS lookup) and by resolved IP (net.LookupIP on an IP
// literal parses it directly, no real network access needed) — the same
// SSRF-protection shape services/calendar_sync_service.go's
// privateBlockingDialContext documents for the CalDAV sync client. These
// cases are all resolved without any real DNS/network call, so they run
// deterministically in this test.

func TestFetchPhotoFromURL_BlocksLocalhostHostname(t *testing.T) {
	if _, _, err := FetchPhotoFromURL("http://localhost/photo.jpg"); err == nil {
		t.Error("expected fetching from localhost to be rejected as an SSRF target")
	}
}

func TestFetchPhotoFromURL_BlocksLoopbackIPv4Literal(t *testing.T) {
	if _, _, err := FetchPhotoFromURL("http://127.0.0.1/photo.jpg"); err == nil {
		t.Error("expected fetching from 127.0.0.1 to be rejected as an SSRF target")
	}
}

func TestFetchPhotoFromURL_BlocksLoopbackIPv6Literal(t *testing.T) {
	if _, _, err := FetchPhotoFromURL("http://[::1]/photo.jpg"); err == nil {
		t.Error("expected fetching from [::1] to be rejected as an SSRF target")
	}
}

func TestFetchPhotoFromURL_BlocksPrivateLANAddress(t *testing.T) {
	// 192.168.0.0/16 is not in the hardcoded hostname blocklist at all -- this
	// proves the resolved-IP check (ip.IsPrivate()), not just the hostname
	// string list, actually rejects it.
	if _, _, err := FetchPhotoFromURL("http://192.168.1.1/photo.jpg"); err == nil {
		t.Error("expected fetching from a private LAN address (192.168.1.1) to be rejected")
	}
}

func TestFetchPhotoFromURL_BlocksLinkLocalMetadataAddress(t *testing.T) {
	// 169.254.169.254 is the classic cloud-metadata SSRF target (AWS/GCP/Azure
	// instance metadata service); ip.IsLinkLocalUnicast() must catch it.
	if _, _, err := FetchPhotoFromURL("http://169.254.169.254/latest/meta-data/"); err == nil {
		t.Error("expected fetching from the link-local metadata address to be rejected")
	}
}

func TestFetchPhotoFromURL_RejectsNonHTTPScheme(t *testing.T) {
	if _, _, err := FetchPhotoFromURL("file:///etc/passwd"); err == nil {
		t.Error("expected a non-http(s) scheme to be rejected")
	}
}

func TestFetchPhotoFromURL_RejectsUnparsableURL(t *testing.T) {
	// A URL containing a raw control character fails url.Parse itself.
	_, err := url.Parse("http://\x7f")
	if err == nil {
		t.Skip("test environment's url.Parse unexpectedly accepted the malformed URL")
	}
	if _, _, err := FetchPhotoFromURL("http://\x7f"); err == nil {
		t.Error("expected an unparsable URL to be rejected")
	}
}
