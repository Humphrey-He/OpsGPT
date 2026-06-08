package parser

import (
	"fmt"
	"strings"
	"unicode"
)

// Chunk represents a text chunk with its position in the original document
type Chunk struct {
	Content   string
	Index     int
	StartChar int
	EndChar   int
	Metadata  ChunkMetadata
}

// ChunkMetadata contains metadata about the chunk
type ChunkMetadata struct {
	Source         string
	DocTitle       string
	TotalChunks    int
	ChunkSize      int
	HasOverlap     bool
}

// Chunker interface for text chunking strategies
type Chunker interface {
	Chunkize(text string, metadata ChunkMetadata) ([]Chunk, error)
}

// Config holds chunking configuration
type ChunkerConfig struct {
	ChunkSize    int // Target size for each chunk in characters
	ChunkOverlap int // Overlap between chunks in characters
	MinChunkSize int // Minimum chunk size
	MaxChunkSize int // Maximum chunk size
}

// DefaultChunkerConfig returns default chunking configuration
func DefaultChunkerConfig() ChunkerConfig {
	return ChunkerConfig{
		ChunkSize:    512,
		ChunkOverlap: 50,
		MinChunkSize: 100,
		MaxChunkSize: 1024,
	}
}

// SlidingWindowChunker implements sliding window text chunking
type SlidingWindowChunker struct {
	config ChunkerConfig
}

// NewSlidingWindowChunker creates a new sliding window chunker
func NewSlidingWindowChunker(config ChunkerConfig) *SlidingWindowChunker {
	if config.ChunkSize <= 0 {
		config = DefaultChunkerConfig()
	}
	return &SlidingWindowChunker{config: config}
}

// Chunkize splits text into overlapping chunks using sliding window
func (c *SlidingWindowChunker) Chunkize(text string, metadata ChunkMetadata) ([]Chunk, error) {
	if len(text) == 0 {
		return nil, nil
	}

	var chunks []Chunk
	textLength := len(text)
	stepSize := c.config.ChunkSize - c.config.ChunkOverlap

	if stepSize <= 0 {
		return nil, fmt.Errorf("chunk overlap must be smaller than chunk size")
	}

	start := 0
	index := 0

	for start < textLength {
		end := start + c.config.ChunkSize
		if end > textLength {
			end = textLength
		}

		// Try to break at sentence or paragraph boundary
		breakPoint := c.findBreakPoint(text, start, end)

		content := strings.TrimSpace(text[start:breakPoint])

		// Only add non-empty chunks
		if len(content) >= c.config.MinChunkSize || start+c.config.ChunkSize >= textLength {
			chunk := Chunk{
				Content: content,
				Index:   index,
				StartChar: start,
				EndChar:   breakPoint,
				Metadata: ChunkMetadata{
					Source:      metadata.Source,
					DocTitle:    metadata.DocTitle,
					TotalChunks: 0, // Will be set after all chunks are created
					ChunkSize:   len(content),
					HasOverlap:  start > 0,
				},
			}
			chunks = append(chunks, chunk)
			index++
		}

		start += stepSize

		// Ensure progress
		if start >= textLength {
			break
		}
	}

	// Update total chunks count
	totalChunks := len(chunks)
	for i := range chunks {
		chunks[i].Metadata.TotalChunks = totalChunks
	}

	return chunks, nil
}

// findBreakPoint finds the best place to break text (at sentence or paragraph boundary)
func (c *SlidingWindowChunker) findBreakPoint(text string, start, end int) int {
	if end >= len(text) {
		return end
	}

	// Look for paragraph break (\n\n)
	paraBreak := strings.LastIndex(text[start:end], "\n\n")
	if paraBreak != -1 && paraBreak > c.config.ChunkSize/4 {
		return start + paraBreak
	}

	// Look for sentence break (. ! ?)
	sentenceBreak := -1
	for i := end - 1; i > start+c.config.ChunkSize/2 && i < len(text); i-- {
		if isSentenceEnding(text[i]) && i+1 < len(text) && isWhitespace(text[i+1]) {
			sentenceBreak = i + 1
			break
		}
	}
	if sentenceBreak != -1 && sentenceBreak-start >= c.config.MinChunkSize {
		return sentenceBreak
	}

	// Look for line break
	lineBreak := strings.LastIndex(text[start:end], "\n")
	if lineBreak != -1 && lineBreak > c.config.ChunkSize/4 {
		return start + lineBreak
	}

	// Look for word boundary (space)
	wordBreak := strings.LastIndex(text[start:end], " ")
	if wordBreak != -1 {
		return start + wordBreak
	}

	return end
}

// isSentenceEnding checks if a character ends a sentence
func isSentenceEnding(r rune) bool {
	return r == '.' || r == '!' || r == '?' || r == '。' || r == '！' || r == '？'
}

// isWhitespace checks if a rune is whitespace
func isWhitespace(r rune) bool {
	return unicode.IsSpace(r)
}

// RecursiveChunker implements recursive character-based chunking with boundary awareness
type RecursiveChunker struct {
	config ChunkerConfig
	separators []string
}

// NewRecursiveChunker creates a new recursive chunker
func NewRecursiveChunker(config ChunkerConfig) *RecursiveChunker {
	return &RecursiveChunker{
		config: config,
		separators: []string{
			"\n\n",   // Paragraph
			"\n",     // Line
			". ",     // Sentence
			" ",      // Word
			"",       // Character (fallback)
		},
	}
}

// Chunkize splits text into chunks recursively
func (c *RecursiveChunker) Chunkize(text string, metadata ChunkMetadata) ([]Chunk, error) {
	if len(text) == 0 {
		return nil, nil
	}

	var chunks []Chunk
	c.recursiveSplit(text, metadata, c.separators, 0, &chunks, 0)

	// Update indices and total count
	totalChunks := len(chunks)
	for i := range chunks {
		chunks[i].Index = i
		chunks[i].Metadata.TotalChunks = totalChunks
	}

	return chunks, nil
}

// recursiveSplit recursively splits text until chunks are small enough
func (c *RecursiveChunker) recursiveSplit(text string, metadata ChunkMetadata, separators []string, sepIndex int, chunks *[]Chunk, startChar int) {
	if len(separators) == 0 || sepIndex >= len(separators) {
		// Base case: add remaining text as chunk
		if len(text) > 0 {
			*chunks = append(*chunks, Chunk{
				Content:   strings.TrimSpace(text),
				Index:     len(*chunks),
				StartChar: startChar,
				EndChar:   startChar + len(text),
				Metadata: ChunkMetadata{
					Source:      metadata.Source,
					DocTitle:    metadata.DocTitle,
					ChunkSize:   len(text),
					HasOverlap:  false,
				},
			})
		}
		return
	}

	separator := separators[sepIndex]
	textLength := len(text)

	// If text is small enough, just add it
	if textLength <= c.config.ChunkSize {
		*chunks = append(*chunks, Chunk{
			Content:   strings.TrimSpace(text),
			Index:     len(*chunks),
			StartChar: startChar,
			EndChar:   startChar + textLength,
			Metadata: ChunkMetadata{
				Source:      metadata.Source,
				DocTitle:    metadata.DocTitle,
				ChunkSize:   textLength,
				HasOverlap:  false,
			},
		})
		return
	}

	// Split by separator
	parts := strings.Split(text, separator)

	currentChunk := ""
	currentStart := startChar

	for i, part := range parts {
		// Calculate separator length for position tracking
		sepLen := len(separator)

		if len(currentChunk)+len(part)+sepLen <= c.config.ChunkSize {
			currentChunk += part + separator
		} else {
			// Save current chunk if non-empty
			if len(strings.TrimSpace(currentChunk)) > 0 {
				*chunks = append(*chunks, Chunk{
					Content:   strings.TrimSpace(currentChunk),
					Index:     len(*chunks),
					StartChar: currentStart,
					EndChar:   currentStart + len(currentChunk),
					Metadata: ChunkMetadata{
						Source:      metadata.Source,
						DocTitle:    metadata.DocTitle,
						ChunkSize:   len(currentChunk),
						HasOverlap:  false,
					},
				})
			}

			// Start new chunk with this part
			currentChunk = part + separator
			currentStart = startChar + strings.Index(text[currentStart-startChar:], part)
		}

		// If part is too large, recursively split
		if len(part) > c.config.ChunkSize && sepIndex+1 < len(separators) {
			// Save current chunk
			if len(strings.TrimSpace(currentChunk)) > 0 {
				*chunks = append(*chunks, Chunk{
					Content:   strings.TrimSpace(currentChunk),
					Index:     len(*chunks),
					StartChar: currentStart,
					EndChar:   currentStart + len(currentChunk),
					Metadata: ChunkMetadata{
						Source:      metadata.Source,
						DocTitle:    metadata.DocTitle,
						ChunkSize:   len(currentChunk),
						HasOverlap:  false,
					},
				})
				currentChunk = ""
			}

			// Recursively split this part
			partStart := startChar
			for j := 0; j < i; j++ {
				partStart += len(parts[j]) + sepLen
			}
			c.recursiveSplit(part, metadata, separators, sepIndex+1, chunks, partStart)
		}
	}

	// Add remaining chunk
	if len(strings.TrimSpace(currentChunk)) > 0 {
		*chunks = append(*chunks, Chunk{
			Content:   strings.TrimSpace(currentChunk),
			Index:     len(*chunks),
			StartChar: currentStart,
			EndChar:   startChar + len(text),
			Metadata: ChunkMetadata{
				Source:      metadata.Source,
				DocTitle:    metadata.DocTitle,
				ChunkSize:   len(currentChunk),
				HasOverlap:  false,
			},
		})
	}
}

// ChunkDocument chunks a document using the specified strategy
func ChunkDocument(doc *Document, config ChunkerConfig) ([]Chunk, error) {
	chunker := NewSlidingWindowChunker(config)
	return chunker.Chunkize(doc.Content, ChunkMetadata{
		Source:   doc.Metadata.Source,
		DocTitle: doc.Metadata.Title,
	})
}

// ChunkDocuments chunks multiple documents
func ChunkDocuments(docs []*Document, config ChunkerConfig) ([]Chunk, error) {
	var allChunks []Chunk

	for _, doc := range docs {
		chunks, err := ChunkDocument(doc, config)
		if err != nil {
			return nil, fmt.Errorf("failed to chunk document %s: %w", doc.Metadata.Source, err)
		}
		allChunks = append(allChunks, chunks...)
	}

	return allChunks, nil
}
