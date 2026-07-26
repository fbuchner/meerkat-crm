package contactmodel

import "testing"

func TestProjectionRoundTrip(t *testing.T) {
	assertRoundTrip(t, fullProjection())
}

func TestDeriveProjection_Normal(t *testing.T) {
	r := fullRecord()

	p := DeriveProjection(&r)

	if p.Firstname != "Ada" {
		t.Errorf("Firstname = %q, want %q", p.Firstname, "Ada")
	}
	if p.Lastname != "Lovelace" {
		t.Errorf("Lastname = %q, want %q", p.Lastname, "Lovelace")
	}
	if p.FN != "Ada Lovelace" {
		t.Errorf("FN = %q, want %q", p.FN, "Ada Lovelace")
	}
	if p.PrimaryEmail != "ada@example.com" {
		t.Errorf("PrimaryEmail = %q, want %q", p.PrimaryEmail, "ada@example.com")
	}
	if p.PrimaryPhone != "+15551234567" {
		t.Errorf("PrimaryPhone = %q, want %q", p.PrimaryPhone, "+15551234567")
	}
	if p.Birthday != "1815-12-10" {
		t.Errorf("Birthday = %q, want %q", p.Birthday, "1815-12-10")
	}
	if p.Org != "Analytical Engines Ltd" {
		t.Errorf("Org = %q, want %q", p.Org, "Analytical Engines Ltd")
	}
}

func TestDeriveProjection_FNFallback(t *testing.T) {
	r := &Record{
		Card: Card{
			Name: &Name{
				Components: []NameComponent{
					{Kind: "given", Value: "Grace"},
					{Kind: "surname", Value: "Hopper"},
				},
				// Full intentionally left empty to exercise the fallback.
			},
		},
	}

	p := DeriveProjection(r)

	if p.FN != "Grace Hopper" {
		t.Errorf("FN fallback = %q, want %q", p.FN, "Grace Hopper")
	}
}

func TestDeriveProjection_EmptyEmailsAndPhonesNoPanic(t *testing.T) {
	r := &Record{Card: Card{
		Name: &Name{Full: "No Contacts"},
	}}

	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("DeriveProjection panicked on empty Emails/Phones: %v", rec)
		}
	}()

	p := DeriveProjection(r)

	if p.PrimaryEmail != "" {
		t.Errorf("PrimaryEmail = %q, want empty", p.PrimaryEmail)
	}
	if p.PrimaryPhone != "" {
		t.Errorf("PrimaryPhone = %q, want empty", p.PrimaryPhone)
	}
}

func TestDeriveProjection_NilRecordNoPanic(t *testing.T) {
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("DeriveProjection panicked on nil *Record: %v", rec)
		}
	}()

	p := DeriveProjection(nil)

	if p != (Projection{}) {
		t.Errorf("expected zero Projection for nil Record, got %+v", p)
	}
}

func TestDeriveProjection_MultipleEmailsLowestPrefWins(t *testing.T) {
	r := &Record{Card: Card{
		Emails: []Email{
			{Address: "no-pref@example.com"},
			{Address: "pref-5@example.com", Pref: ptr(5)},
			{Address: "pref-1@example.com", Pref: ptr(1)},
			{Address: "pref-3@example.com", Pref: ptr(3)},
		},
		Phones: []Phone{
			{Number: "no-pref-000"},
			{Number: "pref-5-000", Pref: ptr(5)},
			{Number: "pref-1-000", Pref: ptr(1)},
		},
	}}

	p := DeriveProjection(r)

	if p.PrimaryEmail != "pref-1@example.com" {
		t.Errorf("PrimaryEmail = %q, want lowest-Pref email %q", p.PrimaryEmail, "pref-1@example.com")
	}
	if p.PrimaryPhone != "pref-1-000" {
		t.Errorf("PrimaryPhone = %q, want lowest-Pref phone %q", p.PrimaryPhone, "pref-1-000")
	}
}

func TestDeriveProjection_UnsetPrefSortsLastAndIsStable(t *testing.T) {
	r := &Record{Card: Card{
		Emails: []Email{
			{Address: "first-no-pref@example.com"},
			{Address: "second-no-pref@example.com"},
		},
	}}

	p := DeriveProjection(r)

	// Neither email specifies Pref, so original index order must win: the
	// first element stays primary (stable sort, "index" tiebreak per spec).
	if p.PrimaryEmail != "first-no-pref@example.com" {
		t.Errorf("PrimaryEmail = %q, want %q (stable order preserved)", p.PrimaryEmail, "first-no-pref@example.com")
	}
}

func TestDeriveProjection_NoAnniversariesBirthdayEmpty(t *testing.T) {
	r := &Record{Card: Card{
		Name:          &Name{Full: "No Birthday"},
		Anniversaries: nil,
	}}

	p := DeriveProjection(r)

	if p.Birthday != "" {
		t.Errorf("Birthday = %q, want empty when there are no Anniversaries", p.Birthday)
	}
}

func TestDeriveProjection_AnniversaryWrongKindIgnored(t *testing.T) {
	r := &Record{Card: Card{
		Anniversaries: []Anniversary{
			{Kind: "wedding", Date: AnniversaryDate{Partial: &PartialDate{Year: ptr(2000), Month: ptr(6), Day: ptr(1)}}},
		},
	}}

	p := DeriveProjection(r)

	if p.Birthday != "" {
		t.Errorf("Birthday = %q, want empty when only a non-birth anniversary is present", p.Birthday)
	}
}

func TestDeriveProjection_BirthdayPartialWithoutYear(t *testing.T) {
	r := &Record{Card: Card{
		Anniversaries: []Anniversary{
			{Kind: "birth", Date: AnniversaryDate{Partial: &PartialDate{Month: ptr(7), Day: ptr(4)}}},
		},
	}}

	p := DeriveProjection(r)

	if p.Birthday != "--07-04" {
		t.Errorf("Birthday = %q, want %q for a year-less partial date", p.Birthday, "--07-04")
	}
}
