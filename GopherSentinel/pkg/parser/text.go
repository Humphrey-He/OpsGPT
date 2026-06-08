package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

// TextParser parses plain text files
type TextParser struct{}

func (p *TextParser) Supports(extension string) bool {
	return extension == ".txt" || extension == ".text"
}

func (p *TextParser) Parse(filePath string) (*Document, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open text file: %w", err)
	}
	defer file.Close()

	return p.ParseReader(file, filePath)
}

func (p *TextParser) ParseReader(reader io.Reader, filename string) (*Document, error) {
	var buf bytes.Buffer
	scanner := bufio.NewScanner(reader)

	// Increase scanner buffer for larger lines
	const maxCapacity = 1024 * 1024 // 1MB
	buf.Grow(maxCapacity)
	scanner.Buffer(make([]byte, 64*1024), maxCapacity)

	for scanner.Scan() {
		buf.Write(scanner.Bytes())
		buf.WriteByte('\n')
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read text file: %w", err)
	}

	content := buf.String()

	// Extract title from first line if it looks like a title
	title := extractTitleFromContent(content, filename)

	return &Document{
		Content:    strings.TrimSpace(content),
		RawContent: content,
		Metadata: DocumentMetadata{
			Title: title,
		},
	}, nil
}

// extractTitleFromContent tries to extract a title from the content
func extractTitleFromContent(content, fallback string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	if scanner.Scan() {
		firstLine := strings.TrimSpace(scanner.Text())
		// If first line is short and doesn't end with punctuation, it might be a title
		if len(firstLine) > 0 && len(firstLine) < 100 && !strings.HasSuffix(firstLine, ".") {
			return firstLine
		}
	}
	return strings.TrimSuffix(fallback, ".txt")
}
