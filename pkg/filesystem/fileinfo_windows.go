//go:build windows
// +build windows

package filesystem

import (
	"os"
	"syscall"
	"time"
)

// SystemTimes holds platform-specific time information
type SystemTimes struct {
	Created  time.Time
	Accessed time.Time
}

// getSystemTimes extracts creation and access times on Windows
func (ops *Operations) getSystemTimes(stat os.FileInfo) *SystemTimes {
	if sys, ok := stat.Sys().(*syscall.Win32FileAttributeData); ok {
		created := time.Unix(0, sys.CreationTime.Nanoseconds())
		accessed := time.Unix(0, sys.LastAccessTime.Nanoseconds())
		return &SystemTimes{Created: created, Accessed: accessed}
	}
	return nil
}
