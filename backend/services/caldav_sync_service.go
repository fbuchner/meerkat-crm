package services

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"meerkat/httputil"
	"meerkat/models"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	defaultCalDAVSyncLimit = 200
	maxCalDAVResponseSize  = 10 * 1024 * 1024
)

// CalDAVSyncResult summarizes a CalDAV activity sync run.
type CalDAVSyncResult struct {
	Created int `json:"created"`
	Skipped int `json:"skipped"`
}

type calDAVEvent struct {
	Title       string
	Description string
	Location    string
	Date        time.Time
}

// CalDAVSyncService imports CalDAV VEVENTs into Meerkat activities.
type CalDAVSyncService struct {
	client      *http.Client
	validateURL func(string) (*url.URL, error)
}

// NewCalDAVSyncService creates a CalDAV sync service with SSRF-safe fetching.
func NewCalDAVSyncService() *CalDAVSyncService {
	return &CalDAVSyncService{
		client:      httputil.NewSafeHTTPClient(30 * time.Second),
		validateURL: httputil.ValidateURLForSSRF,
	}
}

// Sync fetches a CalDAV calendar once and imports valid VEVENTs as user activities.
func (s *CalDAVSyncService) Sync(ctx context.Context, db *gorm.DB, userID uint, input models.CalDAVSyncInput) (CalDAVSyncResult, error) {
	if s.client == nil {
		s.client = httputil.NewSafeHTTPClient(30 * time.Second)
	}
	if s.validateURL == nil {
		s.validateURL = httputil.ValidateURLForSSRF
	}

	parsedURL, err := s.validateURL(strings.TrimSpace(input.URL))
	if err != nil {
		return CalDAVSyncResult{}, fmt.Errorf("invalid calendar URL: %w", err)
	}

	contactIDs := uniqueUintIDs(input.ContactIDs)
	contacts, err := loadUserContacts(db, userID, contactIDs)
	if err != nil {
		return CalDAVSyncResult{}, err
	}

	payloads, err := s.fetchCalendarPayloads(ctx, parsedURL.String(), input.Username, input.Password)
	if err != nil {
		return CalDAVSyncResult{}, err
	}

	var events []calDAVEvent
	for _, payload := range payloads {
		parsedEvents, parseErr := parseICalendarEvents(payload)
		if parseErr != nil {
			return CalDAVSyncResult{}, parseErr
		}
		events = append(events, parsedEvents...)
	}

	limit := input.Limit
	if limit <= 0 || limit > defaultCalDAVSyncLimit {
		limit = defaultCalDAVSyncLimit
	}
	if len(events) > limit {
		events = events[:limit]
	}

	var result CalDAVSyncResult
	err = db.Transaction(func(tx *gorm.DB) error {
		for _, event := range events {
			var existing models.Activity
			err := tx.Where(
				"user_id = ? AND title = ? AND date = ? AND location = ?",
				userID,
				event.Title,
				event.Date,
				event.Location,
			).First(&existing).Error
			if err == nil {
				result.Skipped++
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			activity := models.Activity{
				UserID:      userID,
				Title:       event.Title,
				Description: event.Description,
				Location:    event.Location,
				Date:        event.Date,
			}
			if err := tx.Create(&activity).Error; err != nil {
				return err
			}
			if len(contacts) > 0 {
				if err := tx.Model(&activity).Association("Contacts").Append(contacts); err != nil {
					return err
				}
			}
			result.Created++
		}
		return nil
	})
	if err != nil {
		return CalDAVSyncResult{}, fmt.Errorf("failed to import CalDAV activities: %w", err)
	}

	return result, nil
}

func (s *CalDAVSyncService) fetchCalendarPayloads(ctx context.Context, calendarURL, username, password string) ([]string, error) {
	reportBody := `<?xml version="1.0" encoding="utf-8" ?>
<c:calendar-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:prop>
    <d:getetag />
    <c:calendar-data />
  </d:prop>
  <c:filter>
    <c:comp-filter name="VCALENDAR">
      <c:comp-filter name="VEVENT" />
    </c:comp-filter>
  </c:filter>
</c:calendar-query>`

	respBody, status, err := s.doCalendarRequest(ctx, http.MethodPost, calendarURL, username, password, reportBody)
	if err != nil {
		return nil, err
	}
	if status == http.StatusMethodNotAllowed || status == http.StatusNotImplemented {
		respBody, status, err = s.doCalendarRequest(ctx, http.MethodGet, calendarURL, username, password, "")
		if err != nil {
			return nil, err
		}
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("calendar service returned %d", status)
	}

	payloads := extractCalendarPayloads(respBody)
	if len(payloads) == 0 {
		return nil, fmt.Errorf("calendar response did not include VEVENT data")
	}
	return payloads, nil
}

func (s *CalDAVSyncService) doCalendarRequest(ctx context.Context, method, calendarURL, username, password, body string) ([]byte, int, error) {
	requestMethod := method
	if method == http.MethodPost {
		requestMethod = "REPORT"
	}

	req, err := http.NewRequestWithContext(ctx, requestMethod, calendarURL, strings.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	if username != "" || password != "" {
		req.SetBasicAuth(username, password)
	}
	req.Header.Set("User-Agent", "MeerkatCRM/1.0 CalDAV Sync")
	if requestMethod == "REPORT" {
		req.Header.Set("Depth", "1")
		req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	limitedReader := io.LimitReader(resp.Body, maxCalDAVResponseSize+1)
	respBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, 0, err
	}
	if len(respBody) > maxCalDAVResponseSize {
		return nil, 0, fmt.Errorf("calendar response is too large")
	}

	return respBody, resp.StatusCode, nil
}

func extractCalendarPayloads(responseBody []byte) []string {
	raw := string(responseBody)
	if strings.Contains(raw, "BEGIN:VCALENDAR") && !strings.Contains(raw, "<") {
		return []string{raw}
	}

	var payloads []string
	decoder := xml.NewDecoder(bytes.NewReader(responseBody))
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			if strings.Contains(raw, "BEGIN:VCALENDAR") {
				return []string{raw}
			}
			return nil
		}

		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "calendar-data" {
			continue
		}

		var payload string
		if err := decoder.DecodeElement(&payload, &start); err != nil {
			return nil
		}
		if strings.TrimSpace(payload) != "" {
			payloads = append(payloads, payload)
		}
	}

	if len(payloads) == 0 && strings.Contains(raw, "BEGIN:VCALENDAR") {
		return []string{raw}
	}
	return payloads
}

func parseICalendarEvents(payload string) ([]calDAVEvent, error) {
	lines := unfoldICalLines(payload)
	var events []calDAVEvent
	var current map[string]icalProperty
	inEvent := false

	for _, line := range lines {
		upperLine := strings.ToUpper(strings.TrimSpace(line))
		switch upperLine {
		case "BEGIN:VEVENT":
			current = make(map[string]icalProperty)
			inEvent = true
			continue
		case "END:VEVENT":
			if inEvent {
				if event, ok := buildCalDAVEvent(current); ok {
					events = append(events, event)
				}
			}
			current = nil
			inEvent = false
			continue
		}

		if !inEvent {
			continue
		}
		property, ok := parseICalProperty(line)
		if ok {
			if _, exists := current[property.Name]; !exists {
				current[property.Name] = property
			}
		}
	}

	return events, nil
}

type icalProperty struct {
	Name   string
	Params map[string]string
	Value  string
}

func parseICalProperty(line string) (icalProperty, bool) {
	separator := strings.Index(line, ":")
	if separator < 0 {
		return icalProperty{}, false
	}

	left := line[:separator]
	value := line[separator+1:]
	parts := strings.Split(left, ";")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return icalProperty{}, false
	}

	property := icalProperty{
		Name:   strings.ToUpper(strings.TrimSpace(parts[0])),
		Params: make(map[string]string),
		Value:  unescapeICalValue(value),
	}
	for _, part := range parts[1:] {
		key, val, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		property.Params[strings.ToUpper(strings.TrimSpace(key))] = strings.Trim(strings.TrimSpace(val), `"`)
	}

	return property, true
}

func buildCalDAVEvent(properties map[string]icalProperty) (calDAVEvent, bool) {
	start, ok := properties["DTSTART"]
	if !ok {
		return calDAVEvent{}, false
	}

	eventDate, err := parseICalDate(start.Value, start.Params)
	if err != nil {
		return calDAVEvent{}, false
	}

	title := strings.TrimSpace(properties["SUMMARY"].Value)
	if title == "" {
		title = "Untitled calendar event"
	}

	return calDAVEvent{
		Title:       clampString(title, 200),
		Description: clampString(properties["DESCRIPTION"].Value, 2000),
		Location:    clampString(properties["LOCATION"].Value, 300),
		Date:        eventDate,
	}, true
}

func parseICalDate(value string, params map[string]string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("empty iCalendar date")
	}

	if params["VALUE"] == "DATE" || (len(value) == len("20060102") && !strings.Contains(value, "T")) {
		parsed, err := time.Parse("20060102", value)
		if err != nil {
			return time.Time{}, err
		}
		return parsed.UTC(), nil
	}

	if strings.HasSuffix(value, "Z") {
		parsed, err := time.Parse("20060102T150405Z", value)
		if err != nil {
			return time.Time{}, err
		}
		return parsed.UTC(), nil
	}

	location := time.UTC
	if timezoneID := params["TZID"]; timezoneID != "" {
		if loadedLocation, err := time.LoadLocation(timezoneID); err == nil {
			location = loadedLocation
		}
	}

	for _, layout := range []string{"20060102T150405", "20060102T1504"} {
		parsed, err := time.ParseInLocation(layout, value, location)
		if err == nil {
			return parsed.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid iCalendar date %q", value)
}

func unfoldICalLines(payload string) []string {
	normalized := strings.ReplaceAll(payload, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	var lines []string
	for _, line := range strings.Split(normalized, "\n") {
		if line == "" {
			continue
		}
		if (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) && len(lines) > 0 {
			lines[len(lines)-1] += strings.TrimLeft(line, " \t")
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func unescapeICalValue(value string) string {
	replacer := strings.NewReplacer(
		`\\`, `\`,
		`\n`, "\n",
		`\N`, "\n",
		`\,`, ",",
		`\;`, ";",
	)
	return replacer.Replace(strings.TrimSpace(value))
}

func uniqueUintIDs(ids []uint) []uint {
	if len(ids) == 0 {
		return nil
	}

	seen := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		if id > 0 {
			seen[id] = struct{}{}
		}
	}

	unique := make([]uint, 0, len(seen))
	for id := range seen {
		unique = append(unique, id)
	}
	sort.Slice(unique, func(i, j int) bool { return unique[i] < unique[j] })
	return unique
}

func loadUserContacts(db *gorm.DB, userID uint, contactIDs []uint) ([]models.Contact, error) {
	if len(contactIDs) == 0 {
		return nil, nil
	}

	var contacts []models.Contact
	if err := db.Where("user_id = ? AND id IN ?", userID, contactIDs).Find(&contacts).Error; err != nil {
		return nil, err
	}
	if len(contacts) != len(contactIDs) {
		return nil, fmt.Errorf("one or more contacts not found for current user")
	}
	return contacts, nil
}

func clampString(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}
