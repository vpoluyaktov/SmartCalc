//go:build darwin

package nettools

import (
	"syscall"
	"time"
)

// createTimeval creates a syscall.Timeval for Darwin/macOS (Usec is int32)
func createTimeval(timeout time.Duration) syscall.Timeval {
	return syscall.Timeval{
		Sec:  int64(timeout.Seconds()),
		Usec: int32(int64(timeout.Nanoseconds()/1000) % 1000000),
	}
}
