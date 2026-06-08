package vector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/v2"
	"github.com/qdrant/go-client/v2/qdrant"
)

// QdrantClient is a client for Qdrant vector database
type QdrantClient struct {
	client     *qdrant.Client
	collection string
	vectorSize int
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
		VectorSize: 768, // Default for nomic-embed-text
	}
}

// NewQdrantClient creates a new Qdrant client
func NewQdrantClient(config QdrantConfig) (*QdrantClient, error) {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: getHost(config.URL),
		Port: getPort(config.URL),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	return &QdrantClient{
		client:     client,
		collection: config.Collection,
		vectorSize: config.VectorSize,
	}, nil
}

// getHost extracts host from URL
func getHost(url string) string {
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")
	parts := strings.Split(host, ":")
	return parts[0]
}

// getPort extracts port from URL
func getPort(url string) int {
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")
	parts := strings.Split(host, ":")
	if len(parts) > 1 {
		var port int
		fmt.Sscanf(parts[1], "%d", &port)
		return int(port)
	}
	return 6333
}

// Point represents a vector point in Qdrant
type Point struct {
	ID       string
	Vector   []float32
	Payload  map[string]interface{}
}

// SearchResult represents a search result
type SearchResult struct {
	ID       string
	Score    float32
	Payload  map[string]interface{}
	Content  string
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

	// Convert points to Qdrant format
	qdrantPoints := make([]*qdrant.PointStruct, len(points))
	for i, p := range points {
		qdrantPoints[i] = &qdrant.PointStruct{
			Id:      &qdrant.PointId{Id: &qdrant.PointId_Uuid{Uuid: p.ID}},
			Vector:  convertVector(p.Vector),
			Payload: convertPayload(p.Payload),
		}
	}

	_, err := c.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: c.collection,
		Points:        qdrantPoints,
	})

	if err != nil {
		return fmt.Errorf("failed to upsert points: %w", err)
	}

	return nil
}

// Search searches for similar vectors
func (c *QdrantClient) Search(ctx context.Context, queryVector []float32, limit int, filters *Filter) ([]SearchResult, error) {
	// Ensure collection exists
	if err := c.EnsureCollection(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure collection: %w", err)
	}

	searchParams := &qdrant.SearchParams{}

	var filter *qdrant.Filter
	if filters != nil {
		filter = filters.ToQdrant()
	}

	resp, err := c.client.Search(ctx, &qdrant.SearchPoints{
		CollectionName: c.collection,
		Vector:         convertVector(queryVector),
		Limit:          uint64(limit),
		Params:         searchParams,
		WithPayload: &qdrant.WithPayloadSelector{
			Selector: &qdrant.PayloadSelectorInclude{
				Includes: []string{"content", "source", "title", "chunk_index", "metadata"},
			},
		},
		Filter: filter,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	results := make([]SearchResult, len(resp.Result))
	for i, r := range resp.Result {
		results[i] = SearchResult{
			ID:      extractID(r.Id),
			Score:   r.Score,
			Payload: convertPayloadFromProto(r.Payload),
			Content: extractContent(r.Payload),
		}
	}

	return results, nil
}

// SearchWithScoreThreshold searches with a minimum score threshold
func (c *QdrantClient) SearchWithScoreThreshold(ctx context.Context, queryVector []float32, limit int, scoreThreshold float32, filters *Filter) ([]SearchResult, error) {
	// Ensure collection exists
	if err := c.EnsureCollection(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure collection: %w", err)
	}

	var filter *qdrant.Filter
	if filters != nil {
		filter = filters.ToQdrant()
	}

	resp, err := c.client.Search(ctx, &qdrant.SearchPoints{
		CollectionName: c.collection,
		Vector:         convertVector(queryVector),
		Limit:          uint64(limit),
		ScoreThreshold: &qdrant.ScoreThreshold{
			Value: scoreThreshold,
		},
		WithPayload: &qdrant.WithPayloadSelector{
			Selector: &qdrant.PayloadSelectorInclude{
				Includes: []string{"content", "source", "title", "chunk_index", "metadata"},
			},
		},
		Filter: filter,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	results := make([]SearchResult, len(resp.Result))
	for i, r := range resp.Result {
		results[i] = SearchResult{
			ID:      extractID(r.Id),
			Score:   r.Score,
			Payload: convertPayloadFromProto(r.Payload),
			Content: extractContent(r.Payload),
		}
	}

	return results, nil
}

// Delete removes points from the collection
func (c *QdrantClient) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	pointIDs := make([]*qdrant.PointId, len(ids))
	for i, id := range ids {
		pointIDs[i] = &qdrant.PointId{Id: &qdrant.PointId_Uuid{Uuid: id}}
	}

	_, err := c.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: c.collection,
		Points:         pointIDs,
	})

	if err != nil {
		return fmt.Errorf("failed to delete points: %w", err)
	}

	return nil
}

// DeleteByFilter removes points matching a filter
func (c *QdrantClient) DeleteByFilter(ctx context.Context, filter *Filter) error {
	qdrantFilter := filter.ToQdrant()

	_, err := c.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: c.collection,
		PointsSelector: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Filter{
				Filter: qdrantFilter,
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to delete points by filter: %w", err)
	}

	return nil
}

// GetPoints retrieves points by IDs
func (c *QdrantClient) GetPoints(ctx context.Context, ids []string) ([]SearchResult, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	pointIDs := make([]*qdrant.PointId, len(ids))
	for i, id := range ids {
		pointIDs[i] = &qdrant.PointId{Id: &qdrant.PointId_Uuid{Uuid: id}}
	}

	resp, err := c.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: c.collection,
		Ids:            pointIDs,
		WithPayload: &qdrant.WithPayloadSelector{
			Selector: &qdrant.PayloadSelectorInclude{
				Includes: []string{"content", "source", "title", "chunk_index", "metadata"},
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get points: %w", err)
	}

	results := make([]SearchResult, len(resp.Result))
	for i, r := range resp.Result {
		results[i] = SearchResult{
			ID:      extractID(r.Id),
			Score:   1.0, // Default score for direct retrieval
			Payload: convertPayloadFromProto(r.Payload),
			Content: extractContent(r.Payload),
		}
	}

	return results, nil
}

// EnsureCollection creates the collection if it doesn't exist
func (c *QdrantClient) EnsureCollection(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Check if collection exists
	exists, err := c.client.CollectionExists(ctx, c.collection)
	if err != nil {
		return fmt.Errorf("failed to check collection: %w", err)
	}

	if exists {
		return nil
	}

	// Create collection
	_, err = c.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: c.collection,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     uint64(c.vectorSize),
					Distance: qdrant.Distance_Cosine,
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	// Wait for collection to be ready
	for i := 0; i < 30; i++ {
		info, err := c.client.GetCollectionInfo(ctx, c.collection)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if info.Status == qdrant.CollectionStatus_Green {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// DeleteCollection deletes the entire collection
func (c *QdrantClient) DeleteCollection(ctx context.Context) error {
	_, err := c.client.DeleteCollection(ctx, &qdrant.DeleteCollection{
		CollectionName: c.collection,
	})

	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	return nil
}

// GetCollectionInfo returns information about the collection
func (c *QdrantClient) GetCollectionInfo(ctx context.Context) (*CollectionInfo, error) {
	info, err := c.client.GetCollectionInfo(ctx, c.collection)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection info: %w", err)
	}

	return &CollectionInfo{
		Name:       c.collection,
		Points:     info.PointsCount,
		Vectors:    info.VectorsCount,
		Status:     info.Status.String(),
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
	Should []*Filter `json:"should,omitempty"`
	Must   []*Filter `json:"must,omitempty"`
	MustNot []*Filter `json:"must_not,omitempty"`
}

// FieldCondition creates a field-based filter condition
type FieldCondition struct {
	Key   string
	Value interface{}
	Type  string // "match", "range", "exists"
}

// ToQdrant converts a Filter to Qdrant filter format
func (f *Filter) ToQdrant() *qdrant.Filter {
	if f == nil {
		return nil
	}

	filter := &qdrant.Filter{}

	if len(f.Should) > 0 {
		for _, sf := range f.Should {
			subFilter := sf.ToQdrant()
			if subFilter != nil {
				filter.Should = append(filter.Should, subFilter)
			}
		}
	}

	if len(f.Must) > 0 {
		for _, mf := range f.Must {
			subFilter := mf.ToQdrant()
			if subFilter != nil {
				filter.Must = append(filter.Must, subFilter)
			}
		}
	}

	if len(f.MustNot) > 0 {
		for _, mnf := range f.MustNot {
			subFilter := mnf.ToQdrant()
			if subFilter != nil {
				filter.MustNot = append(filter.MustNot, subFilter)
			}
		}
	}

	return filter
}

// NewFilterMatch creates a match filter
func NewFilterMatch(key string, value interface{}) *Filter {
	return &Filter{
		Must: []*Filter{
			{
				Should: []*Filter{
					{
						Must: []*Filter{
							{
								Must: []*Filter{
									{
										Must: []*Filter{
											{
												Should: []*Filter{
													{
														Must: []*Filter{
															{
																Should: []*Filter{
																	{FieldCondition: &qdrant.FieldCondition{
																		Key: key,
																		Match: &qdrant.Match{
																			Value: value,
																		},
																	}},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// NewFilterMust creates a filter that must match
func NewFilterMust(key string, value interface{}) *Filter {
	return &Filter{
		Must: []*Filter{
			{
				Should: []*Filter{
					{
						Must: []*Filter{
							{
								FieldCondition: &qdrant.FieldCondition{
									Key: key,
									Match: &qdrant.Match{
										Value: value,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Helper functions

func convertVector(v []float32) []float64 {
	result := make([]float64, len(v))
	for i, f := range v {
		result[i] = float64(f)
	}
	return result
}

func convertPayload(p map[string]interface{}) *qdrant.Payload {
	if p == nil {
		return nil
	}

	payload := &qdrant.Payload{
		Fields: make(map[string]*qdrant.Value),
	}

	for k, v := range p {
		payload.Fields[k] = valueToProto(v)
	}

	return payload
}

func valueToProto(v interface{}) *qdrant.Value {
	switch val := v.(type) {
	case string:
		return &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: val}}
	case int:
		return &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(val)}}
	case int64:
		return &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: val}}
	case float32:
		return &qdrant.Value{Kind: &qdrant.Value_FloatValue{FloatValue: float64(val)}}
	case float64:
		return &qdrant.Value{Kind: &qdrant.Value_FloatValue{FloatValue: val}}
	case bool:
		return &qdrant.Value{Kind: &qdrant.Value_BoolValue{BoolValue: val}}
	case []string:
		return &qdrant.Value{Kind: &qdrant.Value_ListValue{
			ListValue: &qdrant.ListValue{
				Values: func() []*qdrant.Value {
					result := make([]*qdrant.Value, len(val))
					for i, s := range val {
						result[i] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: s}}
					}
					return result
				}(),
			},
		}}
	default:
		return &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: fmt.Sprintf("%v", v)}}
	}
}

func convertPayloadFromProto(p *qdrant.Payload) map[string]interface{} {
	if p == nil || p.Fields == nil {
		return nil
	}

	result := make(map[string]interface{})
	for k, v := range p.Fields {
		result[k] = protoToValue(v)
	}
	return result
}

func protoToValue(v *qdrant.Value) interface{} {
	if v == nil || v.Kind == nil {
		return nil
	}

	switch kind := v.Kind.(type) {
	case *qdrant.Value_StringValue:
		return kind.StringValue
	case *qdrant.Value_IntegerValue:
		return kind.IntegerValue
	case *qdrant.Value_FloatValue:
		return kind.FloatValue
	case *qdrant.Value_BoolValue:
		return kind.BoolValue
	case *qdrant.Value_ListValue:
		if kind.ListValue != nil {
			result := make([]interface{}, len(kind.ListValue.Values))
			for i, val := range kind.ListValue.Values {
				result[i] = protoToValue(val)
			}
			return result
		}
		return nil
	default:
		return nil
	}
}

func extractID(id *qdrant.PointId) string {
	if id == nil || id.Id == nil {
		return ""
	}
	switch uuid := id.Id.(type) {
	case *qdrant.PointId_Uuid:
		return uuid.Uuid
	case *qdrant.PointId_Num:
		return fmt.Sprintf("%d", uuid.Num)
	default:
		return uuid.String()
	}
}

func extractContent(payload *qdrant.Payload) string {
	if payload == nil || payload.Fields == nil {
		return ""
	}
	if content, ok := payload.Fields["content"]; ok {
		if str, ok := content.Kind.(*qdrant.Value_StringValue); ok {
			return str.StringValue
		}
	}
	return ""
}

// GeneratePointID generates a new point ID
func GeneratePointID() string {
	return uuid.New().String()
}

// IndexDocument indexes a document chunk
func (c *QdrantClient) IndexDocument(ctx context.Context, content string, vector []float32, metadata map[string]interface{}) error {
	id := GeneratePointID()

	point := Point{
		ID:      id,
		Vector:  vector,
		Payload: metadata,
	}

	// Ensure content is in payload
	if point.Payload == nil {
		point.Payload = make(map[string]interface{})
	}
	point.Payload["content"] = content

	return c.Upsert(ctx, []Point{point})
}
