package markdown

import (
	"bytes"
	"fmt"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

type MarkdownService interface {
	ToHTML(markdown string) (string, error)
	Sanitize(htmlContent string) string
	ToHTMLSanitized(markdown string) (string, error)
}

type markdownServiceImpl struct {
	md      goldmark.Markdown
	policy  *bluemonday.Policy
}

func NewMarkdownService() MarkdownService {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
			extension.Linkify,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)

	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("class").Matching(bluemonday.SpaceSeparatedTokens).OnElements("code", "span", "div", "pre")
	policy.AllowAttrs("id").Matching(bluemonday.SpaceSeparatedTokens).OnElements("h1", "h2", "h3", "h4", "h5", "h6")

	return &markdownServiceImpl{
		md:     md,
		policy: policy,
	}
}

func (s *markdownServiceImpl) ToHTML(markdown string) (string, error) {
	var buf bytes.Buffer
	if err := s.md.Convert([]byte(markdown), &buf); err != nil {
		return "", fmt.Errorf("failed to convert markdown to HTML: %w", err)
	}
	return buf.String(), nil
}

func (s *markdownServiceImpl) Sanitize(htmlContent string) string {
	return s.policy.Sanitize(htmlContent)
}

func (s *markdownServiceImpl) ToHTMLSanitized(markdown string) (string, error) {
	html, err := s.ToHTML(markdown)
	if err != nil {
		return "", err
	}
	return s.Sanitize(html), nil
}
