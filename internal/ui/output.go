package ui

import (
	"fmt"
	"os"
	"time"
)

func Die(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "\u274c "+format+"\n", a...)
	os.Exit(1)
}

func Warn(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "\u26a0\ufe0f  "+format+"\n", a...)
}

func Info(format string, a ...any) {
	fmt.Printf("\u2139\ufe0f  "+format+"\n", a...)
}

func Success(format string, a ...any) {
	fmt.Printf("\u2728 "+format+"\n", a...)
}

func Action(format string, a ...any) {
	fmt.Printf("\U0001f517 "+format+"\n", a...)
}

func Nav(format string, a ...any) {
	fmt.Printf("\U0001f4c2 "+format+"\n", a...)
}

func Launch(format string, a ...any) {
	fmt.Printf("\U0001f680 "+format+"\n", a...)
}

// StatusIcon returns the icon for a session status.
func StatusIcon(status string) string {
	switch status {
	case "active":
		return "▶"
	case "detached":
		return "⏸"
	case "archived":
		return "◼"
	default:
		return "?"
	}
}

// RelativeTime formats a time.Time as a human-readable relative duration
// (e.g., "2m ago", "3h ago", "1d ago").
func RelativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%dy ago", int(d.Hours()/(24*365)))
	}
}

// ShortenPath replaces the user's home directory prefix with ~.
func ShortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if len(path) >= len(home) && path[:len(home)] == home {
		return "~" + path[len(home):]
	}
	return path
}
