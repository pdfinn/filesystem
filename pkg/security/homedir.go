package security

import (
	"os"
	"path/filepath"
)

// ExpandHomePath expands ~ and ~/ in file paths.
func ExpandHomePath(path string) string {
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
