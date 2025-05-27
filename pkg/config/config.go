package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type Config struct {
	// LogLevel specifies the logging level (debug, info, warn, error)
	LogLevel string `yaml:"log_level"`

	// AllowedDirectories contains the list of directories this server can access
	AllowedDirectories []string `yaml:"allowed_directories"`

	// Server configuration
	Server ServerConfig `yaml:"server"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	// Name of the MCP server
	Name string `yaml:"name"`

	// Version of the MCP server
	Version string `yaml:"version"`

	// Transport specifies the transport method (stdio, sse, etc.)
	Transport string `yaml:"transport"`
}

// Load reads and validates configuration from the specified file path
func Load(configPath string) (*Config, error) {
	// Check file bounds per Rule 7 (check return values)
	if configPath == "" {
		return nil, fmt.Errorf("configuration file path is required")
	}

	// Read configuration file with validation
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML configuration
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration per Rule 5 (assertions for anomalous conditions)
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Normalize and validate allowed directories
	if err := normalizeDirectories(&cfg); err != nil {
		return nil, fmt.Errorf("failed to process allowed directories: %w", err)
	}

	return &cfg, nil
}

// validateConfig performs configuration validation per Rule 5
func validateConfig(cfg *Config) error {
	// Validate log level
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if cfg.LogLevel == "" {
		cfg.LogLevel = "info" // Default value
	}

	if !validLevels[cfg.LogLevel] {
		return fmt.Errorf("invalid log level: %s", cfg.LogLevel)
	}

	// Validate server configuration
	if cfg.Server.Name == "" {
		cfg.Server.Name = "secure-filesystem-server" // Default value
	}

	if cfg.Server.Version == "" {
		cfg.Server.Version = "1.0.0" // Default value
	}

	if cfg.Server.Transport == "" {
		cfg.Server.Transport = "stdio" // Default value
	}

	// Validate allowed directories (at least one required)
	if len(cfg.AllowedDirectories) == 0 {
		return fmt.Errorf("at least one allowed directory must be specified")
	}

	return nil
}

// normalizeDirectories processes and validates allowed directories
func normalizeDirectories(cfg *Config) error {
	normalizedDirs := make([]string, 0, len(cfg.AllowedDirectories))

	// Process each directory
	for _, dir := range cfg.AllowedDirectories {

		// Expand home directory if needed
		if dir == "~" || len(dir) > 1 && dir[:2] == "~/" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			if dir == "~" {
				dir = homeDir
			} else {
				dir = filepath.Join(homeDir, dir[2:])
			}
		}

		// Convert to absolute path
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for %s: %w", dir, err)
		}

		// Validate directory exists and is accessible
		info, err := os.Stat(absDir)
		if err != nil {
			return fmt.Errorf("directory %s is not accessible: %w", absDir, err)
		}

		if !info.IsDir() {
			return fmt.Errorf("path %s is not a directory", absDir)
		}

		// Clean and normalize path
		normalizedDir := filepath.Clean(absDir)
		normalizedDirs = append(normalizedDirs, normalizedDir)
	}

	cfg.AllowedDirectories = normalizedDirs
	return nil
}

// Default returns a default configuration
func Default() *Config {
	return &Config{
		LogLevel:           "info",
		AllowedDirectories: []string{"."},
		Server: ServerConfig{
			Name:      "secure-filesystem-server",
			Version:   "1.0.0",
			Transport: "stdio",
		},
	}
}
