package parser

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// MarkdownParser parses Markdown documents
type MarkdownParser struct{}

// Markdown heading pattern
var headingRegex = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)

// Markdown code block pattern
var codeBlockRegex = regexp.MustCompile("```[\\s\\S]*?```")

// Markdown inline code pattern
var inlineCodeRegex = regexp.MustCompile("`[^`]+`")

func (p *MarkdownParser) Supports(extension string) bool {
	return extension == ".md" || extension == ".markdown"
}

func (p *MarkdownParser) Parse(filePath string) (*Document, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read markdown file: %w", err)
	}

	return p.ParseReader(bytes.NewReader(content), filePath)
}

func (p *MarkdownParser) ParseReader(reader io.Reader, filename string) (*Document, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	content := string(data)

	// Extract title from first heading or filename
	title := extractMarkdownTitle(content, filename)

	// Clean markdown content (remove code blocks for simpler processing, but keep structure)
	cleanedContent := cleanMarkdown(content)

	return &Document{
		Content:    cleanedContent,
		RawContent: content,
		Metadata: DocumentMetadata{
			Title: title,
		},
	}, nil
}

// extractMarkdownTitle extracts the title from markdown content
func extractMarkdownTitle(content, fallback string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		matches := headingRegex.FindStringSubmatch(strings.TrimSpace(line))
		if len(matches) == 3 {
			level := len(matches[1])
			if level == 1 { // Only h1
				return strings.TrimSpace(matches[2])
			}
		}
	}

	// Fallback to filename without extension
	base := strings.TrimSuffix(fallback, ".md")
	base = strings.TrimSuffix(base, ".markdown")
	return base
}

// cleanMarkdown removes code blocks and normalizes whitespace
func cleanMarkdown(content string) string {
	// Replace code blocks with placeholder (to preserve structure)
	result := codeBlockRegex.ReplaceAllString(content, "[CODE_BLOCK]")

	// Remove inline code markers but keep content
	result = inlineCodeRegex.ReplaceAllStringFunc(result, func(match string) string {
		// Remove backticks
		return match[1 : len(match)-1]
	})

	// Normalize headers (remove # symbols for plain text)
	lines := strings.Split(result, "\n")
	var cleanedLines []string
	for _, line := range lines {
		matches := headingRegex.FindStringSubmatch(strings.TrimSpace(line))
		if len(matches) == 3 {
			// Keep heading text but mark it
			cleanedLines = append(cleanedLines, "\n## "+matches[2]+"\n")
		} else {
			cleanedLines = append(cleanedLines, line)
		}
	}

	result = strings.Join(cleanedLines, "\n")

	// Normalize multiple blank lines
	result = regexp.MustCompile(`\n{3,}`).ReplaceAllString(result, "\n\n")

	return strings.TrimSpace(result)
}
