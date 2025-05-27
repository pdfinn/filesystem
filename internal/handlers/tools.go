package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"filesystem/pkg/filesystem"
	"filesystem/pkg/security"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolHandlers provides MCP tool implementations for filesystem operations
type ToolHandlers struct {
	pathValidator *security.PathValidator
	fsOps         *filesystem.Operations
	logger        *slog.Logger
}

// NewToolHandlers creates a new tool handlers instance
func NewToolHandlers(pathValidator *security.PathValidator, fsOps *filesystem.Operations, logger *slog.Logger) *ToolHandlers {
	return &ToolHandlers{
		pathValidator: pathValidator,
		fsOps:         fsOps,
		logger:        logger,
	}
}

// RegisterTools registers all filesystem tools with the MCP server
func (th *ToolHandlers) RegisterTools(srv *server.MCPServer) error {
	// Define all tools with proper schema validation per Rule 5
	tools := []struct {
		tool    mcp.Tool
		handler server.ToolHandlerFunc
	}{
		{th.createReadFileTool(), th.handleReadFile},
		{th.createReadMultipleFilesTool(), th.handleReadMultipleFiles},
		{th.createWriteFileTool(), th.handleWriteFile},
		{th.createEditFileTool(), th.handleEditFile},
		{th.createCreateDirectoryTool(), th.handleCreateDirectory},
		{th.createListDirectoryTool(), th.handleListDirectory},
		{th.createDirectoryTreeTool(), th.handleDirectoryTree},
		{th.createMoveFileTool(), th.handleMoveFile},
		{th.createSearchFilesTool(), th.handleSearchFiles},
		{th.createGetFileInfoTool(), th.handleGetFileInfo},
		{th.createListAllowedDirectoriesTool(), th.handleListAllowedDirectories},
	}

	// Register each tool with fixed upper bound per Rule 2
	for i := 0; i < len(tools) && i < 20; i++ {
		tool := tools[i]
		srv.AddTool(tool.tool, tool.handler)
		th.logger.Debug("Tool registered successfully", "tool", tool.tool.Name)
	}

	th.logger.Info("All filesystem tools registered successfully", "count", len(tools))
	return nil
}

// Tool creation methods

func (th *ToolHandlers) createReadFileTool() mcp.Tool {
	return mcp.NewTool("read_file",
		mcp.WithDescription("Read the complete contents of a file from the file system. "+
			"Handles various text encodings and provides detailed error messages "+
			"if the file cannot be read. Use this tool when you need to examine "+
			"the contents of a single file. Only works within allowed directories."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to the file to read")))
}

func (th *ToolHandlers) createReadMultipleFilesTool() mcp.Tool {
	return mcp.NewTool("read_multiple_files",
		mcp.WithDescription("Read the contents of multiple files simultaneously. This is more "+
			"efficient than reading files one by one when you need to analyze "+
			"or compare multiple files. Each file's content is returned with its "+
			"path as a reference. Failed reads for individual files won't stop "+
			"the entire operation. Only works within allowed directories."),
		mcp.WithArray("paths", mcp.Required(), mcp.Description("Array of file paths to read"),
			mcp.Items(map[string]interface{}{"type": "string"})))
}

func (th *ToolHandlers) createWriteFileTool() mcp.Tool {
	return mcp.NewTool("write_file",
		mcp.WithDescription("Create a new file or completely overwrite an existing file with new content. "+
			"Use with caution as it will overwrite existing files without warning. "+
			"Handles text content with proper encoding. Only works within allowed directories."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to the file to write")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Content to write to the file")))
}

func (th *ToolHandlers) createEditFileTool() mcp.Tool {
	return mcp.NewTool("edit_file",
		mcp.WithDescription("Make line-based edits to a text file. Each edit replaces exact line sequences "+
			"with new content. Returns a git-style diff showing the changes made. "+
			"Only works within allowed directories."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to the file to edit")),
		mcp.WithArray("edits", mcp.Required(), mcp.Description("Array of edit operations to apply"),
			mcp.Items(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"oldText": map[string]interface{}{
						"type":        "string",
						"description": "Text to search for - must match exactly",
					},
					"newText": map[string]interface{}{
						"type":        "string",
						"description": "Text to replace with",
					},
				},
				"required": []string{"oldText", "newText"},
			})),
		mcp.WithBoolean("dryRun", mcp.Description("Preview changes using git-style diff format"), mcp.DefaultBool(false)))
}

func (th *ToolHandlers) createCreateDirectoryTool() mcp.Tool {
	return mcp.NewTool("create_directory",
		mcp.WithDescription("Create a new directory or ensure a directory exists. Can create multiple "+
			"nested directories in one operation. If the directory already exists, "+
			"this operation will succeed silently. Perfect for setting up directory "+
			"structures for projects or ensuring required paths exist. Only works within allowed directories."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to the directory to create")))
}

func (th *ToolHandlers) createListDirectoryTool() mcp.Tool {
	return mcp.NewTool("list_directory",
		mcp.WithDescription("Get a detailed listing of all files and directories in a specified path. "+
			"Results clearly distinguish between files and directories with [FILE] and [DIR] "+
			"prefixes. This tool is essential for understanding directory structure and "+
			"finding specific files within a directory. Only works within allowed directories."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to the directory to list")))
}

func (th *ToolHandlers) createDirectoryTreeTool() mcp.Tool {
	return mcp.NewTool("directory_tree",
		mcp.WithDescription("Get a recursive tree view of files and directories as a JSON structure. "+
			"Each entry includes 'name', 'type' (file/directory), and 'children' for directories. "+
			"Files have no children array, while directories always have a children array (which may be empty). "+
			"The output is formatted with 2-space indentation for readability. Only works within allowed directories."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to the directory to analyze")))
}

func (th *ToolHandlers) createMoveFileTool() mcp.Tool {
	return mcp.NewTool("move_file",
		mcp.WithDescription("Move or rename files and directories. Can move files between directories "+
			"and rename them in a single operation. If the destination exists, the "+
			"operation will fail. Works across different directories and can be used "+
			"for simple renaming within the same directory. Both source and destination must be within allowed directories."),
		mcp.WithString("source", mcp.Required(), mcp.Description("Source path to move from")),
		mcp.WithString("destination", mcp.Required(), mcp.Description("Destination path to move to")))
}

func (th *ToolHandlers) createSearchFilesTool() mcp.Tool {
	return mcp.NewTool("search_files",
		mcp.WithDescription("Recursively search for files and directories matching a pattern. "+
			"Searches through all subdirectories from the starting path. The search "+
			"is case-insensitive and matches partial names. Returns full paths to all "+
			"matching items. Great for finding files when you don't know their exact location. "+
			"Only searches within allowed directories."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Root path to search from")),
		mcp.WithString("pattern", mcp.Required(), mcp.Description("Search pattern to match against filenames")),
		mcp.WithArray("excludePatterns", mcp.Description("Patterns to exclude from search results"),
			mcp.DefaultArray([]string{}), mcp.Items(map[string]interface{}{"type": "string"})))
}

func (th *ToolHandlers) createGetFileInfoTool() mcp.Tool {
	return mcp.NewTool("get_file_info",
		mcp.WithDescription("Retrieve detailed metadata about a file or directory. Returns comprehensive "+
			"information including size, creation time, last modified time, permissions, "+
			"and type. This tool is perfect for understanding file characteristics "+
			"without reading the actual content. Only works within allowed directories."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to the file or directory")))
}

func (th *ToolHandlers) createListAllowedDirectoriesTool() mcp.Tool {
	return mcp.NewTool("list_allowed_directories",
		mcp.WithDescription("Returns the list of directories that this server is allowed to access. "+
			"Use this to understand which directories are available before trying to access files."))
}

// Tool handler methods

func (th *ToolHandlers) handleReadFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, errRes := getArguments(req)
	if errRes != nil {
		return errRes, nil
	}

	path, errRes := getRequiredString(args, "path")
	if errRes != nil {
		return errRes, nil
	}

	// Validate path security
	validPath, err := th.pathValidator.ValidatePath(path)
	if err != nil {
		th.logger.Warn("Path validation failed", "path", path, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Error: %s", err.Error())), nil
	}

	// Read file content
	content, err := th.fsOps.ReadFile(validPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(content), nil
}

func (th *ToolHandlers) handleReadMultipleFiles(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, errRes := getArguments(req)
	if errRes != nil {
		return errRes, nil
	}

	pathsSlice, errRes := getRequiredStringSlice(args, "paths")
	if errRes != nil {
		return errRes, nil
	}
	// Validate each path
	paths := make([]string, 0, len(pathsSlice))
	for i := 0; i < len(pathsSlice) && i < 100; i++ {
		path := pathsSlice[i]
		validPath, err := th.pathValidator.ValidatePath(path)
		if err != nil {
			th.logger.Warn("Path validation failed", "path", path, "error", err)
			paths = append(paths, path)
		} else {
			paths = append(paths, validPath)
		}
	}

	if len(paths) == 0 {
		return mcp.NewToolResultError("No valid paths provided"), nil
	}

	// Read multiple files
	content, err := th.fsOps.ReadMultipleFiles(paths)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(content), nil
}

func (th *ToolHandlers) handleWriteFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, errRes := getArguments(req)
	if errRes != nil {
		return errRes, nil
	}

	path, errRes := getRequiredString(args, "path")
	if errRes != nil {
		return errRes, nil
	}

	content, errRes := getRequiredString(args, "content")
	if errRes != nil {
		return errRes, nil
	}

	// Validate path security
	validPath, err := th.pathValidator.ValidatePath(path)
	if err != nil {
		th.logger.Warn("Path validation failed", "path", path, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Error: %s", err.Error())), nil
	}

	// Write file
	err = th.fsOps.WriteFile(validPath, content)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully wrote to %s", path)), nil
}

func (th *ToolHandlers) handleEditFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, errRes := getArguments(req)
	if errRes != nil {
		return errRes, nil
	}

	path, errRes := getRequiredString(args, "path")
	if errRes != nil {
		return errRes, nil
	}

	edits, errRes := getEditOperations(args)
	if errRes != nil {
		return errRes, nil
	}

	dryRun := getOptionalBool(args, "dryRun", false)

	// Validate path security
	validPath, err := th.pathValidator.ValidatePath(path)
	if err != nil {
		th.logger.Warn("Path validation failed", "path", path, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Error: %s", err.Error())), nil
	}

	// Edit file
	diff, err := th.fsOps.EditFile(validPath, edits, dryRun)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(diff), nil
}

func (th *ToolHandlers) handleCreateDirectory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, errRes := getArguments(req)
	if errRes != nil {
		return errRes, nil
	}

	path, errRes := getRequiredString(args, "path")
	if errRes != nil {
		return errRes, nil
	}

	// Validate path security
	validPath, err := th.pathValidator.ValidatePath(path)
	if err != nil {
		th.logger.Warn("Path validation failed", "path", path, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Error: %s", err.Error())), nil
	}

	// Create directory
	err = th.fsOps.CreateDirectory(validPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully created directory %s", path)), nil
}

func (th *ToolHandlers) handleListDirectory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, errRes := getArguments(req)
	if errRes != nil {
		return errRes, nil
	}

	path, errRes := getRequiredString(args, "path")
	if errRes != nil {
		return errRes, nil
	}

	// Validate path security
	validPath, err := th.pathValidator.ValidatePath(path)
	if err != nil {
		th.logger.Warn("Path validation failed", "path", path, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Error: %s", err.Error())), nil
	}

	// List directory
	listing, err := th.fsOps.ListDirectory(validPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(listing), nil
}

func (th *ToolHandlers) handleDirectoryTree(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, errRes := getArguments(req)
	if errRes != nil {
		return errRes, nil
	}

	path, errRes := getRequiredString(args, "path")
	if errRes != nil {
		return errRes, nil
	}

	// Validate path security
	validPath, err := th.pathValidator.ValidatePath(path)
	if err != nil {
		th.logger.Warn("Path validation failed", "path", path, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Error: %s", err.Error())), nil
	}

	// Build directory tree
	tree, err := th.fsOps.DirectoryTree(validPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(tree), nil
}

func (th *ToolHandlers) handleMoveFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, errRes := getArguments(req)
	if errRes != nil {
		return errRes, nil
	}

	source, errRes := getRequiredString(args, "source")
	if errRes != nil {
		return errRes, nil
	}

	destination, errRes := getRequiredString(args, "destination")
	if errRes != nil {
		return errRes, nil
	}

	// Validate both paths
	validSource, err := th.pathValidator.ValidatePath(source)
	if err != nil {
		th.logger.Warn("Source path validation failed", "path", source, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Error: %s", err.Error())), nil
	}

	validDestination, err := th.pathValidator.ValidatePath(destination)
	if err != nil {
		th.logger.Warn("Destination path validation failed", "path", destination, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Error: %s", err.Error())), nil
	}

	// Move file
	err = th.fsOps.MoveFile(validSource, validDestination)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully moved %s to %s", source, destination)), nil
}

func (th *ToolHandlers) handleSearchFiles(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, errRes := getArguments(req)
	if errRes != nil {
		return errRes, nil
	}

	path, errRes := getRequiredString(args, "path")
	if errRes != nil {
		return errRes, nil
	}

	pattern, errRes := getRequiredString(args, "pattern")
	if errRes != nil {
		return errRes, nil
	}

	// Parse exclude patterns
	excludePatterns := getOptionalStringSlice(args, "excludePatterns")

	// Validate path security
	validPath, err := th.pathValidator.ValidatePath(path)
	if err != nil {
		th.logger.Warn("Path validation failed", "path", path, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Error: %s", err.Error())), nil
	}

	// Search files
	results, err := th.fsOps.SearchFiles(validPath, pattern, excludePatterns)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error: %s", err.Error())), nil
	}

	if len(results) == 0 {
		return mcp.NewToolResultText("No matches found"), nil
	}

	return mcp.NewToolResultText(strings.Join(results, "\n")), nil
}

func (th *ToolHandlers) handleGetFileInfo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, errRes := getArguments(req)
	if errRes != nil {
		return errRes, nil

	}

	path, errRes := getRequiredString(args, "path")
	if errRes != nil {
		return errRes, nil
	}

	// Validate path security
	validPath, err := th.pathValidator.ValidatePath(path)
	if err != nil {
		th.logger.Warn("Path validation failed", "path", path, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Error: %s", err.Error())), nil
	}

	// Get file info
	info, err := th.fsOps.GetFileInfo(validPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error: %s", err.Error())), nil
	}

	// Format info as JSON
	infoJSON, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("Failed to format file info"), nil
	}

	return mcp.NewToolResultText(string(infoJSON)), nil
}

func (th *ToolHandlers) handleListAllowedDirectories(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dirs := th.pathValidator.GetAllowedDirectories()
	result := fmt.Sprintf("Allowed directories:\n%s", strings.Join(dirs, "\n"))
	return mcp.NewToolResultText(result), nil
}
