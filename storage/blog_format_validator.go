package storage

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Validate checks required fields; empty tags are allowed.
func (p *PostMeta) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("name is required")
	}
	if !slugPattern.MatchString(p.Slug) {
		return errors.New("slug must be lowercase letters, digits or hyphens, no leading/trailing hyphen")
	}
	// Tags may be empty or contain empty strings; no validation.
	if p.Date.After(time.Now()) {
		return errors.New("date cannot be in the future")
	}
	if strings.TrimSpace(p.Description) == "" {
		return errors.New("description is required")
	}
	return nil
}
