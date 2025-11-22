package storage

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/soockee/cybersocke.com/config"
	"github.com/soockee/cybersocke.com/storage/models"
	"github.com/soockee/cybersocke.com/storage/validation"
)

// Helper to create a valid baseline PostMeta for mutation
func validPostMeta() models.PostMeta {
	return models.PostMeta{
		Name:         "Test Post",
		Slug:         "test-post.md",
		Lead:         "A valid lead summary.",
		Tags:         []string{"type/article", "theme/knowledge"},
		CreatedRaw:   "2024-01-01",
		UpdatedRaw:   "2024-01-02T10:00:00Z",
		PublishedRaw: "true",
	}
}

func loadContract(t *testing.T) {
	if config.Contract != nil {
		return
	}
	data, err := os.ReadFile("schema/post_contract.yaml")
	if err != nil {
		// allow relative path from test execution root
		data, err = os.ReadFile("../schema/post_contract.yaml")
	}
	if err != nil {
		t.Fatalf("load contract file: %v", err)
	}
	c, err := config.ParsePostContract(data)
	if err != nil {
		t.Fatalf("parse contract: %v", err)
	}
	config.Contract = c
}

func TestValidate_PublishedValues(t *testing.T) {
	loadContract(t)
	tests := []struct {
		name         string
		publishedRaw string
		wantError    bool
		wantValue    bool
	}{
		{"true token", "true", false, true},
		{"false token", "false", false, false},
		{"empty string", "", false, false},
		{"yes disallowed", "yes", true, false},
		{"numeric disallowed", "1", true, false},
		{"on disallowed", "on", true, false},
		{"no disallowed", "no", true, false},
		{"zero disallowed", "0", true, false},
		{"off disallowed", "off", true, false},
		{"random token", "maybe", true, false},
	}
	for _, tt := range tests {
		p := validPostMeta()
		p.PublishedRaw = tt.publishedRaw
		err := validation.ValidateMeta(&p, config.Contract, "")
		if tt.wantError && err == nil {
			t.Errorf("expected error for published '%s', got nil", tt.publishedRaw)
		}
		if !tt.wantError && err != nil {
			t.Errorf("unexpected error for published '%s': %v", tt.publishedRaw, err)
		}
		if !tt.wantError && p.Published != tt.wantValue {
			t.Errorf("expected published=%v, got %v", tt.wantValue, p.Published)
		}
		if tt.wantError && err != nil && !strings.Contains(err.Error(), "invalid published value") {
			t.Errorf("expected 'invalid published value', got: %v", err)
		}
	}
}

func TestValidate_SlugPattern(t *testing.T) {
	loadContract(t)
	tests := []struct {
		name      string
		slug      string
		wantError bool
	}{
		{"valid slug", "valid-slug.md", false},
		{"double hyphen", "invalid--slug.md", true},
		{"leading hyphen", "-invalid-slug.md", true},
		{"trailing hyphen", "invalid-slug-.md", true},
		{"only extension", ".md", true},
	}
	for _, tt := range tests {
		p := validPostMeta()
		p.Slug = tt.slug
		err := validation.ValidateMeta(&p, config.Contract, "")
		if tt.wantError && err == nil {
			t.Errorf("expected error for slug '%s', got nil", tt.slug)
		}
		if !tt.wantError && err != nil {
			t.Errorf("unexpected error for slug '%s': %v", tt.slug, err)
		}
		if tt.wantError && err != nil && !strings.Contains(err.Error(), "slug must be lowercase kebab-case") {
			t.Errorf("expected slug pattern error, got: %v", err)
		}
	}
}

func TestValidate_LeadBounds(t *testing.T) {
	loadContract(t)
	tests := []struct {
		name      string
		lead      string
		wantError bool
		errMsg    string
	}{
		{"empty lead", "", true, "lead is required"},
		{"1 char ok", "A", false, ""},
		{"240 chars ok", strings.Repeat("a", 240), false, ""},
		{"241 chars fail", strings.Repeat("a", 241), true, "lead length exceeds 240"},
		{"whitespace only", "   ", true, "lead is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validPostMeta()
			p.Lead = tt.lead
			err := validation.ValidateMeta(&p, config.Contract, "")
			if tt.wantError && err == nil {
				t.Errorf("expected error for lead '%s', got nil", tt.name)
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error for lead '%s': %v", tt.name, err)
			}
			if tt.wantError && err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidate_LeadNewline(t *testing.T) {
	loadContract(t)
	tests := []struct {
		name      string
		lead      string
		wantError bool
	}{
		{"no newline", "Single line lead.", false},
		{"with newline", "First line\nSecond line", true},
		{"multiple newlines", "Line 1\n\nLine 3", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validPostMeta()
			p.Lead = tt.lead
			err := validation.ValidateMeta(&p, config.Contract, "")
			if tt.wantError && err == nil {
				t.Errorf("expected error for lead with newline, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantError && err != nil && !strings.Contains(err.Error(), "lead must be a single line") {
				t.Errorf("expected 'lead must be a single line', got: %v", err)
			}
		})
	}
}

func TestValidate_Timestamps(t *testing.T) {
	loadContract(t)
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	farFuture := now.Add(48 * time.Hour)

	tests := []struct {
		name       string
		createdRaw string
		updatedRaw string
		wantError  bool
		errMsg     string
	}{
		{
			name:       "created day after updated",
			createdRaw: now.Format("2006-01-02"),
			updatedRaw: yesterday.Format(time.RFC3339),
			wantError:  true,
			errMsg:     "created timestamp after updated timestamp",
		},
		{
			name:       "far-future updated",
			createdRaw: yesterday.Format("2006-01-02"),
			updatedRaw: farFuture.Format(time.RFC3339),
			wantError:  true,
			errMsg:     "updated timestamp cannot be in the far future",
		},
		{
			name:       "valid timestamps",
			createdRaw: yesterday.Format("2006-01-02"),
			updatedRaw: now.Format(time.RFC3339),
			wantError:  false,
		},
		{
			name:       "same day within skew",
			createdRaw: now.Format("2006-01-02"),
			updatedRaw: now.Add(-1 * time.Hour).Format(time.RFC3339),
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validPostMeta()
			p.CreatedRaw = tt.createdRaw
			p.UpdatedRaw = tt.updatedRaw
			err := validation.ValidateMeta(&p, config.Contract, "")
			if tt.wantError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantError && err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestDeriveDisplayName_Fallback(t *testing.T) {
	tests := []struct {
		slug string
		want string
	}{
		{"my-first-post.md", "My First Post"},
		{"hello-world.md", "Hello World"},
		{"single.md", "Single"},
		{"multiple-word-title.md", "Multiple Word Title"},
		{"", ""},
		{"no-extension", "No Extension"},
	}

	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			got := models.DeriveDisplayName(tt.slug)
			if got != tt.want {
				t.Errorf("DeriveDisplayName(%q) = %q, want %q", tt.slug, got, tt.want)
			}
		})
	}
}

func TestDeriveDisplayName_Fallback_Integration(t *testing.T) {
	loadContract(t)
	// Test that Validate derives Name from slug when Name is blank
	p := models.PostMeta{
		Name:         "", // blank name
		Slug:         "auto-derived-title.md",
		Lead:         "A valid lead summary.",
		Tags:         []string{"test"},
		CreatedRaw:   "2024-01-01",
		UpdatedRaw:   "2024-01-02T10:00:00Z",
		PublishedRaw: "true",
	}

	err := validation.ValidateMeta(&p, config.Contract, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Auto Derived Title"
	if p.Name != expected {
		t.Errorf("expected derived Name=%q, got %q", expected, p.Name)
	}
}
