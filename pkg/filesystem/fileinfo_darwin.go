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

// getSystemTimes extracts creation and access times on macOS
func (ops *Operations) getSystemTimes(stat os.FileInfo) *SystemTimes {
	if sys, ok := stat.Sys().(*syscall.Stat_t); ok {
		return &SystemTimes{
			Created:  time.Unix(sys.Birthtimespec.Sec, sys.Birthtimespec.Nsec),
			Accessed: time.Unix(sys.Atimespec.Sec, sys.Atimespec.Nsec),
		}
	}
	return nil
}
