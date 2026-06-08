package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"GopherSentinel/internal/memory"
	"GopherSentinel/internal/rag"
	"GopherSentinel/pkg/llm"
	"GopherSentinel/pkg/vector"
)

var askCmd = &cobra.Command{
	Use:   "ask [question]",
	Short: "Ask questions about your documents",
	Long: `Ask questions about your indexed documents using RAG.

The answer will be generated based on the relevant context
retrieved from the vector database.

Example:
  GopherSentinel ask "What is the architecture?"
  GopherSentinel ask -i  # Interactive mode`,
	Args: cobra.RangeArgs(0, 1),
	RunE: runAsk,
}

var askFlags = struct {
	interactive bool
	stream      bool
	topK        int
	session     string
	clear       bool
	mockMode    bool
}{}

func init() {
	askCmd.Flags().BoolVarP(&askFlags.interactive, "interactive", "i", false, "interactive mode")
	askCmd.Flags().BoolVar(&askFlags.stream, "stream", true, "enable streaming responses")
	askCmd.Flags().IntVar(&askFlags.topK, "top-k", 5, "number of documents to retrieve")
	askCmd.Flags().StringVar(&askFlags.session, "session", "", "session ID for conversation history")
	askCmd.Flags().BoolVar(&askFlags.clear, "clear", false, "clear session history")
	askCmd.Flags().BoolVar(&askFlags.mockMode, "mock", false, "use mock mode (for testing without external services)")
}

func runAsk(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C gracefully
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			if scanner.Text() == "exit" || scanner.Text() == "quit" || scanner.Text() == "q" {
				cancel()
				return
			}
		}
	}()

	// Initialize configuration
	config := initConfig()

	// Initialize LLM client
	llmClient := llm.NewOllamaClient(llm.OllamaConfig{
		BaseURL:        config.Ollama.BaseURL,
		Model:         config.Ollama.Model,
		EmbeddingModel: config.Ollama.EmbeddingModel,
		Timeout:       time.Duration(config.Ollama.Timeout) * time.Second,
	})

	// Check if Ollama is available
	if !llmClient.IsAvailable(ctx) {
		fmt.Printf("⚠️  Warning: Ollama is not available at %s\n", config.Ollama.BaseURL)
		fmt.Println("   Please ensure Ollama is running.")
	}

	// Initialize vector store
	var vectorStore vector.VectorStore
	var err error

	if askFlags.mockMode {
		// Use mock vector store for testing
		fmt.Println("🔧 Using mock vector store (no external services required)")
		vectorStore = vector.NewMockVectorStore()
	} else {
		// Use real Qdrant client
		vectorStore, err = vector.NewQdrantClient(vector.QdrantConfig{
			URL:        config.Qdrant.URL,
			Collection: config.Qdrant.Collection,
			VectorSize: config.Qdrant.VectorSize,
		})
		if err != nil {
			return fmt.Errorf("failed to initialize vector store: %w", err)
		}
	}

	// Initialize RAG chain
	ragConfig := rag.RAGConfig{
		ChunkSize:      config.RAG.ChunkSize,
		ChunkOverlap:   config.RAG.ChunkOverlap,
		TopK:           askFlags.topK,
		ScoreThreshold: 0.7,
		MaxContextLen:  config.RAG.MaxContextTokens,
	}
	ragChain := rag.NewRAGChain(llmClient, vectorStore, ragConfig)

	// Initialize conversation history
	history := memory.NewConversationHistory(config.CLI.HistoryLimit)

	// Handle single question mode
	if len(args) > 0 {
		question := args[0]
		return answerQuestion(ctx, ragChain, history, question, askFlags.stream)
	}

	// Interactive mode
	if askFlags.interactive || !stdinHasInput() {
		return interactiveMode(ctx, ragChain, llmClient, history)
	}

	return nil
}

// interactiveMode runs the interactive question loop
func interactiveMode(ctx context.Context, ragChain *rag.RAGChain, llmClient *llm.OllamaClient, history *memory.ConversationHistory) error {
	fmt.Println("🤖 GopherSentinel Interactive Mode")
	fmt.Println("   Type 'exit' or 'quit' to exit")
	fmt.Println("   Type 'clear' to clear history")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("❓ You: ")
		if !scanner.Scan() {
			break
		}

		question := strings.TrimSpace(scanner.Text())
		if question == "" {
			continue
		}

		if question == "exit" || question == "quit" || question == "q" {
			fmt.Println("\n👋 Goodbye!")
			break
		}

		if question == "clear" {
			history.Clear()
			fmt.Println("🗑️  History cleared")
			continue
		}

		// Add to history
		history.AddUserMessage(question)

		// Get answer
		fmt.Print("\n🤖 Assistant: ")
		err := answerQuestionStream(ctx, ragChain, question, askFlags.topK)
		if err != nil {
			fmt.Printf("\n⚠️  Error: %v\n", err)
			continue
		}

		// Add response to history (simplified - will be updated with actual response)
		history.AddAssistantMessage("")

		fmt.Println()
	}

	return nil
}

// answerQuestion answers a single question
func answerQuestion(ctx context.Context, ragChain *rag.RAGChain, history *memory.ConversationHistory, question string, stream bool) error {
	if stream {
		return answerQuestionStream(ctx, ragChain, question, askFlags.topK)
	}

	query := rag.Query{
		Question: question,
		TopK:     askFlags.topK,
		Stream:   false,
	}

	answer, err := ragChain.Ask(ctx, query)
	if err != nil {
		return err
	}

	fmt.Printf("\n📚 Answer:\n%s\n\n", answer.Content)

	if len(answer.Sources) > 0 {
		fmt.Println("📖 Sources:")
		for i, src := range answer.Sources {
			fmt.Printf("   [%d] %s (score: %.2f)\n", i+1, src.Source, src.Score)
		}
	}

	return nil
}

// answerQuestionStream answers a question with streaming output
func answerQuestionStream(ctx context.Context, ragChain *rag.RAGChain, question string, topK int) error {
	query := rag.Query{
		Question: question,
		TopK:     topK,
		Stream:   true,
	}

	streamChan, err := ragChain.AskStream(ctx, query)
	if err != nil {
		return err
	}

	// Process stream
	for resp := range streamChan {
		switch resp.Type {
		case "sources":
			if len(resp.Sources) > 0 {
				fmt.Println("\n📖 Sources:")
				for i, src := range resp.Sources {
					title := src.Title
					if title == "" {
						title = "Unknown"
					}
					fmt.Printf("   [%d] %s (relevance: %.2f)\n", i+1, title, src.Score)
				}
			}
		case "content":
			fmt.Print(resp.Content)
		case "error":
			return fmt.Errorf("stream error: %s", resp.Error)
		case "done":
			return nil
		}
	}

	return nil
}

// stdinHasInput checks if stdin has input available
func stdinHasInput() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode()&os.ModeCharDevice) == 0
}

// CLIConfig holds CLI-specific configuration
type CLIConfig struct {
	Stream       bool
	HistoryLimit int
}
