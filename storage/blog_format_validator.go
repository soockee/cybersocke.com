package storage

import (
	"strings"
	"time"
)

// Shared parsing helpers for timestamp and date fields.
// Validation logic lives in storage/validation package.

// parseTimestamp tries several layouts to parse a string into time.Time
func parseTimestamp(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	layouts := []string{time.RFC3339, "2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02"}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, raw); err == nil {
			return ts
		}
	}
	return time.Time{}
}

// parseAnyTimestamp attempts to parse a timestamp from an arbitrary frontmatter field.
// Accepts string or time.Time; returns zero value if parsing fails or type unsupported.
// parseDate parses a strict date (YYYY-MM-DD) and returns zero time on failure.
func parseDate(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	ts, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}
	}
	return ts
}
