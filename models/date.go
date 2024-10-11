package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Custom type for handling dates in "YYYY-MM-DD" format
type Date struct {
	Time  time.Time
	Valid bool // Valid is true if Time is not NULL
}

const DateFormat = "2006-01-02"

// Implementing custom unmarshalling for the Date type
func (d *Date) UnmarshalJSON(b []byte) error {
	// Remove the surrounding quotes from the JSON string
	dateStr := string(b)
	if dateStr == "null" {
		d.Time, d.Valid = time.Time{}, false
		return nil
	}

	dateStr = dateStr[1 : len(dateStr)-1]

	// Parse the string into the custom date format
	parsedTime, err := time.Parse(DateFormat, dateStr)
	if err != nil {
		return fmt.Errorf("could not parse date: %v", err)
	}

	d.Time, d.Valid = parsedTime, true
	return nil
}

// Implement custom marshalling to convert Date back to "YYYY-MM-DD" format
func (d Date) MarshalJSON() ([]byte, error) {
	if !d.Valid {
		return []byte("null"), nil
	}
	formatted := fmt.Sprintf("\"%s\"", d.Time.Format(DateFormat))
	return []byte(formatted), nil
}

// Helper method to convert custom Date type to time.Time
func (d Date) ToTime() (time.Time, bool) {
	return d.Time, d.Valid
}

// Implement the driver.Valuer interface for GORM to handle the Date type
func (d Date) Value() (driver.Value, error) {
	if !d.Valid {
		return nil, nil
	}
	return d.Time, nil
}

// Implement the sql.Scanner interface to scan the value from the database
func (d *Date) Scan(value interface{}) error {
	if value == nil {
		d.Time, d.Valid = time.Time{}, false
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		d.Time, d.Valid = v, true
		return nil
	default:
		return fmt.Errorf("cannot scan type %T into Date", value)
	}
}
