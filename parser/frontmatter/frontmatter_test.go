package frontmatter

import (
	"strings"
	"testing"
)

// TestParseStripsFrontmatter verifies that the parser returns only the body content
// (excluding the YAML frontmatter block) and populates the provided struct.
func TestParseStripsFrontmatter(t *testing.T) {
	md := "---\ncreated: 2025-11-21\ntags:\n  - a/b\nlead: Example Lead\n---\n\n# Heading\n\nBody text here.\n"
	var meta struct {
		Created string   `yaml:"created"`
		Tags    []string `yaml:"tags"`
		Lead    string   `yaml:"lead"`
	}
	body, err := Parse(strings.NewReader(md), &meta)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(meta.Tags) != 1 || meta.Tags[0] != "a/b" {
		t.Fatalf("expected tags parsed, got %#v", meta.Tags)
	}
	if meta.Lead != "Example Lead" {
		t.Fatalf("expected lead parsed, got %q", meta.Lead)
	}
	if strings.Contains(string(body), "created:") || strings.Contains(string(body), "lead:") || strings.Contains(string(body), "tags:") {
		t.Fatalf("body still contains frontmatter: %q", string(body))
	}
	if !strings.Contains(string(body), "# Heading") {
		t.Fatalf("body missing markdown heading; got: %q", string(body))
	}
}
