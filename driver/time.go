package driver

import (
	"database/sql/driver"
	"time"
)

// Convert a string in [time.RFC3339Nano] format into a [time.Time]
// if it roundtrips back to the same string.
// This way times can be persisted to, and recovered from, the database,
// but if a string is needed, [database.sql] will recover the same string.
// TODO: optimize and fuzz test.
func maybeDate(text string) driver.Value {
	date, err := time.Parse(time.RFC3339Nano, text)
	if err == nil && date.Format(time.RFC3339Nano) == text {
		return date
	}
	return text
}
