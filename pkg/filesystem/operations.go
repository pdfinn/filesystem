package filesystem

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/sergi/go-diff/diffmatchpatch"

	"filesystem/pkg/security"
)

const maxReadSize int64 = 1 * 1024 * 1024 // 1MB

// maxTreeDepth defines the maximum depth DirectoryTree will recurse
const maxTreeDepth int = 20

// FileInfo represents detailed file information
type FileInfo struct {
	Size        int64     `json:"size"`
	Created     time.Time `json:"created"`
	Modified    time.Time `json:"modified"`
	Accessed    time.Time `json:"accessed"`
	IsDirectory bool      `json:"isDirectory"`
	IsFile      bool      `json:"isFile"`
	Permissions string    `json:"permissions"`
}

// TreeEntry represents a directory tree entry
type TreeEntry struct {
	Name     string       `json:"name"`
	Type     string       `json:"type"`
	Children *[]TreeEntry `json:"children,omitempty"`
}

// EditOperation represents a file edit operation
type EditOperation struct {
	OldText string `json:"oldText"`
	NewText string `json:"newText"`
}

// Operations provides secure filesystem operations
type Operations struct {
	logger        *slog.Logger
	pathValidator *security.PathValidator
}

// NewOperations creates a new filesystem operations instance
func NewOperations(validator *security.PathValidator, logger *slog.Logger) *Operations {
	return &Operations{
		logger:        logger,
		pathValidator: validator,
	}
}

// ReadFile reads a file's content
func (ops *Operations) ReadFile(filePath string) (string, error) {
	// Input validation per Rule 7
	if filePath == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	ops.logger.Debug("Reading file", "path", filePath)

	info, err := os.Stat(filePath)
	if err != nil {
		ops.logger.Error("Failed to stat file", "path", filePath, "error", err)
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	if info.Size() > maxReadSize {
		ops.logger.Warn("File size exceeds limit", "path", filePath, "size", info.Size())
		return "", fmt.Errorf("file exceeds maximum allowed size")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		ops.logger.Error("Failed to read file", "path", filePath, "error", err)
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	ops.logger.Debug("File read successfully", "path", filePath, "size", len(data))
	return string(data), nil
}

// ReadMultipleFiles reads multiple files and returns their contents
func (ops *Operations) ReadMultipleFiles(filePaths []string) (string, error) {
	// Input validation per Rule 7
	if len(filePaths) == 0 {
		return "", fmt.Errorf("no file paths provided")
	}

	results := make([]string, 0, len(filePaths))

	// Process files
	for _, filePath := range filePaths {

		content, err := ops.ReadFile(filePath)
		if err != nil {
			// Continue processing other files even if one fails
			result := fmt.Sprintf("%s: Error - %s", filePath, err.Error())
			results = append(results, result)
			ops.logger.Warn("Failed to read file in batch", "path", filePath, "error", err)
		} else {
			result := fmt.Sprintf("%s:\n%s\n", filePath, content)
			results = append(results, result)
		}
	}

	return strings.Join(results, "\n---\n"), nil
}

// WriteFile writes content to a file
func (ops *Operations) WriteFile(filePath, content string) error {
	// Input validation per Rule 7
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	ops.logger.Debug("Writing file", "path", filePath, "size", len(content))

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		ops.logger.Error("Failed to write file", "path", filePath, "error", err)
		return fmt.Errorf("failed to write file: %w", err)
	}

	ops.logger.Info("File written successfully", "path", filePath, "size", len(content))
	return nil
}

// EditFile applies edits to a file and returns a diff
func (ops *Operations) EditFile(filePath string, edits []EditOperation, dryRun bool) (string, error) {
	// Input validation per Rule 7
	if filePath == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}
	if len(edits) == 0 {
		return "", fmt.Errorf("no edits provided")
	}

	ops.logger.Debug("Editing file", "path", filePath, "edits_count", len(edits), "dry_run", dryRun)

	// Read original content
	originalContent, err := ops.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// Apply edits
	modifiedContent, err := ops.applyEdits(originalContent, edits)
	if err != nil {
		return "", err
	}

	// Create diff
	diff := ops.createUnifiedDiff(originalContent, modifiedContent, filePath)

	// Write file if not dry run
	if !dryRun {
		err = ops.WriteFile(filePath, modifiedContent)
		if err != nil {
			return "", err
		}
		ops.logger.Info("File edits applied", "path", filePath, "edits_count", len(edits))
	} else {
		ops.logger.Debug("Dry run completed", "path", filePath)
	}

	return diff, nil
}

// applyEdits applies a series of edits to content
func (ops *Operations) applyEdits(content string, edits []EditOperation) (string, error) {
	modifiedContent := ops.normalizeLineEndings(content)

	// Apply edits sequentially
	for i, edit := range edits {
		oldText := ops.normalizeLineEndings(edit.OldText)
		newText := ops.normalizeLineEndings(edit.NewText)

		// Try exact match first
		if strings.Contains(modifiedContent, oldText) {
			modifiedContent = strings.Replace(modifiedContent, oldText, newText, 1)
			continue
		}

		// Try line-by-line matching with whitespace flexibility
		if err := ops.applyLineBasedEdit(&modifiedContent, oldText, newText); err != nil {
			return "", fmt.Errorf("could not apply edit %d: %w", i+1, err)
		}
	}

	return modifiedContent, nil
}

// applyLineBasedEdit applies an edit using line-by-line matching
func (ops *Operations) applyLineBasedEdit(content *string, oldText, newText string) error {
	oldLines := strings.Split(oldText, "\n")
	contentLines := strings.Split(*content, "\n")

	// Search for matching lines
	for i := 0; i <= len(contentLines)-len(oldLines); i++ {
		if ops.linesMatch(contentLines[i:i+len(oldLines)], oldLines) {
			// Replace matched lines
			newLines := strings.Split(newText, "\n")

			// Preserve indentation of first line
			if len(newLines) > 0 && len(contentLines) > i {
				originalIndent := ops.extractIndentation(contentLines[i])
				newLines[0] = originalIndent + strings.TrimLeft(newLines[0], " \t")
			}

			// Replace lines
			contentLines = append(contentLines[:i], append(newLines, contentLines[i+len(oldLines):]...)...)
			*content = strings.Join(contentLines, "\n")
			return nil
		}
	}

	return fmt.Errorf("could not find matching lines for edit")
}

// linesMatch checks if two line slices match with whitespace normalization
func (ops *Operations) linesMatch(contentLines, oldLines []string) bool {
	if len(contentLines) != len(oldLines) {
		return false
	}

	// Compare lines
	for i := 0; i < len(oldLines); i++ {
		if strings.TrimSpace(contentLines[i]) != strings.TrimSpace(oldLines[i]) {
			return false
		}
	}

	return true
}

// extractIndentation extracts leading whitespace from a line
func (ops *Operations) extractIndentation(line string) string {
	re := regexp.MustCompile(`^[ \t]*`)
	return re.FindString(line)
}

// normalizeLineEndings normalizes line endings to Unix style
func (ops *Operations) normalizeLineEndings(text string) string {
	return strings.ReplaceAll(text, "\r\n", "\n")
}

// createUnifiedDiff creates a unified diff between original and modified content
func (ops *Operations) createUnifiedDiff(original, modified, filename string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(original, modified, false)

	// Create patch
	patches := dmp.PatchMake(original, diffs)
	patch := dmp.PatchToText(patches)

	// Format with backticks
	numBackticks := 3
	for strings.Contains(patch, strings.Repeat("`", numBackticks)) {
		numBackticks++
		if numBackticks > 10 { // Safety bound per Rule 2
			break
		}
	}

	backticks := strings.Repeat("`", numBackticks)
	return fmt.Sprintf("%sdiff\n%s%s\n\n", backticks, patch, backticks)
}

// CreateDirectory creates a directory and all parent directories
func (ops *Operations) CreateDirectory(dirPath string) error {
	// Input validation per Rule 7
	if dirPath == "" {
		return fmt.Errorf("directory path cannot be empty")
	}

	ops.logger.Debug("Creating directory", "path", dirPath)

	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		ops.logger.Error("Failed to create directory", "path", dirPath, "error", err)
		return fmt.Errorf("failed to create directory: %w", err)
	}

	ops.logger.Info("Directory created successfully", "path", dirPath)
	return nil
}

// ListDirectory lists the contents of a directory
func (ops *Operations) ListDirectory(dirPath string) (string, error) {
	// Input validation per Rule 7
	if dirPath == "" {
		return "", fmt.Errorf("directory path cannot be empty")
	}

	ops.logger.Debug("Listing directory", "path", dirPath)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		ops.logger.Error("Failed to read directory", "path", dirPath, "error", err)
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	results := make([]string, 0, len(entries))

	// Process entries
	for _, entry := range entries {
		prefix := "[FILE]"
		if entry.IsDir() {
			prefix = "[DIR]"
		}
		results = append(results, fmt.Sprintf("%s %s", prefix, entry.Name()))
	}

	ops.logger.Debug("Directory listed successfully", "path", dirPath, "entries_count", len(results))
	return strings.Join(results, "\n"), nil
}

// DirectoryTree builds a recursive tree structure of a directory
func (ops *Operations) DirectoryTree(dirPath string) (string, error) {
	// Input validation per Rule 7
	if dirPath == "" {
		return "", fmt.Errorf("directory path cannot be empty")
	}

	// Validate the provided path
	validPath, err := ops.pathValidator.ValidatePath(dirPath)
	if err != nil {
		return "", err
	}

	ops.logger.Debug("Building directory tree", "path", validPath)

	// Validate root directory is within allowed paths
	validPath, err := ops.pathValidator.ValidatePath(dirPath)
	if err != nil {
		return "", err
	}

	// Track visited real paths to avoid infinite recursion
	visited := make(map[string]bool)

	tree, err := ops.buildTree(validPath, visited)
	if err != nil {
		return "", err
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		ops.logger.Error("Failed to marshal tree to JSON", "error", err)
		return "", fmt.Errorf("failed to create JSON tree: %w", err)
	}

	ops.logger.Debug("Directory tree built successfully", "path", dirPath)
	return string(jsonData), nil
}

// buildTree recursively builds a tree structure
func (ops *Operations) buildTree(dirPath string, visited map[string]bool, depth int) ([]TreeEntry, error) {
	if depth > maxTreeDepth {
		return nil, fmt.Errorf("maximum directory depth exceeded")
	}
	realPath, err := filepath.EvalSymlinks(dirPath)
	if err != nil {
		// If symlink resolution fails, fall back to cleaned path
		realPath = filepath.Clean(dirPath)
	}

	// Normalize to absolute path for consistent map keys
	if abs, err := filepath.Abs(realPath); err == nil {
		realPath = abs
	}

	if visited[realPath] {
		ops.logger.Debug("Skipping already visited path", "path", realPath)
		return []TreeEntry{}, nil
	}
	visited[realPath] = true

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	result := make([]TreeEntry, 0, len(entries))

	// Process entries
	for _, entry := range entries {

		treeEntry := TreeEntry{
			Name: entry.Name(),
			Type: "file",
		}

		if entry.IsDir() {
			treeEntry.Type = "directory"

			// Recursively build subtree
			subPath := filepath.Join(dirPath, entry.Name())
			validPath, err := ops.pathValidator.ValidatePath(subPath)
			if err != nil {
				ops.logger.Warn("Path validation failed", "path", subPath, "error", err)
				// Skip this directory if validation fails
				continue
			}
			children, err := ops.buildTree(validPath, visited, depth+1)
			if err != nil {
				ops.logger.Warn("Failed to build subtree", "path", subPath, "error", err)
				// Continue with empty children rather than failing
				children = []TreeEntry{}
			}
			treeEntry.Children = &children
		}

		result = append(result, treeEntry)
	}

	return result, nil
}

// MoveFile moves or renames a file or directory
func (ops *Operations) MoveFile(sourcePath, destPath string) error {
	// Input validation per Rule 7
	if sourcePath == "" {
		return fmt.Errorf("source path cannot be empty")
	}
	if destPath == "" {
		return fmt.Errorf("destination path cannot be empty")
	}

	ops.logger.Debug("Moving file", "source", sourcePath, "destination", destPath)

	// Check if destination already exists to avoid overwriting
	if _, err := os.Stat(destPath); err == nil {
		ops.logger.Warn("Destination already exists", "path", destPath)
		return fmt.Errorf("destination already exists")
	} else if !os.IsNotExist(err) {
		ops.logger.Error("Failed to check destination", "path", destPath, "error", err)
		return fmt.Errorf("failed to check destination: %w", err)
	}

	err := rename(sourcePath, destPath)
	if err != nil {
		// Detect cross-device rename and fallback to copy/remove
		if linkErr, ok := err.(*os.LinkError); ok && errors.Is(linkErr.Err, syscall.EXDEV) {
			ops.logger.Debug("Cross-device rename detected, falling back to copy", "source", sourcePath, "destination", destPath)

			if copyErr := copyRecursive(sourcePath, destPath); copyErr != nil {
				ops.logger.Error("Copy fallback failed", "error", copyErr)
				return fmt.Errorf("failed to copy during move: %w", copyErr)
			}
			if rmErr := os.RemoveAll(sourcePath); rmErr != nil {
				ops.logger.Error("Failed to remove source after copy", "error", rmErr)
				return fmt.Errorf("failed to remove source after copy: %w", rmErr)
			}
		} else {
			ops.logger.Error("Failed to move file", "source", sourcePath, "destination", destPath, "error", err)
			return fmt.Errorf("failed to move file: %w", err)
		}
	}

	ops.logger.Info("File moved successfully", "source", sourcePath, "destination", destPath)
	return nil
}

// copyRecursive copies a file or directory from src to dst.
// It preserves file permissions and directory structure.
func copyRecursive(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst, info.Mode())
}

// copyDir recursively copies a directory tree.
func copyDir(srcDir, dstDir string) error {
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dstDir, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target, info.Mode())
	})
}

// copyFile copies a single file from src to dst using the provided permissions.
func copyFile(src, dst string, perm fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// SearchFiles recursively searches for files matching a pattern
func (ops *Operations) SearchFiles(rootPath, pattern string, excludePatterns []string) ([]string, error) {
	// Input validation per Rule 7
	if rootPath == "" {
		return nil, fmt.Errorf("root path cannot be empty")
	}
	if pattern == "" {
		return nil, fmt.Errorf("search pattern cannot be empty")
	}

	ops.logger.Debug("Searching files", "root", rootPath, "pattern", pattern, "excludes", excludePatterns)

	var results []string
	lowerPattern := strings.ToLower(pattern)

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			ops.logger.Warn("Error walking directory", "path", path, "error", err)
			return nil // Continue walking
		}

		// Validate each path before processing to ensure we stay within allowed directories
		if _, valErr := ops.pathValidator.ValidatePath(path); valErr != nil {
			ops.logger.Warn("Path validation failed", "path", path, "error", valErr)
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check exclude patterns
		relativePath, relErr := filepath.Rel(rootPath, path)
		if relErr == nil && ops.shouldExclude(relativePath, excludePatterns) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if filename matches pattern
		filename := strings.ToLower(d.Name())
		if strings.Contains(filename, lowerPattern) {
			results = append(results, path)
		}

		return nil
	})

	if err != nil {
		ops.logger.Error("Failed to search files", "error", err)
		return nil, fmt.Errorf("failed to search files: %w", err)
	}

	ops.logger.Debug("File search completed", "root", rootPath, "results_count", len(results))
	return results, nil
}

// shouldExclude checks if a path should be excluded based on patterns
func (ops *Operations) shouldExclude(relativePath string, excludePatterns []string) bool {
	// Check exclude patterns
	for _, pattern := range excludePatterns {

		// Add glob pattern if not present
		if !strings.Contains(pattern, "*") {
			pattern = "**/" + pattern + "/**"
		}

		matched, err := doublestar.Match(pattern, relativePath)
		if err == nil && matched {
			return true
		}
	}

	return false
}

// GetFileInfo retrieves detailed information about a file or directory
func (ops *Operations) GetFileInfo(filePath string) (*FileInfo, error) {
	// Input validation per Rule 7
	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	ops.logger.Debug("Getting file info", "path", filePath)

	stat, err := os.Stat(filePath)
	if err != nil {
		ops.logger.Error("Failed to get file info", "path", filePath, "error", err)
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	info := &FileInfo{
		Size:        stat.Size(),
		Modified:    stat.ModTime(),
		IsDirectory: stat.IsDir(),
		IsFile:      stat.Mode().IsRegular(),
		Permissions: fmt.Sprintf("%o", stat.Mode().Perm()),
	}

	// Get creation and access times (platform-specific)
	if sys := ops.getSystemTimes(stat); sys != nil {
		info.Created = sys.Created
		info.Accessed = sys.Accessed
	} else {
		// Fallback to modification time
		info.Created = stat.ModTime()
		info.Accessed = stat.ModTime()
	}

	ops.logger.Debug("File info retrieved successfully", "path", filePath)
	return info, nil
}
