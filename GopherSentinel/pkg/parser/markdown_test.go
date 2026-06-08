package parser

import (
	"strings"
	"testing"
)

func TestMarkdownParser_Parse(t *testing.T) {
	parser := &MarkdownParser{}

	tests := []struct {
		name     string
		content  string
		wantTitle string
		wantContains string
	}{
		{
			name:     "with h1 title",
			content:  "# Hello World\n\nThis is content.",
			wantTitle: "Hello World",
			wantContains: "This is content",
		},
		{
			name:     "without title",
			content:  "Just some content here.",
			wantTitle: "test.md",
			wantContains: "Just some content here",
		},
		{
			name:     "multiple headings",
			content:  "# Title\n\n## Section 1\n\nContent 1\n\n## Section 2\n\nContent 2",
			wantTitle: "Title",
			wantContains: "Content 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := parser.ParseReader(strings.NewReader(tt.content), "test.md")
			if err != nil {
				t.Fatalf("ParseReader() error = %v", err)
			}

			if doc.Metadata.Title != tt.wantTitle {
				t.Errorf("Title = %v, want %v", doc.Metadata.Title, tt.wantTitle)
			}

			if !strings.Contains(doc.Content, tt.wantContains) {
				t.Errorf("Content = %v, want contains %v", doc.Content, tt.wantContains)
			}
		})
	}
}

func TestMarkdownParser_Supports(t *testing.T) {
	parser := &MarkdownParser{}

	tests := []struct {
		ext  string
		want bool
	}{
		{".md", true},
		{".markdown", true},
		{".txt", false},
		{".pdf", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			if got := parser.Supports(tt.ext); got != tt.want {
				t.Errorf("Supports(%v) = %v, want %v", tt.ext, got, tt.want)
			}
		})
	}
}
