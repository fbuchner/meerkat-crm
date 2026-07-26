package contactmodel

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Projection holds derived/projected scalar columns computed from a Record's
// Card. It is used later by the P1 persistence layer for denormalized DB
// columns (extending the intent of today's BeforeSave denormalization).
type Projection struct {
	Firstname, Lastname, FN    string
	PrimaryEmail, PrimaryPhone string
	Birthday                   string // ISO or --MM-DD
	Org                        string
}

// DeriveProjection computes a Projection from a Record. Pure function; no DB.
//
//   - Firstname = first NameComponent kind=given .Value; Lastname = kind=surname .Value
//   - FN = Card.Name.Full (fallback: join given+surname)
//   - PrimaryEmail = Emails sorted by (Pref asc, index) [0].Address
//   - PrimaryPhone = Phones sorted by (Pref asc, index) [0].Number
//   - Birthday = Anniversaries where Kind=="birth" -> partial/timestamp formatted ISO
//   - Org = Organizations[0].Name
func DeriveProjection(r *Record) Projection {
	var p Projection
	if r == nil {
		return p
	}
	card := r.Card

	if card.Name != nil {
		for _, c := range card.Name.Components {
			if c.Kind == "given" {
				p.Firstname = c.Value
				break
			}
		}
		for _, c := range card.Name.Components {
			if c.Kind == "surname" {
				p.Lastname = c.Value
				break
			}
		}
		p.FN = card.Name.Full
	}
	if p.FN == "" {
		p.FN = strings.TrimSpace(strings.Join(nonEmpty(p.Firstname, p.Lastname), " "))
	}

	if len(card.Emails) > 0 {
		emails := make([]Email, len(card.Emails))
		copy(emails, card.Emails)
		sort.SliceStable(emails, func(i, j int) bool {
			return effectivePref(emails[i].Pref) < effectivePref(emails[j].Pref)
		})
		p.PrimaryEmail = emails[0].Address
	}

	if len(card.Phones) > 0 {
		phones := make([]Phone, len(card.Phones))
		copy(phones, card.Phones)
		sort.SliceStable(phones, func(i, j int) bool {
			return effectivePref(phones[i].Pref) < effectivePref(phones[j].Pref)
		})
		p.PrimaryPhone = phones[0].Number
	}

	for _, a := range card.Anniversaries {
		if a.Kind == "birth" {
			p.Birthday = formatAnniversaryDate(a.Date)
			break
		}
	}

	if len(card.Organizations) > 0 {
		p.Org = card.Organizations[0].Name
	}

	return p
}

// nonEmpty returns only the non-empty strings from vals, preserving order.
func nonEmpty(vals ...string) []string {
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

// effectivePref returns the numeric value used for pref comparisons: the
// dereferenced Pref if set, or the largest possible value (sorts last) if not
// — an unset Pref is treated as "no explicit preference", ranking behind any
// item that does specify one.
func effectivePref(pref *int) int {
	if pref == nil {
		return math.MaxInt
	}
	return *pref
}

// formatAnniversaryDate renders an AnniversaryDate as an ISO-ish string:
// full "YYYY-MM-DD" when a complete partial date or timestamp is present,
// "--MM-DD" for a partial date lacking a year (matching vCard's convention
// for year-less BDAY values), or a best-effort partial rendering otherwise.
func formatAnniversaryDate(d AnniversaryDate) string {
	if d.Timestamp != nil && *d.Timestamp != "" {
		ts := *d.Timestamp
		if len(ts) >= 10 {
			return ts[:10]
		}
		return ts
	}
	if d.Partial != nil {
		p := d.Partial
		switch {
		case p.Year != nil && p.Month != nil && p.Day != nil:
			return fmt.Sprintf("%04d-%02d-%02d", *p.Year, *p.Month, *p.Day)
		case p.Year == nil && p.Month != nil && p.Day != nil:
			return fmt.Sprintf("--%02d-%02d", *p.Month, *p.Day)
		case p.Year != nil && p.Month != nil && p.Day == nil:
			return fmt.Sprintf("%04d-%02d", *p.Year, *p.Month)
		case p.Year != nil && p.Month == nil && p.Day == nil:
			return fmt.Sprintf("%04d", *p.Year)
		case p.Year == nil && p.Month != nil && p.Day == nil:
			return fmt.Sprintf("--%02d", *p.Month)
		}
	}
	return ""
}
