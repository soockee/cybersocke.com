package validation

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/soockee/cybersocke.com/config"
	"github.com/soockee/cybersocke.com/storage/models"
)

// ValidateMeta validates a PostMeta against the contract schema.
// Returns ValidationError for failures.
func ValidateMeta(meta *models.PostMeta, contract *config.PostContract, originalFilename string) error {
	if contract == nil {
		return NewValidationError(ErrCodeContract, "contract not loaded", "")
	}

	// Enforce slug matches sanitized filename if provided
	if originalFilename != "" {
		expectedSlug := SanitizeFilename(originalFilename)
		if meta.Slug != expectedSlug {
			return NewValidationError(ErrCodeSlug,
				fmt.Sprintf("slug %q does not match sanitized filename %q", meta.Slug, expectedSlug),
				"slug")
		}
	}

	// Derive Name from slug if blank
	if strings.TrimSpace(meta.Name) == "" && strings.TrimSpace(meta.Slug) != "" {
		meta.Name = DeriveDisplayName(meta.Slug)
	}

	// Build field map for efficient lookup
	fieldRules := make(map[string]config.FieldRule)
	for _, rule := range contract.Fields {
		fieldRules[rule.Name] = rule
	}

	// Validate Name
	if rule, ok := fieldRules["name"]; ok {
		if rule.Required && strings.TrimSpace(meta.Name) == "" {
			return NewValidationError(ErrCodeMissingField, "name is required", "name")
		}
		if rule.MaxLength > 0 && len(meta.Name) > rule.MaxLength {
			return NewValidationError(ErrCodeInvalidField,
				fmt.Sprintf("name length exceeds %d", rule.MaxLength), "name")
		}
		if rule.SingleLine && strings.Contains(meta.Name, "\n") {
			return NewValidationError(ErrCodeInvalidField, "name must be a single line", "name")
		}
	}

	// Validate Slug
	slugPattern := regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*\.md$`)
	if !slugPattern.MatchString(meta.Slug) {
		return NewValidationError(ErrCodeSlug, "slug must be lowercase kebab-case and end with .md", "slug")
	}
	if len(meta.Slug) > 128 {
		return NewValidationError(ErrCodeSlug, "slug exceeds max length 128", "slug")
	}

	// Validate Lead
	if rule, ok := fieldRules["lead"]; ok {
		meta.Lead = strings.TrimSpace(meta.Lead)
		if rule.Required && len(meta.Lead) == 0 {
			return NewValidationError(ErrCodeMissingField, "lead is required", "lead")
		}
		if rule.MaxLength > 0 && len(meta.Lead) > rule.MaxLength {
			return NewValidationError(ErrCodeInvalidField,
				fmt.Sprintf("lead length exceeds %d", rule.MaxLength), "lead")
		}
		if rule.SingleLine && strings.Contains(meta.Lead, "\n") {
			return NewValidationError(ErrCodeInvalidField, "lead must be a single line", "lead")
		}
	}

	// Validate Created (strict date YYYY-MM-DD)
	if rule, ok := fieldRules["created"]; ok {
		createdTs := parseDate(meta.CreatedRaw)
		if rule.Required && createdTs.IsZero() {
			return NewValidationError(ErrCodeTimestamp, "created date required (YYYY-MM-DD)", "created")
		}
		meta.Created = createdTs
	}

	// Validate Updated (flexible timestamp formats)
	if rule, ok := fieldRules["updated"]; ok {
		upd := parseTimestamp(meta.UpdatedRaw)
		if rule.Required && upd.IsZero() {
			return NewValidationError(ErrCodeTimestamp, "updated timestamp required", "updated")
		}
		if upd.After(time.Now().Add(24 * time.Hour)) {
			return NewValidationError(ErrCodeTimestamp, "updated timestamp cannot be in the far future", "updated")
		}
		if !meta.Created.IsZero() && meta.Created.After(upd.Add(2*time.Hour)) {
			return NewValidationError(ErrCodeTimestamp, "created timestamp after updated timestamp", "updated")
		}
		meta.Updated = upd
	}

	// Validate Published (strict boolean "true"/"false", blank = false)
	if rule, ok := fieldRules["published"]; ok {
		rawPub := strings.ToLower(strings.TrimSpace(meta.PublishedRaw))
		switch rawPub {
		case "", "false":
			meta.Published = false
		case "true":
			meta.Published = true
		default:
			if rule.Required {
				return NewValidationError(ErrCodeInvalidField, "invalid published value", "published")
			}
		}
	}

	// Validate Tags presence (cardinality checked separately by ValidateTags)
	if rule, ok := fieldRules["tags"]; ok {
		if rule.Required && len(meta.Tags) == 0 {
			return NewValidationError(ErrCodeMissingField, "tags are required", "tags")
		}
	}

	return nil
}

// ValidateTags enforces tag family rules from contract (family/value pattern, cardinalities).
func ValidateTags(meta *models.PostMeta, contract *config.PostContract) error {
	if contract == nil {
		return NewValidationError(ErrCodeContract, "contract not loaded", "tags")
	}

	// Build allowed families map
	allowedFamilies := make(map[string]config.TagFamilyRule)
	for family, rule := range contract.TagFamilies {
		allowedFamilies[family] = rule
	}

	counts := map[string]int{}
	unique := map[string]struct{}{}
	filtered := make([]string, 0, len(meta.Tags))

	for _, raw := range meta.Tags {
		t := strings.TrimSpace(raw)
		if t == "" {
			continue
		}
		if _, dup := unique[t]; dup {
			return NewValidationError(ErrCodeDuplicateTag, fmt.Sprintf("duplicate tag: %s", t), "tags")
		}
		unique[t] = struct{}{}

		// Enforce family/value pattern
		parts := strings.SplitN(t, "/", 2)
		if len(parts) != 2 {
			return NewValidationError(ErrCodeBadTag, fmt.Sprintf("tag %q must be family/value", t), "tags")
		}

		family := parts[0]
		if _, ok := allowedFamilies[family]; !ok {
			return NewValidationError(ErrCodeBadTag, fmt.Sprintf("unknown tag family: %s", family), "tags")
		}

		counts[family]++
		filtered = append(filtered, t)
	}

	// Enforce cardinality rules
	for family, rule := range allowedFamilies {
		count := counts[family]
		if count < rule.Min {
			return NewValidationError(ErrCodeBadTag,
				fmt.Sprintf("%s/* tags must be %d-%d (got %d)", family, rule.Min, rule.Max, count), "tags")
		}
		if count > rule.Max {
			return NewValidationError(ErrCodeBadTag,
				fmt.Sprintf("%s/* tags must be %d-%d (got %d)", family, rule.Min, rule.Max, count), "tags")
		}
	}

	meta.Tags = filtered // normalized
	return nil
}
