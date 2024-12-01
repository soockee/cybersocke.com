package services

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

func GetPost(id string) []byte {
	return []byte(input)
}

var input = `
---
name: "frontmatter"
tags: ["go", "yaml"]
---
rest of the content
`

func RenderMD(source []byte) bytes.Buffer {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)
	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		panic(err)
	}
	return buf
}
