package carddav

import (
	"bytes"
	"testing"

	"meerkat/models"

	"github.com/emersion/go-vcard"
)

// decodeLegacy normalizes raw vCard text and maps its first card.
func decodeLegacy(t *testing.T, raw string) *models.Contact {
	t.Helper()
	card, err := vcard.NewDecoder(bytes.NewReader(NormalizeLegacyVCF([]byte(raw)))).Decode()
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	contact, _, _, _ := VCardToContact(card, nil)
	return contact
}

// Regression test: vCard 2.1 bare parameters ("TEL;CELL:...") used to lose the
// value, so imports from older phones kept only the name.
func TestNormalizeLegacyVCF_BareParams(t *testing.T) {
	got := decodeLegacy(t, "BEGIN:VCARD\r\n"+
		"VERSION:2.1\r\n"+
		"N:Doe;Jane;;;\r\n"+
		"TEL;PREF;CELL:+49 170 1234567\r\n"+
		"TEL;HOME;VOICE:+49 30 123456\r\n"+
		"EMAIL;INTERNET;WORK:jane@example.com\r\n"+
		"ADR;HOME:;;Main St 1;Berlin;;10115;Germany\r\n"+
		"URL;WORK:https://example.com/jane\r\n"+
		"END:VCARD\r\n")

	wantPhones := []models.ContactPhone{{Type: "cell", Value: "+49 170 1234567"}, {Type: "home", Value: "+49 30 123456"}}
	if len(got.Phones) != 2 || got.Phones[0] != wantPhones[0] || got.Phones[1] != wantPhones[1] {
		t.Errorf("phones = %+v, want %+v", got.Phones, wantPhones)
	}
	if len(got.Emails) != 1 || got.Emails[0] != (models.ContactEmail{Type: "work", Value: "jane@example.com"}) {
		t.Errorf("emails = %+v", got.Emails)
	}
	if len(got.Addresses) != 1 || got.Addresses[0].City != "Berlin" || got.Addresses[0].Type != "home" {
		t.Errorf("addresses = %+v", got.Addresses)
	}
	if len(got.URLs) != 1 || got.URLs[0].Value != "https://example.com/jane" {
		t.Errorf("urls = %+v", got.URLs)
	}
	if got.VCardExtra != "" {
		t.Errorf("nothing should land in VCardExtra, got %q", got.VCardExtra)
	}
}

func TestNormalizeLegacyVCF_QuotedPrintable(t *testing.T) {
	got := decodeLegacy(t, "BEGIN:VCARD\r\n"+
		"VERSION:2.1\r\n"+
		"N;CHARSET=UTF-8;ENCODING=QUOTED-PRINTABLE:M=C3=BCller;J=C3=B6rg;;;\r\n"+
		// Soft line break inside the value, and the bare QUOTED-PRINTABLE form.
		"ADR;HOME;CHARSET=UTF-8;QUOTED-PRINTABLE:;;Stra=C3=9Fe 1=0D=0AHinterhaus;K=C3=B6=\r\n"+
		"ln;;;\r\n"+
		"ORG;CHARSET=ISO-8859-1;ENCODING=QUOTED-PRINTABLE:B=E4ckerei\r\n"+
		"END:VCARD\r\n")

	if got.Firstname != "Jörg" || got.Lastname != "Müller" {
		t.Errorf("name = %q %q, want Jörg Müller", got.Firstname, got.Lastname)
	}
	if len(got.Addresses) != 1 || got.Addresses[0].Street != "Straße 1\nHinterhaus" || got.Addresses[0].City != "Köln" {
		t.Errorf("addresses = %+v", got.Addresses)
	}
	if got.Organization != "Bäckerei" {
		t.Errorf("organization = %q, want Bäckerei", got.Organization)
	}
}

// vCard 3.0 input must decode exactly as it did without normalization.
func TestNormalizeLegacyVCF_LeavesModernCardsAlone(t *testing.T) {
	raw := "BEGIN:VCARD\r\n" +
		"VERSION:3.0\r\n" +
		"N:Doe;Jane;;;\r\n" +
		"TEL;TYPE=CELL:+15551234567\r\n" +
		"item1.EMAIL;TYPE=\"INTERNET;HOME\":jane@example.com\r\n" +
		"NOTE:a\\nb\\, c\r\n" +
		"END:VCARD\r\n"
	got := string(NormalizeLegacyVCF([]byte(raw)))
	if got != raw {
		t.Errorf("modern card changed:\n%q\nwant\n%q", got, raw)
	}
}

// A folded vCard 3.0 card must survive byte for byte. Without the VERSION:2.1
// gate, unfolding rewrote the line and dropped its CHARSET parameter, so modern
// imports went through the 2.1 rewriting path they have no need of.
func TestNormalizeLegacyVCF_LeavesFoldedModernCardAlone(t *testing.T) {
	raw := "BEGIN:VCARD\r\n" +
		"VERSION:3.0\r\n" +
		"NOTE;CHARSET=UTF-8:this is a long note that an exporter folded across\r\n" +
		"  two physical lines\r\n" +
		"END:VCARD\r\n"
	got := string(NormalizeLegacyVCF([]byte(raw)))
	if got != raw {
		t.Errorf("folded modern card changed:\n%q\nwant\n%q", got, raw)
	}
}

// FuzzNormalizeLegacyVCF checks that normalizing arbitrary input never panics and
// that whatever decodes afterwards still maps.
//
// Run: go test ./carddav -run '^$' -fuzz FuzzNormalizeLegacyVCF -fuzztime 30s
func FuzzNormalizeLegacyVCF(f *testing.F) {
	f.Add([]byte("BEGIN:VCARD\r\nVERSION:2.1\r\nTEL;CELL:+1\r\nN;QUOTED-PRINTABLE:=C3=A4=\r\n;\r\nEND:VCARD\r\n"))
	f.Add([]byte("BEGIN:VCARD\nTEL;\"a:b\";;X:1\n =\nEND:VCARD\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		dec := vcard.NewDecoder(bytes.NewReader(NormalizeLegacyVCF(data)))
		for i := 0; i < 100; i++ {
			card, err := dec.Decode()
			if err != nil {
				return
			}
			VCardToContact(card, nil)
		}
	})
}
