package main

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"
	"github.com/zombor/hsa-tracker/internal/receipt"
	"github.com/zombor/hsa-tracker/internal/scanning"
)

//go:embed VERSION.txt
var versionFile string

var version = strings.TrimSpace(versionFile)

func main() {
	// Check for version flag before parsing other flags
	for _, arg := range os.Args[1:] {
		if arg == "--version" || arg == "-version" || arg == "-v" {
			fmt.Println(version)
			os.Exit(0)
		}
	}

	fs := ff.NewFlagSet("hsa-tracker")
	var (
		port        = fs.IntLong("port", 8080, "HTTP server port")
		dbPath      = fs.StringLong("db", "hsa-tracker.db", "Database file path")
		storagePath = fs.StringLong("storage", "./receipts", "Storage directory path")
		scannerType = fs.StringLong("scanner", "gemini", "Scanner type: 'gemini' or 'ollama'")
		geminiKey   = fs.StringLong("gemini-key", "", "Google Gemini API key (or set GEMINI_API_KEY env var)")
		geminiModel = fs.StringLong("gemini-model", "gemini-2.5-pro", "Google Gemini model name")
		ollamaURL   = fs.StringLong("ollama-url", "http://localhost:11434", "Ollama API base URL")
		ollamaModel = fs.StringLong("ollama-model", "llava", "Ollama model name (e.g., llava, llava-phi3, bakllava, qwen2-vl)")
		authUser    = fs.StringLong("auth-user", "", "Basic auth username (optional)")
		authPass    = fs.StringLong("auth-pass", "", "Basic auth password (optional)")
		showVersion = fs.BoolLong("version", "Show version information")
	)

	if err := ff.Parse(fs, os.Args[1:],
		ff.WithEnvVarPrefix("HSA_TRACKER"),
	); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", ffhelp.Flags(fs))
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Check version flag after parsing
	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	// Initialize database
	slog.Info("Initializing database...")
	db, err := receipt.NewBoltDB(*dbPath)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize scanner based on type
	var scanner scanning.Scanner
	switch *scannerType {
	case "gemini":
		// Get Gemini API key from flag or environment
		apiKey := *geminiKey
		if apiKey == "" {
			apiKey = os.Getenv("GEMINI_API_KEY")
		}
		if apiKey == "" {
			slog.Error("Gemini API key is required. Set --gemini-key flag or GEMINI_API_KEY environment variable")
			os.Exit(1)
		}
		slog.Info("Initializing Gemini scanner...", "model", *geminiModel)
		scanner, err = scanning.NewGemini(apiKey, *geminiModel)
		if err != nil {
			slog.Error("Failed to initialize Gemini", "error", err)
			os.Exit(1)
		}
	case "ollama":
		slog.Info("Initializing Ollama scanner...", "url", *ollamaURL, "model", *ollamaModel)
		scanner, err = scanning.NewOllama(*ollamaURL, *ollamaModel)
		if err != nil {
			slog.Error("Failed to initialize Ollama", "error", err)
			os.Exit(1)
		}
	default:
		slog.Error("Invalid scanner type", "type", *scannerType, "valid", "gemini or ollama")
		os.Exit(1)
	}
	defer scanner.Close()

	// Initialize storage
	slog.Info("Initializing storage...")
	store, err := receipt.NewLocalStorage(*storagePath)
	if err != nil {
		slog.Error("Failed to initialize storage", "error", err)
		os.Exit(1)
	}

	// Initialize service
	receiptService := receipt.NewService(db, scanner, store)

	// Initialize server
	basicAuth := receipt.BasicAuth{
		Username: *authUser,
		Password: *authPass,
	}
	server := receipt.NewServer(receiptService, basicAuth)

	// Start server in goroutine
	addr := fmt.Sprintf(":%d", *port)
	go func() {
		if err := server.Start(addr); err != nil {
			slog.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("Server started", "address", fmt.Sprintf("http://localhost%s", addr))
	if *authUser != "" || *authPass != "" {
		slog.Info("Basic auth enabled", "user", *authUser)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	slog.Info("Shutting down...")
}
