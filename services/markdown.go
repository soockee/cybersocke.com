package services

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// StripDataview removes dataview-style frontmatter/meta directives from the source markdown.
// It filters out lines beginning with known prefixes and also sanitizes certain patterns
// (e.g., Obsidian-style block IDs like "^id" and embedded HTML whitespace markers).
// The function returns the cleaned markdown content as a byte slice ready for rendering.
func StripDataview(source []byte) []byte {
	if len(source) == 0 {
		return source
	}
	lines := strings.Split(string(source), "\n")
	var out []string
	for _, line := range lines {
		if line == "" {
			out = append(out, line)
			continue
		}
		// Skip dataview / metadata directives
		if strings.HasPrefix(line, "dataview:") ||
			strings.HasPrefix(line, "note_tags:") ||
			strings.HasPrefix(line, "note_name:") ||
			strings.HasPrefix(line, "note_created:") ||
			strings.HasPrefix(line, "note_attachments:") ||
			strings.HasPrefix(line, "hide_gallery:") ||
			strings.HasPrefix(line, "draft:") ||
			strings.HasPrefix(line, "note_updated:") ||
			strings.HasPrefix(line, "creation_time:") ||
			strings.HasPrefix(line, "update_time:") {
			continue
		}
		// Remove obsidian block id anchors like ^blockId
		if strings.HasPrefix(line, "^") {
			continue
		}
		// Strip embedded html whitespace markers
		line = strings.ReplaceAll(line, "<html>", "")
		line = strings.ReplaceAll(line, "</html>", "")
		line = strings.ReplaceAll(line, "&nbsp;", " ")
		out = append(out, line)
	}
	return []byte(strings.Join(out, "\n"))
}

func RenderMD(source []byte) bytes.Buffer {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			html.WithUnsafe(),
		),
	)
	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		panic(err)
	}
	return buf
}
