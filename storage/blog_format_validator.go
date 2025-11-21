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
	// Derive display Name from slug if missing.
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
	// Lead is now the required textual summary (description field removed from PostMeta).
	if strings.TrimSpace(p.Lead) == "" {
		return errors.New("lead is required")
	}
	// Tags optional; no validation beyond presence type.

	// Created timestamp required (per minimal frontmatter spec). Accept string or already-parsed time.Time.
	createdTs := parseAnyTimestamp(p.Created)
	if createdTs.IsZero() {
		return errors.New("created timestamp required")
	}
	// Store parsed time back (preserving original interface usage).
	p.Created = createdTs

	// Parse Updated from raw flexible string
	upd := parseTimestamp(p.UpdatedRaw)
	if upd.IsZero() {
		return errors.New("updated timestamp required")
	}
	if upd.After(time.Now().Add(24 * time.Hour)) {
		return errors.New("updated timestamp cannot be in the far future")
	}
	if createdTs.After(upd.Add(2 * time.Hour)) { // allow small skew
		return errors.New("created timestamp after updated timestamp")
	}
	p.Updated = upd

	// Parse Published from raw flexible value; treat blank as false.
	rawPub := strings.ToLower(strings.TrimSpace(p.PublishedRaw))
	switch rawPub {
	case "", "false", "no", "0", "off":
		p.Published = false
	case "true", "yes", "1", "on":
		p.Published = true
	default:
		// If the original YAML provided a quoted value that isn't a recognized boolean keyword, surface an error.
		return errors.New("invalid published value")
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

// parseAnyTimestamp attempts to parse a timestamp from an arbitrary frontmatter field.
// Accepts string or time.Time; returns zero value if parsing fails or type unsupported.
func parseAnyTimestamp(v any) time.Time {
	switch t := v.(type) {
	case time.Time:
		return t
	case string:
		return parseTimestamp(t)
	default:
		return time.Time{}
	}
}
