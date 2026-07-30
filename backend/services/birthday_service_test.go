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

// TestDaysUntilBirthday_ShortStringReturns999 covers the guard clause for
// birthday strings too short to contain a month/day (len < 7), e.g. empty
// string or malformed data.
func TestDaysUntilBirthday_ShortStringReturns999(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	assert.Equal(t, 999, DaysUntilBirthday("", now))
	assert.Equal(t, 999, DaysUntilBirthday("1990", now))
	assert.Equal(t, 999, DaysUntilBirthday("--1-1", now)) // len 5, too short
}

// TestDaysUntilBirthday_UnparsableMonthOrDayReturns999 covers the parse-error
// branch: a birthday string long enough to be parsed but with an
// out-of-range month or day.
func TestDaysUntilBirthday_UnparsableMonthOrDayReturns999(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	assert.Equal(t, 999, DaysUntilBirthday("1990-13-40", now), "month 13 and day 40 are both out of range")
	assert.Equal(t, 999, DaysUntilBirthday("1990-99-99", now), "month and day out of range")
	assert.Equal(t, 999, DaysUntilBirthday("--ab-cd", now), "non-numeric month/day")
}

// TestDaysUntilBirthday_DecemberToJanuaryBoundary is the explicit Dec 31 ->
// Jan 1 boundary case called out in docs/fork-plan/45-test-coverage-closure.md
// Phase 3b: today is Dec 31, birthday is Jan 1 (tomorrow). The birthday-this-
// year (Jan 1 of the *current* now.Year()) is necessarily "before" today
// (Dec 31 of that same year), so the function must wrap it forward into next
// year rather than leaving it hundreds of days in the past.
func TestDaysUntilBirthday_DecemberToJanuaryBoundary(t *testing.T) {
	today := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	// Full ISO date birthday.
	assert.Equal(t, 1, DaysUntilBirthday("2020-01-01", today), "Jan 1 birthday should be 1 day away when today is Dec 31")

	// Year-unknown (--MM-DD) birthday.
	assert.Equal(t, 1, DaysUntilBirthday("--01-01", today), "Jan 1 birthday (year-unknown format) should be 1 day away when today is Dec 31")
}

// TestDaysUntilBirthday_JanuaryFirstBirthdayToday covers the other side of
// the year boundary: today IS Jan 1 and the birthday is also Jan 1, so the
// function must report 0 (today), not wrap forward a full year.
func TestDaysUntilBirthday_JanuaryFirstBirthdayToday(t *testing.T) {
	today := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	assert.Equal(t, 0, DaysUntilBirthday("2020-01-01", today))
	assert.Equal(t, 0, DaysUntilBirthday("--01-01", today))
}

// TestDaysUntilBirthday_ForwardLookingNotRecentPast documents the direction
// DaysUntilBirthday actually calculates in: it always returns days *until*
// the next occurrence, never "days since it last happened". Checking on
// Jan 2 for a birthday on Dec 31 does NOT report "1 day ago" (which a naive
// signed diff might); it reports the number of days until the *next*
// Dec 31, i.e. almost a full year away.
func TestDaysUntilBirthday_ForwardLookingNotRecentPast(t *testing.T) {
	today := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC) // 2026 is not a leap year

	got := DaysUntilBirthday("2020-12-31", today)

	assert.Equal(t, 363, got, "should count forward to the next Dec 31 (365 - 2 days elapsed), not backward to the one that just passed")
	assert.Positive(t, got, "DaysUntilBirthday must never return a negative 'days ago' value")
}

// TestDaysUntilBirthday_LeapYearFeb29CheckedInNonLeapYear documents current
// behavior for a Feb 29 birthday when now.Year() is not a leap year: since
// birthdayThisYear is built with time.Date(now.Year(), Feb, 29, ...) and Go's
// time.Date normalizes out-of-range days, Feb 29 rolls forward to March 1 in
// non-leap years rather than clamping to Feb 28.
func TestDaysUntilBirthday_LeapYearFeb29CheckedInNonLeapYear(t *testing.T) {
	today := time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC) // 2025 is not a leap year

	got := DaysUntilBirthday("2000-02-29", today)

	assert.Equal(t, 9, got, "Feb 29 normalizes to Mar 1 in a non-leap year (time.Date day-overflow), giving 9 days from Feb 20")
}

// TestDaysUntilBirthday_LeapYearFeb29CheckedInLeapYear sanity-checks the
// Feb 29 birthday when now.Year() actually is a leap year, where no
// normalization is needed.
func TestDaysUntilBirthday_LeapYearFeb29CheckedInLeapYear(t *testing.T) {
	today := time.Date(2024, 2, 20, 0, 0, 0, 0, time.UTC) // 2024 is a leap year

	got := DaysUntilBirthday("2000-02-29", today)

	assert.Equal(t, 9, got)
}
