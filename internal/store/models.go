package store

import "time"

type Session struct {
	Name       string         `json:"name"`
	SessionID  string         `json:"session_id"`
	ClaudePID  int            `json:"claude_pid"`
	Cwd        string         `json:"cwd"`
	Status     string         `json:"status"`
	ClaudeArgs []string       `json:"claude_args,omitempty"`
	Notes      string         `json:"notes,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	AccessedAt time.Time      `json:"accessed_at"`
	Tags       []string       `json:"tags,omitempty"`
	History    []HistoryEntry `json:"history,omitempty"`
}

type HistoryEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Event     string    `json:"event"`
	Details   string    `json:"details,omitempty"`
}
