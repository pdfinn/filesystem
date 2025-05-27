//go:build !darwin

package filesystem

import (
	"os"
	"time"
)

// SystemTimes holds platform-specific time information
// On non-Darwin systems creation and access times may not be available.
type SystemTimes struct {
	Created  time.Time
	Accessed time.Time
}

// getSystemTimes returns nil on non-Darwin platforms
func (ops *Operations) getSystemTimes(stat os.FileInfo) *SystemTimes {
	return nil
}
