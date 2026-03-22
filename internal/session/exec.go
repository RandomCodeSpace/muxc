package session

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var unsafeChars = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

// TmuxSessionName returns the tmux session name for a muxc session.
// Sanitizes the name to only allow [a-zA-Z0-9_-].
func TmuxSessionName(name string) string {
	sanitized := unsafeChars.ReplaceAllString(name, "-")
	return "muxc-" + sanitized
}

// TmuxHasSession checks if a tmux session exists.
func TmuxHasSession(tmuxBin, sessionName string) bool {
	cmd := exec.Command(tmuxBin, "has-session", "-t", sessionName)
	return cmd.Run() == nil
}

// TmuxNewSession creates a new detached tmux session running the given command.
// The command is wrapped to show a message when claude exits.
func TmuxNewSession(tmuxBin, sessionName, cwd string, command []string) error {
	// Wrap: run claude, then show exit message so tmux doesn't vanish instantly
	shellCmd := strings.Join(command, " ") + `; echo ""; echo "Session ended. Press Enter to close."; read`

	args := []string{"new-session", "-d", "-s", sessionName}
	if cwd != "" {
		args = append(args, "-c", cwd)
	}
	args = append(args, "--", "sh", "-c", shellCmd)

	cmd := exec.Command(tmuxBin, args...)
	return cmd.Run()
}

// TmuxAttach attaches to an existing tmux session.
// If already inside tmux ($TMUX is set), uses switch-client instead to avoid nesting.
func TmuxAttach(tmuxBin, sessionName string) error {
	var cmd *exec.Cmd
	if os.Getenv("TMUX") != "" {
		cmd = exec.Command(tmuxBin, "switch-client", "-t", sessionName)
	} else {
		cmd = exec.Command(tmuxBin, "attach-session", "-t", sessionName)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// TmuxKillSession kills a tmux session.
func TmuxKillSession(tmuxBin, sessionName string) error {
	cmd := exec.Command(tmuxBin, "kill-session", "-t", sessionName)
	return cmd.Run()
}

// ListTmuxSessions returns the names of all active muxc-* tmux sessions.
func ListTmuxSessions(tmuxBin string) map[string]bool {
	if tmuxBin == "" {
		return nil
	}
	cmd := exec.Command(tmuxBin, "list-sessions", "-F", "#{session_name}")
	out, err := cmd.Output()
	if err != nil {
		return nil // no tmux server or no sessions
	}
	result := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.HasPrefix(line, "muxc-") {
			result[line] = true
		}
	}
	return result
}
