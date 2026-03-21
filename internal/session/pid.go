package session

import (
	"fmt"
	"os"
	"strings"
	"syscall"
)

// CheckPID returns true if the PID is alive and belongs to a claude process
func CheckPID(pid int) bool {
	if pid <= 0 {
		return false
	}
	// Check if process exists
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return false
	}
	// Verify it's a claude process (guard against PID reuse)
	cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return false
	}
	return strings.Contains(string(cmdline), "claude")
}
