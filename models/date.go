package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Custom type for handling dates in "YYYY-MM-DD" format
type Date time.Time

const DateFormat = "2006-01-02"

// Implementing custom unmarshalling for the Date type
func (d *Date) UnmarshalJSON(b []byte) error {
	// Remove the surrounding quotes from the JSON string
	dateStr := string(b)
	dateStr = dateStr[1 : len(dateStr)-1]

	// Parse the string into the custom date format
	parsedTime, err := time.Parse(DateFormat, dateStr)
	if err != nil {
		return fmt.Errorf("could not parse date: %v", err)
	}

	*d = Date(parsedTime)
	return nil
}

// Implement custom marshalling to convert Date back to "YYYY-MM-DD" format
func (d Date) MarshalJSON() ([]byte, error) {
	formatted := fmt.Sprintf("\"%s\"", time.Time(d).Format(DateFormat))
	return []byte(formatted), nil
}

// Helper method to convert custom Date type to time.Time
func (d Date) ToTime() time.Time {
	return time.Time(d)
}

// Implement the driver.Valuer interface for GORM to handle the Date type
func (d Date) Value() (driver.Value, error) {
	return time.Time(d), nil
}

// Implement the sql.Scanner interface to scan the value from the database
func (d *Date) Scan(value interface{}) error {
	if value == nil {
		*d = Date(time.Time{})
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		*d = Date(v)
		return nil
	default:
		return fmt.Errorf("cannot scan type %T into Date", value)
	}
}
