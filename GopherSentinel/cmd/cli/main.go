package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Version is the application version
	Version = "0.1.0"
	// BuildDate is the build date
	BuildDate = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "GopherSentinel",
	Short: "GopherSentinel - AI-powered DevOps Assistant",
	Long: `GopherSentinel is an AI-powered DevOps assistant that combines:
- RAG-based knowledge retrieval from project documentation
- Multi-agent system for complex problem diagnosis
- Tool integration for monitoring, logging, and operations

Use 'GopherSentinel <command> --help' for more information on each command.`,
	Version: fmt.Sprintf("%s (Built: %s)", Version, BuildDate),
}

func init() {
	// Bind viper to flags
	rootCmd.PersistentFlags().StringP("config", "c", "configs/config.yaml", "config file path")
	rootCmd.PersistentFlags().Bool("verbose", false, "enable verbose logging")

	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	// Add commands
	rootCmd.AddCommand(ingestCmd)
	rootCmd.AddCommand(askCmd)
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
