package session

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// claudeSessionFile represents the JSON structure Claude Code writes to ~/.claude/sessions/<pid>.json
type claudeSessionFile struct {
	PID       int    `json:"pid"`
	SessionID string `json:"sessionId"`
}

// RunResult contains the outcome of running Claude.
type RunResult struct {
	PID           int
	SessionID     string
	ResumeFailure bool // true if Claude couldn't find the session to resume
	Err           error
}

// RunClaude runs the claude binary as a child process, forwarding stdio and signals.
// It captures the Claude session ID during execution.
func RunClaude(claudeBin string, args []string, cwd string) RunResult {
	return runClaude(claudeBin, args, cwd, false)
}

// RunClaudeResume runs the claude binary for a resume attempt.
// If Claude exits because it can't find the session, ResumeFailure is set to true.
func RunClaudeResume(claudeBin string, args []string, cwd string) RunResult {
	return runClaude(claudeBin, args, cwd, true)
}

func runClaude(claudeBin string, args []string, cwd string, detectResume bool) RunResult {
	if err := os.Chdir(cwd); err != nil {
		return RunResult{Err: fmt.Errorf("chdir to %s: %w", cwd, err)}
	}

	cmd := exec.Command(claudeBin, args...)
	cmd.Stdin = os.Stdin

	// If detecting resume failure, tee stdout and stderr to capture output
	// while still showing to user. Claude may write "No conversation found"
	// to either stream.
	var stdoutBuf, stderrBuf bytes.Buffer
	if detectResume {
		cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
		cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Start(); err != nil {
		return RunResult{Err: fmt.Errorf("starting claude: %w", err)}
	}

	pid := cmd.Process.Pid

	// Read session ID in background while Claude is running
	sessionIDCh := make(chan string, 1)
	go func() {
		if id, err := readClaudeSessionID(pid); err == nil {
			sessionIDCh <- id
		} else {
			sessionIDCh <- ""
		}
	}()

	// Forward signals to the child process
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for sig := range sigs {
			_ = cmd.Process.Signal(sig)
		}
	}()

	waitErr := cmd.Wait()
	signal.Stop(sigs)
	close(sigs)

	// Collect session ID
	var sessionID string
	select {
	case sessionID = <-sessionIDCh:
	default:
	}

	result := RunResult{PID: pid, SessionID: sessionID, Err: waitErr}

	// Check if this was a resume failure — Claude may write the error to
	// either stdout or stderr depending on the version.
	if detectResume && waitErr != nil {
		combined := stdoutBuf.String() + stderrBuf.String()
		if strings.Contains(combined, "No conversation found") {
			result.ResumeFailure = true
		}
	}

	return result
}

// readClaudeSessionID reads the session ID from Claude Code's session file for the given PID.
// Claude writes ~/.claude/sessions/<pid>.json shortly after startup.
func readClaudeSessionID(pid int) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	sessionFile := filepath.Join(home, ".claude", "sessions", fmt.Sprintf("%d.json", pid))

	// Poll for up to 10 seconds — Claude needs time to start and write the file
	var data []byte
	for i := 0; i < 100; i++ {
		data, err = os.ReadFile(sessionFile)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		return "", fmt.Errorf("reading claude session file %s: %w", sessionFile, err)
	}

	var sf claudeSessionFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return "", fmt.Errorf("parsing claude session file: %w", err)
	}
	if sf.SessionID == "" {
		return "", fmt.Errorf("no sessionId in claude session file %s", sessionFile)
	}
	return sf.SessionID, nil
}
