package main

import (
	"time"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	Ollama struct {
		BaseURL        string
		Model          string
		EmbeddingModel string
		Timeout        int
	}
	Qdrant struct {
		URL        string
		Collection string
		VectorSize int
	}
	RAG struct {
		ChunkSize        int
		ChunkOverlap     int
		TopK             int
		MaxContextTokens int
	}
	CLI struct {
		Stream       bool
		HistoryLimit int
	}
}

func initConfig() *Config {
	// Set defaults
	config := &Config{
		Ollama: struct {
			BaseURL        string
			Model          string
			EmbeddingModel string
			Timeout        int
		}{
			BaseURL:        "http://localhost:11434",
			Model:          "llama3",
			EmbeddingModel: "nomic-embed-text",
			Timeout:        120,
		},
		Qdrant: struct {
			URL        string
			Collection string
			VectorSize int
		}{
			URL:        "http://localhost:6333",
			Collection: "gopher_sentinel",
			VectorSize: 768,
		},
		RAG: struct {
			ChunkSize        int
			ChunkOverlap     int
			TopK             int
			MaxContextTokens int
		}{
			ChunkSize:        512,
			ChunkOverlap:     50,
			TopK:             5,
			MaxContextTokens: 4096,
		},
		CLI: struct {
			Stream       bool
			HistoryLimit int
		}{
			Stream:       true,
			HistoryLimit: 50,
		},
	}

	// Try to read config file
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("configs")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("../configs")

	if err := viper.ReadInConfig(); err == nil {
		if url := viper.GetString("ollama.base_url"); url != "" {
			config.Ollama.BaseURL = url
		}
		if model := viper.GetString("ollama.model"); model != "" {
			config.Ollama.Model = model
		}
		if embModel := viper.GetString("ollama.embedding_model"); embModel != "" {
			config.Ollama.EmbeddingModel = embModel
		}
		if timeout := viper.GetInt("ollama.timeout"); timeout > 0 {
			config.Ollama.Timeout = timeout
		}

		if qdrantURL := viper.GetString("qdrant.url"); qdrantURL != "" {
			config.Qdrant.URL = qdrantURL
		}
		if collection := viper.GetString("qdrant.collection"); collection != "" {
			config.Qdrant.Collection = collection
		}
		if vectorSize := viper.GetInt("qdrant.vector_size"); vectorSize > 0 {
			config.Qdrant.VectorSize = vectorSize
		}

		if chunkSize := viper.GetInt("rag.chunk_size"); chunkSize > 0 {
			config.RAG.ChunkSize = chunkSize
		}
		if overlap := viper.GetInt("rag.chunk_overlap"); overlap >= 0 {
			config.RAG.ChunkOverlap = overlap
		}
		if topK := viper.GetInt("rag.top_k"); topK > 0 {
			config.RAG.TopK = topK
		}
		if maxTokens := viper.GetInt("rag.max_context_tokens"); maxTokens > 0 {
			config.RAG.MaxContextTokens = maxTokens
		}

		if stream := viper.GetBool("cli.stream"); !stream {
			config.CLI.Stream = stream
		}
		if historyLimit := viper.GetInt("cli.history_limit"); historyLimit > 0 {
			config.CLI.HistoryLimit = historyLimit
		}
	}

	// Also check environment variables
	if ollamaURL := getEnv("OLLAMA_BASE_URL", ""); ollamaURL != "" {
		config.Ollama.BaseURL = ollamaURL
	}
	if model := getEnv("OLLAMA_MODEL", ""); model != "" {
		config.Ollama.Model = model
	}
	if qdrantURL := getEnv("QDRANT_URL", ""); qdrantURL != "" {
		config.Qdrant.URL = qdrantURL
	}

	return config
}

func getEnv(key, defaultValue string) string {
	if value := viper.GetString(key); value != "" {
		return value
	}
	return defaultValue
}

// TimeDuration converts seconds to time.Duration
func (c *Config) GetTimeout() time.Duration {
	return time.Duration(c.Ollama.Timeout) * time.Second
}
