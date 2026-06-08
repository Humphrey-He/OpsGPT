package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Print the version and build information of GopherSentinel",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("GopherSentinel %s\n", Version)
		fmt.Printf("Build Date: %s\n", BuildDate)
		fmt.Println()
		fmt.Println("Tech Stack:")
		fmt.Println("  - Language: Go 1.21+")
		fmt.Println("  - LLM: Ollama / OpenAI")
		fmt.Println("  - Vector DB: Qdrant")
		fmt.Println("  - Cache: Redis")
	},
}
