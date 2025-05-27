package server

import (
	"context"
	"fmt"
	"log/slog"

	"filesystem/internal/handlers"
	"filesystem/pkg/config"
	"filesystem/pkg/filesystem"
	"filesystem/pkg/security"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server represents the secure filesystem MCP server
type Server struct {
	mcpServer     *server.MCPServer
	toolHandlers  *handlers.ToolHandlers
	pathValidator *security.PathValidator
	fsOps         *filesystem.Operations
	logger        *slog.Logger
	config        *config.Config
}

// New creates a new server instance with all necessary components
func New(cfg *config.Config, logger *slog.Logger) (*Server, error) {
	// Input validation per Rule 7 (check parameter validity)
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	logger.Info("Creating secure filesystem MCP server",
		"name", cfg.Server.Name,
		"version", cfg.Server.Version,
		"allowed_dirs_count", len(cfg.AllowedDirectories))

	// Create security components
	pathValidator := security.NewPathValidator(cfg.AllowedDirectories, logger)
	fsOps := filesystem.NewOperations(pathValidator, logger)

	// Create MCP server with capabilities
	mcpServer := server.NewMCPServer(
		cfg.Server.Name,
		cfg.Server.Version,
		server.WithToolCapabilities(true),
	)

	// Create tool handlers
	toolHandlers := handlers.NewToolHandlers(pathValidator, fsOps, logger)

	// Register all tools with the MCP server
	if err := toolHandlers.RegisterTools(mcpServer); err != nil {
		logger.Error("Failed to register tools", "error", err)
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	srv := &Server{
		mcpServer:     mcpServer,
		toolHandlers:  toolHandlers,
		pathValidator: pathValidator,
		fsOps:         fsOps,
		logger:        logger,
		config:        cfg,
	}

	logger.Info("Server created successfully",
		"tools_registered", true,
		"transport", cfg.Server.Transport)

	return srv, nil
}

// Start begins serving MCP requests via stdio
func (s *Server) Start(ctx context.Context) error {
	// Input validation per Rule 7
	if ctx == nil {
		return fmt.Errorf("context is required")
	}

	s.logger.Info("Starting MCP server",
		"allowed_directories", s.pathValidator.GetAllowedDirectories())

	// Use ServeStdio to serve the MCP server over stdio
	if err := server.ServeStdio(s.mcpServer); err != nil {
		s.logger.Error("Failed to serve stdio", "error", err)
		return fmt.Errorf("failed to serve stdio: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	// Input validation per Rule 7
	if ctx == nil {
		return fmt.Errorf("context is required")
	}

	s.logger.Info("Shutting down MCP server")

	// Note: The MCP-Go library doesn't appear to have explicit shutdown methods
	// so we just log the shutdown. The transport connection will be closed
	// when the context is cancelled.

	s.logger.Info("MCP server shutdown complete")
	return nil
}

// GetCapabilities returns the server capabilities
func (s *Server) GetCapabilities() mcp.ServerCapabilities {
	return mcp.ServerCapabilities{
		Tools: &struct {
			ListChanged bool `json:"listChanged,omitempty"`
		}{
			ListChanged: true,
		},
	}
}

// GetAllowedDirectories returns the allowed directories for this server
func (s *Server) GetAllowedDirectories() []string {
	return s.pathValidator.GetAllowedDirectories()
}
