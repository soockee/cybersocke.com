package validation

import (
	"strings"
	"time"
)

// parseTimestamp tries several layouts to parse a string into time.Time.
// Returns zero time on failure.
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

// DeriveDisplayName builds a human-friendly Title Case name from a slug.
// Example: "my-first-post.md" -> "My First Post".
func DeriveDisplayName(slug string) string {
	if slug == "" {
		return ""
	}
	base := strings.TrimSuffix(slug, ".md")
	parts := strings.Split(base, "-")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}

// SanitizeFilename converts original filename to a valid slug (lowercase, kebab-case, .md suffix).
func SanitizeFilename(filename string) string {
	slug := strings.TrimSpace(filename)
	slug = strings.ToLower(slug)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	if !strings.HasSuffix(slug, ".md") {
		slug += ".md"
	}
	return slug
}
