package services

import (
	"mycorrhizal/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetUpcomingBirthdays_TruncatesAfterMaxResultsPastTwoWeeks is a
// regression test for the resultCount truncation loop in
// GetUpcomingBirthdays: once a birthday is both beyond the two-week window
// and past maxResults (5), the loop must stop growing resultCount for good
// (birthdays are sorted ascending by days-until, so nothing later in the
// slice would ever re-satisfy either condition). This exact loop was
// rewritten from an if/else chain to a switch during a lint cleanup pass;
// break inside a switch only exits the switch, not an enclosing for loop,
// so a naive conversion would have silently kept iterating instead of
// stopping — this test locks in the correct (labeled-break) behavior.
func TestGetUpcomingBirthdays_TruncatesAfterMaxResultsPastTwoWeeks(t *testing.T) {
	db, _ := setupRouter()

	user := models.User{Username: "birthdaytester", Password: "password123", Email: "bday@example.com"}
	require.NoError(t, db.Create(&user).Error)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// 8 contacts, each 3 days apart: days-until = 5,8,11,...,26. The first
	// 5 (up to day 17, still >2 weeks for the later ones) exceed maxResults
	// only after the 2-week (14-day) window closes it off; construct so
	// exactly 6 contacts fall within 14 days (forcing resultCount past
	// maxResults via the "within two weeks" branch) and 2 fall beyond both
	// the window and maxResults, which must NOT be included.
	days := []int{2, 4, 6, 8, 10, 12, 20, 24} // 6 within 14 days, 2 well beyond
	for i, d := range days {
		bday := now.AddDate(0, 0, d)
		contact := models.Contact{
			UserID:    user.ID,
			Firstname: "Contact",
			Lastname:  string(rune('A' + i)),
			Birthday:  bday.Format("2006-01-02"),
			Archived:  false,
		}
		require.NoError(t, db.Create(&contact).Error)
	}

	birthdays, err := GetUpcomingBirthdays(db, user.ID, now)
	require.NoError(t, err)

	// All 6 within-two-weeks birthdays must be present; the 2 far-future
	// ones must be excluded (they're both past maxResults and past the
	// two-week window).
	assert.Len(t, birthdays, 6, "expected only the 6 within-two-week birthdays, got %d: %+v", len(birthdays), birthdays)
	for _, b := range birthdays {
		days := DaysUntilBirthday(b.Birthday, now)
		assert.LessOrEqual(t, days, 14, "no returned birthday should be more than 2 weeks out given maxResults was already exceeded")
	}
}

// TestGetUpcomingBirthdays_CapsAtMaxResultsWhenNoneWithinTwoWeeks asserts
// the maxResults=5 cap applies when nothing is within the two-week window.
func TestGetUpcomingBirthdays_CapsAtMaxResultsWhenNoneWithinTwoWeeks(t *testing.T) {
	db, _ := setupRouter()

	user := models.User{Username: "birthdaytester2", Password: "password123", Email: "bday2@example.com"}
	require.NoError(t, db.Create(&user).Error)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// 8 contacts, all well beyond the 2-week window (20+ days out).
	for i := 0; i < 8; i++ {
		bday := now.AddDate(0, 0, 20+i)
		contact := models.Contact{
			UserID:    user.ID,
			Firstname: "Contact",
			Lastname:  string(rune('A' + i)),
			Birthday:  bday.Format("2006-01-02"),
			Archived:  false,
		}
		require.NoError(t, db.Create(&contact).Error)
	}

	birthdays, err := GetUpcomingBirthdays(db, user.ID, now)
	require.NoError(t, err)
	assert.Len(t, birthdays, 5, "expected exactly maxResults=5 birthdays when none are within the two-week window")
}
