package vector

import (
	"context"
	"testing"
)

func TestMockVectorStore_IndexAndSearch(t *testing.T) {
	store := NewMockVectorStore()
	ctx := context.Background()

	// Test indexing
	err := store.IndexDocument(ctx, "test content", []float32{0.1, 0.2, 0.3}, map[string]interface{}{
		"source": "test",
	})
	if err != nil {
		t.Fatalf("IndexDocument failed: %v", err)
	}

	// Test search
	results, err := store.Search(ctx, []float32{0.1, 0.2, 0.3}, 10, nil)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if results[0].Content != "test content" {
		t.Errorf("Expected 'test content', got '%s'", results[0].Content)
	}
}

func TestMockVectorStore_Count(t *testing.T) {
	store := NewMockVectorStore()
	ctx := context.Background()

	if store.Count() != 0 {
		t.Errorf("Expected count 0, got %d", store.Count())
	}

	// Add some documents
	store.IndexDocument(ctx, "doc1", []float32{0.1}, nil)
	store.IndexDocument(ctx, "doc2", []float32{0.2}, nil)

	if store.Count() != 2 {
		t.Errorf("Expected count 2, got %d", store.Count())
	}
}

func TestMockVectorStore_Clear(t *testing.T) {
	store := NewMockVectorStore()
	ctx := context.Background()

	store.IndexDocument(ctx, "doc1", []float32{0.1}, nil)
	store.IndexDocument(ctx, "doc2", []float32{0.2}, nil)

	store.Clear()

	if store.Count() != 0 {
		t.Errorf("Expected count 0 after clear, got %d", store.Count())
	}
}

func TestMockVectorStore_Limit(t *testing.T) {
	store := NewMockVectorStore()
	ctx := context.Background()

	// Add 5 documents
	for i := 0; i < 5; i++ {
		store.IndexDocument(ctx, "doc", []float32{float32(i) * 0.1}, nil)
	}

	// Search with limit 2
	results, err := store.Search(ctx, []float32{0.5}, 2, nil)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}
