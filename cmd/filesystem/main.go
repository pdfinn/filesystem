package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"filesystem/internal/server"
	"filesystem/pkg/config"
	"filesystem/pkg/security"
)

const (
	// exitCodeSuccess indicates successful termination
	exitCodeSuccess = 0

	// exitCodeError indicates error termination
	exitCodeError = 1
)

// main initializes and runs the secure filesystem MCP server
func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "path to configuration file (optional)")
	flag.Parse()

	// Get allowed directories from command line arguments (compatible with TS version)
	args := flag.Args()

	var cfg *config.Config
	var err error

	if configPath != "" {
		// Load configuration from file
		cfg, err = config.Load(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
			os.Exit(exitCodeError)
		}
	} else if len(args) > 0 {
		// Create configuration from command line arguments (TypeScript compatibility)
		cfg = config.Default()
		cfg.AllowedDirectories = args

		// Validate and normalize directories
		if err := validateCommandLineDirectories(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Invalid directory arguments: %v\n", err)
			os.Exit(exitCodeError)
		}
	} else {
		// Show usage information
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <allowed-directory> [additional-directories...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "   or: %s -config <config-file>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s /home/user/documents /home/user/projects\n", os.Args[0])
		os.Exit(exitCodeError)
	}

	// Initialize structured logger per custom instructions
	logger := initializeLogger(cfg.LogLevel)
	logger.Info("Starting secure filesystem MCP server",
		"version", cfg.Server.Version,
		"config_source", getConfigSource(configPath, args),
		"allowed_directories", cfg.AllowedDirectories)

	// Create server instance with dependency injection
	srv, err := server.New(cfg, logger)
	if err != nil {
		logger.Error("Failed to create server", "error", err)
		os.Exit(exitCodeError)
	}

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start(ctx)
	}()

	// Log startup complete to stderr (compatible with TS version)
	fmt.Fprintf(os.Stderr, "Secure MCP Filesystem Server running on stdio\n")
	fmt.Fprintf(os.Stderr, "Allowed directories: %v\n", cfg.AllowedDirectories)

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", "signal", sig)
		cancel()
	case err := <-errChan:
		if err != nil {
			logger.Error("Server error", "error", err)
			cancel()
		}
	}

	// Graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error", "error", err)
		os.Exit(exitCodeError)
	}

	logger.Info("Server shutdown complete")
	os.Exit(exitCodeSuccess)
}

// validateCommandLineDirectories validates directories provided via command line
func validateCommandLineDirectories(cfg *config.Config) error {
	// Input validation per Rule 7
	if cfg == nil {
		return fmt.Errorf("configuration is required")
	}
	if len(cfg.AllowedDirectories) == 0 {
		return fmt.Errorf("at least one directory must be specified")
	}

	// Validate each directory
	for i, dir := range cfg.AllowedDirectories {

		// Expand home directory if needed
		dir = security.ExpandHomePath(dir)

		// Convert to absolute path
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for %s: %w", dir, err)
		}

		cfg.AllowedDirectories[i] = absDir
		dir = absDir

		// Check if directory exists and is accessible
		info, err := os.Stat(dir)
		if err != nil {
			return fmt.Errorf("directory %s is not accessible: %w", dir, err)
		}

		if !info.IsDir() {
			return fmt.Errorf("path %s is not a directory", dir)
		}
	}

	return nil
}

// getConfigSource returns a string indicating how configuration was loaded
func getConfigSource(configPath string, args []string) string {
	if configPath != "" {
		return "config_file"
	}
	if len(args) > 0 {
		return "command_line"
	}
	return "default"
}

// initializeLogger creates a structured logger with the specified level
// per custom instructions for slog usage
func initializeLogger(level string) *slog.Logger {
	var logLevel slog.Level

	// Parse log level with validation per Rule 7
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		// Default to info level for invalid inputs
		logLevel = slog.LevelInfo
	}

	// Create handler with structured logging options
	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	handler := slog.NewJSONHandler(os.Stderr, opts)
	return slog.New(handler)
}
