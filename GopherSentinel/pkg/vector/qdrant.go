package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// QdrantClient is a client for Qdrant vector database using REST API
type QdrantClient struct {
	baseURL    string
	collection string
	vectorSize int
	httpClient *http.Client
}

// QdrantConfig holds configuration for the Qdrant client
type QdrantConfig struct {
	URL        string
	Collection string
	VectorSize int
}

// DefaultQdrantConfig returns default configuration
func DefaultQdrantConfig() QdrantConfig {
	return QdrantConfig{
		URL:        "http://localhost:6333",
		Collection: "gopher_sentinel",
		VectorSize: 768,
	}
}

// NewQdrantClient creates a new Qdrant client
func NewQdrantClient(config QdrantConfig) (*QdrantClient, error) {
	baseURL := strings.TrimSuffix(config.URL, "/")

	return &QdrantClient{
		baseURL:    baseURL,
		collection: config.Collection,
		vectorSize: config.VectorSize,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Point represents a vector point in Qdrant
type Point struct {
	ID      string
	Vector  []float32
	Payload map[string]interface{}
}

// SearchResult represents a search result
type SearchResult struct {
	ID      string
	Score   float32
	Payload map[string]interface{}
	Content string
}

// Upsert inserts or updates points in the collection
func (c *QdrantClient) Upsert(ctx context.Context, points []Point) error {
	if len(points) == 0 {
		return nil
	}

	// Ensure collection exists
	if err := c.EnsureCollection(ctx); err != nil {
		return fmt.Errorf("failed to ensure collection: %w", err)
	}

	// Prepare points for upsert
	upsertPoints := make([]map[string]interface{}, len(points))
	for i, p := range points {
		upsertPoints[i] = map[string]interface{}{
			"id":      p.ID,
			"vector":  p.Vector,
			"payload": p.Payload,
		}
	}

	body := map[string]interface{}{
		"points": upsertPoints,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points", c.baseURL, c.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upsert failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Search searches for similar vectors
func (c *QdrantClient) Search(ctx context.Context, queryVector []float32, limit int, filters *Filter) ([]SearchResult, error) {
	return c.SearchWithScoreThreshold(ctx, queryVector, limit, 0, filters)
}

// SearchWithScoreThreshold searches with a minimum score threshold
func (c *QdrantClient) SearchWithScoreThreshold(ctx context.Context, queryVector []float32, limit int, scoreThreshold float32, filters *Filter) ([]SearchResult, error) {
	// Ensure collection exists
	if err := c.EnsureCollection(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure collection: %w", err)
	}

	// Build request body
	body := map[string]interface{}{
		"vector": queryVector,
		"limit":  limit,
		"with_payload": map[string]bool{
			"enable": true,
		},
	}

	if scoreThreshold > 0 {
		body["score_threshold"] = scoreThreshold
	}

	if filters != nil {
		body["filter"] = filters.ToMap()
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/search", c.baseURL, c.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Result []struct {
			ID      string                 `json:"id"`
			Score   float32                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]SearchResult, len(result.Result))
	for i, r := range result.Result {
		content := ""
		if c, ok := r.Payload["content"].(string); ok {
			content = c
		}
		results[i] = SearchResult{
			ID:      r.ID,
			Score:   r.Score,
			Payload: r.Payload,
			Content: content,
		}
	}

	return results, nil
}

// Delete removes points from the collection
func (c *QdrantClient) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	body := map[string]interface{}{
		"points": ids,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/delete", c.baseURL, c.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// DeleteByFilter removes points matching a filter
func (c *QdrantClient) DeleteByFilter(ctx context.Context, filter *Filter) error {
	body := map[string]interface{}{
		"filter": filter.ToMap(),
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/delete", c.baseURL, c.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetPoints retrieves points by IDs
func (c *QdrantClient) GetPoints(ctx context.Context, ids []string) ([]SearchResult, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	body := map[string]interface{}{
		"ids":           ids,
		"with_payload":  true,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points", c.baseURL, c.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get points failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Result []struct {
			ID      string                 `json:"id"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]SearchResult, len(result.Result))
	for i, r := range result.Result {
		content := ""
		if c, ok := r.Payload["content"].(string); ok {
			content = c
		}
		results[i] = SearchResult{
			ID:      r.ID,
			Score:   1.0,
			Payload: r.Payload,
			Content: content,
		}
	}

	return results, nil
}

// EnsureCollection creates the collection if it doesn't exist
func (c *QdrantClient) EnsureCollection(ctx context.Context) error {
	// Check if collection exists
	url := fmt.Sprintf("%s/collections/%s", c.baseURL, c.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil // Collection exists
	}

	// Create collection
	createBody := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     c.vectorSize,
			"distance": "Cosine",
		},
	}

	jsonBody, err := json.Marshal(createBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url = fmt.Sprintf("%s/collections/%s", c.baseURL, c.collection)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	createResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusOK && createResp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(createResp.Body)
		return fmt.Errorf("create collection failed (status %d): %s", createResp.StatusCode, string(respBody))
	}

	// Wait for collection to be ready
	for i := 0; i < 30; i++ {
		info, err := c.GetCollectionInfo(ctx)
		if err == nil && info.Status == "green" {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// DeleteCollection deletes the entire collection
func (c *QdrantClient) DeleteCollection(ctx context.Context) error {
	url := fmt.Sprintf("%s/collections/%s", c.baseURL, c.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete collection failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetCollectionInfo returns information about the collection
func (c *QdrantClient) GetCollectionInfo(ctx context.Context) (*CollectionInfo, error) {
	url := fmt.Sprintf("%s/collections/%s", c.baseURL, c.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get collection info failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Result struct {
			Status string `json:"status"`
			PointsCount uint64 `json:"points_count"`
			VectorsCount uint64 `json:"vectors_count"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &CollectionInfo{
		Name:       c.collection,
		Points:     result.Result.PointsCount,
		Vectors:    result.Result.VectorsCount,
		Status:     result.Result.Status,
		VectorSize: uint64(c.vectorSize),
	}, nil
}

// CollectionInfo contains information about a collection
type CollectionInfo struct {
	Name       string
	Points     uint64
	Vectors    uint64
	Status     string
	VectorSize uint64
}

// Filter represents a filter for searching
type Filter struct {
	Should  []*Filter `json:"should,omitempty"`
	Must    []*Filter `json:"must,omitempty"`
	MustNot []*Filter `json:"must_not,omitempty"`
}

// ToMap converts Filter to a map for JSON serialization
func (f *Filter) ToMap() map[string]interface{} {
	if f == nil {
		return nil
	}

	result := make(map[string]interface{})

	if len(f.Should) > 0 {
		should := make([]map[string]interface{}, len(f.Should))
		for i, sf := range f.Should {
			should[i] = sf.ToMap()
		}
		result["should"] = should
	}

	if len(f.Must) > 0 {
		must := make([]map[string]interface{}, len(f.Must))
		for i, mf := range f.Must {
			must[i] = mf.ToMap()
		}
		result["must"] = must
	}

	if len(f.MustNot) > 0 {
		mustNot := make([]map[string]interface{}, len(f.MustNot))
		for i, mnf := range f.MustNot {
			mustNot[i] = mnf.ToMap()
		}
		result["must_not"] = mustNot
	}

	return result
}

// GeneratePointID generates a new point ID
func GeneratePointID() string {
	return uuid.New().String()
}

// IndexDocument indexes a document chunk
func (c *QdrantClient) IndexDocument(ctx context.Context, content string, vector []float32, metadata map[string]interface{}) error {
	id := GeneratePointID()

	// Ensure content is in metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["content"] = content

	point := Point{
		ID:      id,
		Vector:  vector,
		Payload: metadata,
	}

	return c.Upsert(ctx, []Point{point})
}

// IsAvailable checks if Qdrant is available
func (c *QdrantClient) IsAvailable(ctx context.Context) bool {
	url := fmt.Sprintf("%s/collections", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
