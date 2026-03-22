package session

import "testing"

func TestTmuxSessionName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"myproject", "muxc-myproject"},
		{"my.project", "muxc-my-project"},
		{"my:project", "muxc-my-project"},
		{"my project", "muxc-my-project"},
		{"my$project", "muxc-my-project"},
		{"simple", "muxc-simple"},
		{"with-dash", "muxc-with-dash"},
		{"with_under", "muxc-with_under"},
		{"UPPER", "muxc-UPPER"},
		{"123", "muxc-123"},
		{"a/b/c", "muxc-a-b-c"},
	}
	for _, tt := range tests {
		got := TmuxSessionName(tt.input)
		if got != tt.want {
			t.Errorf("TmuxSessionName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestListTmuxSessions(t *testing.T) {
	// With empty tmuxBin, should return nil gracefully
	result := ListTmuxSessions("")
	if result != nil {
		t.Errorf("expected nil for empty tmuxBin, got %v", result)
	}
}
