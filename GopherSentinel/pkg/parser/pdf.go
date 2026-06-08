package parser

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

// PDFParser parses PDF documents using basic text extraction
// Note: For production, consider using libraries like github.com/ledongthuc/pdf
type PDFParser struct{}

// PDF signature
var pdfSignature = []byte("%PDF-")

func (p *PDFParser) Supports(extension string) bool {
	return extension == ".pdf"
}

func (p *PDFParser) Parse(filePath string) (*Document, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF file: %w", err)
	}

	return p.ParseReader(bytes.NewReader(content), filePath)
}

func (p *PDFParser) ParseReader(reader io.Reader, filename string) (*Document, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	// Verify PDF signature
	if !bytes.HasPrefix(data, pdfSignature) {
		return nil, fmt.Errorf("invalid PDF file: missing PDF signature")
	}

	// Basic text extraction from PDF
	// This is a simplified approach - for production use a proper PDF library
	text := extractTextFromPDF(data)

	return &Document{
		Content:    text,
		RawContent: string(data),
		Metadata: DocumentMetadata{
			Title: extractPDFTitle(filename),
		},
	}, nil
}

// extractTextFromPDF extracts readable text from PDF bytes
// This is a simplified implementation - PDFs are complex binary format
func extractTextFromPDF(data []byte) string {
	// Find text between BT (Begin Text) and ET (End Text) markers
	var result strings.Builder

	inText := false
	currentLine := bytes.NewBuffer(nil)

	for i := 0; i < len(data)-1; i++ {
		// Check for BT (Begin Text)
		if i+1 < len(data) && data[i] == 'B' && data[i+1] == 'T' {
			// Skip the BT and any following whitespace
			inText = true
			i++
			continue
		}

		// Check for ET (End Text)
		if i+1 < len(data) && data[i] == 'E' && data[i+1] == 'T' {
			inText = false
			if currentLine.Len() > 0 {
				result.WriteString(strings.TrimSpace(currentLine.String()))
				result.WriteString("\n")
				currentLine.Reset()
			}
			i++
			continue
		}

		if inText {
			// Extract text between parentheses (Tj and TJ operators)
			if data[i] == '(' && i+1 < len(data) {
				end := findClosingParen(data, i+1)
				if end > i {
					text := string(data[i+1 : end])
					// Clean PDF string escape sequences
					text = cleanPDFString(text)
					currentLine.WriteString(text)
					i = end
				}
			}
		}
	}

	return strings.TrimSpace(result.String())
}

// findClosingParen finds the matching closing parenthesis
func findClosingParen(data []byte, start int) int {
	depth := 1
	for i := start; i < len(data) && depth > 0; i++ {
		if data[i] == '\\' && i+1 < len(data) {
			i++ // Skip escaped character
			continue
		}
		if data[i] == '(' {
			depth++
		} else if data[i] == ')' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// cleanPDFString removes PDF escape sequences from strings
func cleanPDFString(s string) string {
	// Remove common PDF escape sequences
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\r", "\r")
	s = strings.ReplaceAll(s, "\\t", "\t")
	s = strings.ReplaceAll(s, "\\(", "(")
	s = strings.ReplaceAll(s, "\\)", ")")
	s = strings.ReplaceAll(s, "\\\\", "\\")

	// Remove any non-printable characters
	var result strings.Builder
	for _, r := range s {
		if r >= 32 && r < 127 {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// extractPDFTitle extracts title from filename
func extractPDFTitle(filename string) string {
	base := strings.TrimSuffix(filename, ".pdf")
	base = strings.TrimPrefix(base, "./")
	base = strings.TrimPrefix(base, "/")
	return base
}
