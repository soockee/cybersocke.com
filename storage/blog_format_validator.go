package storage

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// slugPattern now includes the .md extension (required) per new spec.
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*\.md$`)

// Validate checks required fields; empty tags are allowed.
func (p *PostMeta) Validate() error {
	// Derive Name if missing from slug (strip extension, hyphens -> spaces, Title Case each segment for UI display)
	if strings.TrimSpace(p.Name) == "" && strings.TrimSpace(p.Slug) != "" {
		base := strings.TrimSuffix(p.Slug, ".md")
		parts := strings.Split(base, "-")
		for i, part := range parts {
			if len(part) > 0 {
				parts[i] = strings.ToUpper(part[:1]) + part[1:]
			}
		}
		p.Name = strings.Join(parts, " ")
	}
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("name is required")
	}
	if !slugPattern.MatchString(p.Slug) {
		return errors.New("slug must be lowercase kebab-case and end with .md")
	}
	// Tags may be empty or contain empty strings; no validation.
	// Assign Description from Lead if absent
	if strings.TrimSpace(p.Description) == "" && strings.TrimSpace(p.Lead) != "" {
		p.Description = p.Lead
	}
	if strings.TrimSpace(p.Description) == "" {
		return errors.New("description (or lead) is required")
	}

	// Canonical date comes solely from `updated`; ignore created/modified fields.
	updatedTs := parseTimestamp(p.UpdatedRaw)
	if updatedTs.IsZero() {
		return errors.New("updated timestamp required")
	}
	p.Date = updatedTs
	if p.Date.After(time.Now().Add(24 * time.Hour)) { // small clock skew tolerance
		return errors.New("date cannot be in the far future")
	}
	return nil
}

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
