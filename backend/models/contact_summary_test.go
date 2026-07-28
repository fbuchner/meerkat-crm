package models

import (
	"testing"

	"mycorrhizal/contactmodel"
)

// TestNewContactRecordResponse_PreservesPersistedCardOnlyData is the
// regression test for a real, live bug (found while auditing WP-73's work):
// NewContactRecordResponse called RecordFromContact directly, which
// silently drops any Card-only data with no flat-field home (SpeakToAs
// here) from GET /api/v1/contacts/{id} and the POST/PUT response bodies —
// exactly the data a nested REST write would have just set. It must go
// through RecordForContact instead, which prefers the already-persisted
// Card. See models.RecordForContact's doc comment for the full history.
func TestNewContactRecordResponse_PreservesPersistedCardOnlyData(t *testing.T) {
	c := &Contact{
		Firstname: "Ada",
		Card: contactmodel.Card{
			Name: &contactmodel.Name{Components: []contactmodel.NameComponent{{Kind: "given", Value: "Ada"}}},
			SpeakToAs: &contactmodel.SpeakToAs{
				Pronouns: []contactmodel.Pronouns{{Pronouns: "she/her"}},
			},
		},
	}

	resp := NewContactRecordResponse(c, "")

	if resp.Card.SpeakToAs == nil || len(resp.Card.SpeakToAs.Pronouns) != 1 || resp.Card.SpeakToAs.Pronouns[0].Pronouns != "she/her" {
		t.Errorf("ContactRecordResponse.Card.SpeakToAs = %+v, want the persisted she/her preserved in the API response", resp.Card.SpeakToAs)
	}
}

// TestNewContactSummary_IncludesNicknameAndCircles is a regression test for
// the frontend-migration pre-work gap: ContactsPage's list view renders
// nickname and circles per row, but GET /contacts' slim ContactSummary
// projection didn't carry either field until this fix.
func TestNewContactSummary_IncludesNicknameAndCircles(t *testing.T) {
	c := &Contact{
		Firstname: "Ada",
		Lastname:  "Lovelace",
		Nickname:  "Countess",
		Circles:   []string{"Friends", "Work"},
	}

	summary := NewContactSummary(c)

	if summary.Nickname != "Countess" {
		t.Errorf("ContactSummary.Nickname = %q, want %q", summary.Nickname, "Countess")
	}
	if len(summary.Circles) != 2 || summary.Circles[0] != "Friends" || summary.Circles[1] != "Work" {
		t.Errorf("ContactSummary.Circles = %+v, want [Friends Work]", summary.Circles)
	}
}
