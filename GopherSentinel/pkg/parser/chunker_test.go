package parser

import (
	"testing"
)

func TestSlidingWindowChunker_Chunkize(t *testing.T) {
	config := ChunkerConfig{
		ChunkSize:    100,
		ChunkOverlap: 20,
		MinChunkSize: 30,
		MaxChunkSize: 150,
	}
	chunker := NewSlidingWindowChunker(config)

	tests := []struct {
		name    string
		text    string
		wantMin int // minimum expected chunks
	}{
		{
			name:    "short text",
			text:    "This is a short text.",
			wantMin: 1,
		},
		{
			name:    "medium text",
			text:    "This is a medium length text that should be split into multiple chunks based on the chunk size configuration. We need enough content here to trigger chunking behavior.",
			wantMin: 2,
		},
		{
			name:    "long text with paragraphs",
			text:    "Paragraph one with some content.\n\nParagraph two with more content.\n\nParagraph three with additional content here.",
			wantMin: 1,
		},
		{
			name:    "empty text",
			text:    "",
			wantMin: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks, err := chunker.Chunkize(tt.text, ChunkMetadata{
				Source:   "test.txt",
				DocTitle: "Test",
			})
			if err != nil {
				t.Fatalf("Chunkize() error = %v", err)
			}

			if len(chunks) < tt.wantMin {
				t.Errorf("Chunkize() returned %d chunks, want at least %d", len(chunks), tt.wantMin)
			}

			// Verify chunk structure
			for i, chunk := range chunks {
				if chunk.Index != i {
					t.Errorf("Chunk[%d].Index = %d, want %d", i, chunk.Index, i)
				}
				if chunk.Metadata.Source != "test.txt" {
					t.Errorf("Chunk[%d].Source = %v, want test.txt", i, chunk.Metadata.Source)
				}
				if chunk.Metadata.TotalChunks != len(chunks) {
					t.Errorf("Chunk[%d].TotalChunks = %d, want %d", i, chunk.Metadata.TotalChunks, len(chunks))
				}
			}
		})
	}
}

func TestSlidingWindowChunker_Overlap(t *testing.T) {
	config := ChunkerConfig{
		ChunkSize:    50,
		ChunkOverlap: 20,
		MinChunkSize: 10,
		MaxChunkSize: 100,
	}
	chunker := NewSlidingWindowChunker(config)

	// Create text long enough to produce overlapping chunks
	text := "This is a test string that should be chunked with overlap. " +
		"This is a test string that should be chunked with overlap. " +
		"This is a test string that should be chunked with overlap. " +
		"This is a test string that should be chunked with overlap."

	chunks, err := chunker.Chunkize(text, ChunkMetadata{})
	if err != nil {
		t.Fatalf("Chunkize() error = %v", err)
	}

	if len(chunks) < 2 {
		t.Skip("Text too short for overlap test")
	}

	// Check that subsequent chunks have overlap flag set
	for i := 1; i < len(chunks); i++ {
		if !chunks[i].Metadata.HasOverlap {
			t.Errorf("Chunk[%d].HasOverlap = false, want true", i)
		}
	}
}

func TestDefaultChunkerConfig(t *testing.T) {
	config := DefaultChunkerConfig()

	if config.ChunkSize <= 0 {
		t.Error("ChunkSize should be positive")
	}
	if config.ChunkOverlap < 0 {
		t.Error("ChunkOverlap should be non-negative")
	}
	if config.MinChunkSize <= 0 {
		t.Error("MinChunkSize should be positive")
	}
	if config.MaxChunkSize < config.ChunkSize {
		t.Error("MaxChunkSize should be >= ChunkSize")
	}
}

func TestChunkDocument(t *testing.T) {
	doc := &Document{
		Content: "This is a test document with enough content to be chunked. " +
			"More content here. Even more content. Adding more text. " +
			"Final line of the document.",
		Metadata: DocumentMetadata{
			Source: "test.md",
			Title: "Test Document",
		},
	}

	config := DefaultChunkerConfig()
	chunks, err := ChunkDocument(doc, config)
	if err != nil {
		t.Fatalf("ChunkDocument() error = %v", err)
	}

	if len(chunks) == 0 {
		t.Error("ChunkDocument() returned empty chunks")
	}

	// Check that metadata is propagated
	for _, chunk := range chunks {
		if chunk.Metadata.Source != "test.md" {
			t.Errorf("Expected Source=test.md, got %s", chunk.Metadata.Source)
		}
		if chunk.Metadata.DocTitle != "Test Document" {
			t.Errorf("Expected DocTitle=Test Document, got %s", chunk.Metadata.DocTitle)
		}
	}
}
