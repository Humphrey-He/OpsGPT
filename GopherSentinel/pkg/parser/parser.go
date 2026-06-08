package parser

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Document represents a parsed document with its content and metadata
type Document struct {
	Content     string
	Metadata    DocumentMetadata
	RawContent  string
}

// DocumentMetadata contains metadata about the document
type DocumentMetadata struct {
	Source      string
	Title       string
	FileType    string
	Size        int64
	ChunkCount  int
}

// Parser interface for document parsing
type Parser interface {
	Parse(filePath string) (*Document, error)
	ParseReader(reader io.Reader, filename string) (*Document, error)
	Supports(extension string) bool
}

// GetParser returns the appropriate parser for a file
func GetParser(filePath string) (Parser, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".md", ".markdown":
		return &MarkdownParser{}, nil
	case ".pdf":
		return &PDFParser{}, nil
	case ".txt", ".text":
		return &TextParser{}, nil
	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}

// ParseFile parses a file and returns a Document
func ParseFile(filePath string) (*Document, error) {
	parser, err := GetParser(filePath)
	if err != nil {
		return nil, err
	}

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	doc, err := parser.Parse(filePath)
	if err != nil {
		return nil, err
	}

	doc.Metadata.Size = fileInfo.Size()
	doc.Metadata.Source = filePath
	doc.Metadata.Title = filepath.Base(filePath)
	doc.Metadata.FileType = strings.TrimPrefix(filepath.Ext(filePath), ".")

	return doc, nil
}

// ParseDirectory parses all supported files in a directory recursively
func ParseDirectory(dirPath string) ([]*Document, error) {
	var documents []*Document

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Try to get a parser for this file
		parser, err := GetParser(path)
		if err != nil {
			// Unsupported file type, skip
			return nil
		}

		doc, err := parser.Parse(path)
		if err != nil {
			// Log error but continue
			fmt.Printf("Warning: failed to parse %s: %v\n", path, err)
			return nil
		}

		doc.Metadata.Source = path
		doc.Metadata.Title = filepath.Base(path)
		doc.Metadata.FileType = strings.TrimPrefix(filepath.Ext(path), ".")
		doc.Metadata.Size = info.Size()

		documents = append(documents, doc)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return documents, nil
}
