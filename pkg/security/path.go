package security

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// PathValidator provides secure path validation and access control
type PathValidator struct {
	allowedDirectories []string
	logger             *slog.Logger
}

// NewPathValidator creates a new path validator with allowed directories
func NewPathValidator(allowedDirs []string, logger *slog.Logger) *PathValidator {
	// Pre-allocate with known size per Rule 3 (no dynamic allocation after init)
	normalizedDirs := make([]string, 0, len(allowedDirs))

	// Normalize all allowed directories with fixed upper bound per Rule 2
	for i := 0; i < len(allowedDirs) && i < 1000; i++ {
		dir := filepath.Clean(allowedDirs[i])
		normalizedDirs = append(normalizedDirs, dir)
	}

	return &PathValidator{
		allowedDirectories: normalizedDirs,
		logger:             logger,
	}
}

// ValidatePath securely validates a requested path against allowed directories
// Returns the real absolute path if valid, error otherwise
func (pv *PathValidator) ValidatePath(requestedPath string) (string, error) {
	// Input validation per Rule 7 (check parameter validity)
	if requestedPath == "" {
		pv.logger.Warn("Empty path provided for validation")
		return "", fmt.Errorf("path cannot be empty")
	}

	// Expand home directory if needed
	expandedPath := pv.expandHomePath(requestedPath)

	// Convert to absolute path
	var absolutePath string
	if filepath.IsAbs(expandedPath) {
		absolutePath = filepath.Clean(expandedPath)
	} else {
		workDir, err := os.Getwd()
		if err != nil {
			pv.logger.Error("Failed to get working directory", "error", err)
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
		absolutePath = filepath.Clean(filepath.Join(workDir, expandedPath))
	}

	// Check if path is within allowed directories
	if !pv.isPathAllowed(absolutePath) {
		pv.logger.Warn("Access denied to path outside allowed directories",
			"requested_path", requestedPath,
			"absolute_path", absolutePath,
			"allowed_dirs", pv.allowedDirectories)
		return "", fmt.Errorf("access denied - path outside allowed directories: %s", absolutePath)
	}

	// Handle symlinks by checking their real path
	realPath, err := pv.validateRealPath(absolutePath)
	if err != nil {
		return "", err
	}

	pv.logger.Debug("Path validation successful",
		"requested_path", requestedPath,
		"real_path", realPath)

	return realPath, nil
}

// expandHomePath expands ~ and ~/ in file paths
func (pv *PathValidator) expandHomePath(path string) string {
	if path == "~" {
		if homeDir, err := os.UserHomeDir(); err == nil {
			return homeDir
		}
	} else if len(path) > 1 && path[:2] == "~/" {
		if homeDir, err := os.UserHomeDir(); err == nil {
			return filepath.Join(homeDir, path[2:])
		}
	}
	return path
}

// isPathAllowed checks if a path is within any allowed directory
func (pv *PathValidator) isPathAllowed(absolutePath string) bool {
	normalizedPath := filepath.Clean(absolutePath)

	// Check against each allowed directory with fixed upper bound per Rule 2
	for i := 0; i < len(pv.allowedDirectories) && i < 1000; i++ {
		allowedDir := pv.allowedDirectories[i]

		// Check if path starts with allowed directory
		if pv.isPathUnderDirectory(normalizedPath, allowedDir) {
			return true
		}
	}

	return false
}

// isPathUnderDirectory checks if a path is under a given directory
func (pv *PathValidator) isPathUnderDirectory(path, dir string) bool {
	// Ensure both paths end with separator for proper comparison
	if !strings.HasSuffix(dir, string(filepath.Separator)) {
		dir += string(filepath.Separator)
	}
	if !strings.HasSuffix(path, string(filepath.Separator)) {
		path += string(filepath.Separator)
	}

	return strings.HasPrefix(path, dir)
}

// validateRealPath handles symlinks and validates the real path
func (pv *PathValidator) validateRealPath(absolutePath string) (string, error) {
	// Try to get real path (resolves symlinks)
	realPath, err := filepath.EvalSymlinks(absolutePath)
	if err != nil {
		// If file doesn't exist, check parent directory
		parentDir := filepath.Dir(absolutePath)
		realParentPath, parentErr := filepath.EvalSymlinks(parentDir)
		if parentErr != nil {
			pv.logger.Debug("Parent directory does not exist",
				"parent_dir", parentDir,
				"error", parentErr)
			return "", fmt.Errorf("parent directory does not exist: %s", parentDir)
		}

		// Validate parent directory is allowed
		if !pv.isPathAllowed(realParentPath) {
			pv.logger.Warn("Parent directory outside allowed directories",
				"parent_dir", realParentPath)
			return "", fmt.Errorf("access denied - parent directory outside allowed directories")
		}

		// Return the original absolute path for new files
		return absolutePath, nil
	}

	// Validate real path is allowed
	if !pv.isPathAllowed(realPath) {
		pv.logger.Warn("Symlink target outside allowed directories",
			"symlink_target", realPath)
		return "", fmt.Errorf("access denied - symlink target outside allowed directories")
	}

	return realPath, nil
}

// GetAllowedDirectories returns a copy of allowed directories
func (pv *PathValidator) GetAllowedDirectories() []string {
	// Return copy to prevent modification per Rule 6 (data hiding)
	dirs := make([]string, len(pv.allowedDirectories))
	copy(dirs, pv.allowedDirectories)
	return dirs
}
