package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/faq"
	"github.com/Soypete/twitch-llm-bot/logging"
)

func main() {
	var logLevel string
	var configPath string

	// Define subcommands
	syncCmd := flag.NewFlagSet("sync", flag.ExitOnError)
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	testCmd := flag.NewFlagSet("test", flag.ExitOnError)

	// Global flags
	flag.StringVar(&logLevel, "logLevel", "info", "Log level (debug, info, warn, error)")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Check for help first
	if os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help" {
		printUsage()
		os.Exit(0)
	}

	// Setup required environment variables for langchain-go
	_ = os.Setenv("OPENAI_API_KEY", "test")
	llmPath := os.Getenv("LLAMA_CPP_PATH")
	if llmPath == "" {
		fmt.Fprintln(os.Stderr, "Error: LLAMA_CPP_PATH environment variable is required")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "sync":
		syncCmd.StringVar(&configPath, "config", "configs/faq/entries.yaml", "Path to FAQ config file")
		syncCmd.StringVar(&logLevel, "logLevel", "info", "Log level")
		_ = syncCmd.Parse(os.Args[2:])
		runSync(configPath, llmPath, logLevel)

	case "list":
		listCmd.StringVar(&logLevel, "logLevel", "info", "Log level")
		_ = listCmd.Parse(os.Args[2:])
		runList(logLevel)

	case "test":
		var threshold float64
		testCmd.Float64Var(&threshold, "threshold", 0.75, "Similarity threshold")
		testCmd.StringVar(&logLevel, "logLevel", "info", "Log level")
		_ = testCmd.Parse(os.Args[2:])
		if testCmd.NArg() < 1 {
			fmt.Fprintln(os.Stderr, "Error: test command requires a message argument")
			fmt.Fprintln(os.Stderr, "Usage: faq test [options] \"your message here\"")
			os.Exit(1)
		}
		runTest(testCmd.Arg(0), threshold, llmPath, logLevel)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`FAQ Management CLI

Usage:
  faq <command> [options]

Commands:
  sync    Synchronize FAQ entries from config file to database
  list    List all FAQ entries in the database
  test    Test semantic matching for a message

Global Environment Variables:
  LLAMA_CPP_PATH    Base URL for the LLM/embedding API (required)
  POSTGRES_URL      PostgreSQL connection string (required)

Examples:
  # Sync FAQ entries from config to database
  faq sync --config configs/faq/entries.yaml

  # List current FAQ entries
  faq list

  # Test similarity match for a message
  faq test --threshold 0.75 "where can I watch your videos"`)
}

func runSync(configPath, llmPath, logLevel string) {
	logger := logging.NewLogger(logging.LogLevel(logLevel), os.Stdout)
	ctx := context.Background()

	logger.Info("starting FAQ sync", "configPath", configPath)

	// Load config
	config, err := faq.LoadConfig(configPath)
	if err != nil {
		logger.Error("failed to load FAQ config", "error", err.Error())
		os.Exit(1)
	}

	logger.Info("loaded FAQ config",
		"entries", len(config.Entries),
		"embeddingModel", config.EmbeddingModel,
		"threshold", config.SimilarityThreshold,
	)

	// Connect to database
	db, err := database.NewPostgres(logger)
	if err != nil {
		logger.Error("failed to connect to database", "error", err.Error())
		os.Exit(1)
	}
	defer db.Close()

	// Create embedding service
	embeddingService, err := faq.NewEmbeddingService(llmPath, config.EmbeddingModel)
	if err != nil {
		logger.Error("failed to create embedding service", "error", err.Error())
		os.Exit(1)
	}

	// Create syncer and run sync
	syncer := faq.NewSyncer(db.DB(), embeddingService, logger)
	result, err := syncer.SyncFromConfig(ctx, config)
	if err != nil {
		logger.Error("FAQ sync failed", "error", err.Error())
		os.Exit(1)
	}

	// Print results
	fmt.Println("\n=== FAQ Sync Results ===")
	fmt.Printf("Entries processed: %d\n", result.EntriesProcessed)
	fmt.Printf("Entries created:   %d\n", result.EntriesCreated)
	fmt.Printf("Entries updated:   %d\n", result.EntriesUpdated)
	fmt.Printf("Entries deleted:   %d\n", result.EntriesDeleted)
	fmt.Printf("Errors:            %d\n", len(result.Errors))
	fmt.Printf("Duration:          %v\n", result.Duration)

	if len(result.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, e := range result.Errors {
			fmt.Printf("  - %v\n", e)
		}
		os.Exit(1)
	}

	fmt.Println("\nFAQ sync completed successfully!")
}

func runList(logLevel string) {
	logger := logging.NewLogger(logging.LogLevel(logLevel), os.Stdout)
	ctx := context.Background()

	// Connect to database
	db, err := database.NewPostgres(logger)
	if err != nil {
		logger.Error("failed to connect to database", "error", err.Error())
		os.Exit(1)
	}
	defer db.Close()

	// Create syncer to use ListEntries
	syncer := faq.NewSyncer(db.DB(), nil, logger)
	entries, err := syncer.ListEntries(ctx)
	if err != nil {
		logger.Error("failed to list FAQ entries", "error", err.Error())
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Println("No FAQ entries found. Run 'faq sync' to populate from config.")
		return
	}

	fmt.Printf("\n=== FAQ Entries (%d total) ===\n\n", len(entries))

	currentCategory := ""
	for _, e := range entries {
		category := "uncategorized"
		if e.Category != nil {
			category = *e.Category
		}

		if category != currentCategory {
			currentCategory = category
			fmt.Printf("\n[%s]\n", currentCategory)
			fmt.Println("-------------------------------------------")
		}

		activeStr := "active"
		if !e.IsActive {
			activeStr = "inactive"
		}

		fmt.Printf("ID: %s (%s)\n", e.ID, activeStr)
		fmt.Printf("Q:  %s\n", e.Question)
		fmt.Printf("A:  %s\n", truncateString(e.Response, 80))
		fmt.Printf("Cooldown: %ds", e.CooldownSeconds)
		if e.LastTriggeredAt != nil {
			fmt.Printf(" | Last triggered: %s", e.LastTriggeredAt.Format("2006-01-02 15:04:05"))
		}
		fmt.Println()
	}
}

func runTest(message string, threshold float64, llmPath, logLevel string) {
	logger := logging.NewLogger(logging.LogLevel(logLevel), os.Stdout)
	ctx := context.Background()

	logger.Info("testing FAQ match", "message", message, "threshold", threshold)

	// Connect to database
	db, err := database.NewPostgres(logger)
	if err != nil {
		logger.Error("failed to connect to database", "error", err.Error())
		os.Exit(1)
	}
	defer db.Close()

	// Create embedding service (use default model)
	embeddingService, err := faq.NewEmbeddingService(llmPath, "text-embedding-3-small")
	if err != nil {
		logger.Error("failed to create embedding service", "error", err.Error())
		os.Exit(1)
	}

	// Create syncer to test matching
	syncer := faq.NewSyncer(db.DB(), embeddingService, logger)
	match, err := syncer.TestMatch(ctx, message, threshold)
	if err != nil {
		logger.Error("failed to test FAQ match", "error", err.Error())
		os.Exit(1)
	}

	if match == nil {
		fmt.Printf("\n=== No Match Found ===\n")
		fmt.Printf("Message:   \"%s\"\n", message)
		fmt.Printf("Threshold: %.2f\n", threshold)
		fmt.Println("\nTry lowering the threshold or adding more FAQ entries.")
		return
	}

	fmt.Printf("\n=== Match Found! ===\n")
	fmt.Printf("Message:    \"%s\"\n", message)
	fmt.Printf("Threshold:  %.2f\n", threshold)
	fmt.Printf("Similarity: %.4f (%.1f%%)\n", match.SimilarityScore, match.SimilarityScore*100)
	fmt.Println("-------------------------------------------")
	fmt.Printf("FAQ ID:     %s\n", match.ID)
	fmt.Printf("Question:   %s\n", match.Question)
	fmt.Printf("Response:   %s\n", match.Response)
	if match.Category.Valid {
		fmt.Printf("Category:   %s\n", match.Category.String)
	}
	fmt.Printf("Cooldown:   %ds\n", match.CooldownSeconds)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
