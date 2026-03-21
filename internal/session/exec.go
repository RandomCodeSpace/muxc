package session

import (
	"fmt"
	"os"
	"syscall"
)

// ExecClaude replaces the current process with the claude binary.
// This function does not return on success.
func ExecClaude(claudeBin string, args []string, cwd string) error {
	// Change to the session's working directory
	if err := os.Chdir(cwd); err != nil {
		return fmt.Errorf("chdir to %s: %w", cwd, err)
	}
	// Build argv (binary name must be first)
	argv := append([]string{claudeBin}, args...)
	env := os.Environ()
	// This replaces the current process — does not return on success
	return syscall.Exec(claudeBin, argv, env)
}
