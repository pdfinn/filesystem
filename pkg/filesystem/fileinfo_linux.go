//go:build linux
// +build linux

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

// getSystemTimes extracts creation and access times on Linux
func (ops *Operations) getSystemTimes(stat os.FileInfo) *SystemTimes {
	if sys, ok := stat.Sys().(*syscall.Stat_t); ok {
		// Linux does not provide a true creation time through Stat_t
		// so use the change time (Ctim) as the closest approximation.
		created := time.Unix(sys.Ctim.Sec, sys.Ctim.Nsec)
		accessed := time.Unix(sys.Atim.Sec, sys.Atim.Nsec)
		return &SystemTimes{Created: created, Accessed: accessed}
	}
	return nil
}
