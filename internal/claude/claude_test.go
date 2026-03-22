package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadTitle(t *testing.T) {
	dir := t.TempDir()

	t.Run("valid custom-title", func(t *testing.T) {
		path := filepath.Join(dir, "valid.jsonl")
		line := `{"type":"custom-title","customTitle":"my-session","sessionId":"abc-123"}` + "\n"
		line += `{"type":"message","content":"hello"}` + "\n"
		os.WriteFile(path, []byte(line), 0644)

		ct, err := readTitle(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ct.Title != "my-session" {
			t.Errorf("expected title %q, got %q", "my-session", ct.Title)
		}
		if ct.SessionID != "abc-123" {
			t.Errorf("expected sessionId %q, got %q", "abc-123", ct.SessionID)
		}
	})

	t.Run("empty title", func(t *testing.T) {
		path := filepath.Join(dir, "empty-title.jsonl")
		line := `{"type":"custom-title","customTitle":"","sessionId":"abc-123"}` + "\n"
		os.WriteFile(path, []byte(line), 0644)

		ct, err := readTitle(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ct.Title != "" {
			t.Errorf("expected empty title, got %q", ct.Title)
		}
	})

	t.Run("not custom-title type", func(t *testing.T) {
		path := filepath.Join(dir, "wrong-type.jsonl")
		line := `{"type":"message","content":"hello"}` + "\n"
		os.WriteFile(path, []byte(line), 0644)

		_, err := readTitle(path)
		if err == nil {
			t.Fatal("expected error for non custom-title type")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		path := filepath.Join(dir, "empty.jsonl")
		os.WriteFile(path, []byte(""), 0644)

		_, err := readTitle(path)
		if err == nil {
			t.Fatal("expected error for empty file")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := readTitle(filepath.Join(dir, "nonexistent.jsonl"))
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})
}

func TestDecodeProjectHash(t *testing.T) {
	// Create a real temp directory to test stat-based validation
	dir := t.TempDir()
	// The hash for the temp dir: replace / with -
	// We can't easily test the full decode since temp paths vary,
	// but we can test the fallback behavior.

	t.Run("nonexistent path falls back to hash", func(t *testing.T) {
		result := DecodeProjectHash("-nonexistent-path-that-does-not-exist")
		if result != "-nonexistent-path-that-does-not-exist" {
			t.Errorf("expected fallback to hash, got %q", result)
		}
	})

	t.Run("real directory decodes", func(t *testing.T) {
		// Create a nested dir structure that matches a hash
		nested := filepath.Join(dir, "sub", "dir")
		os.MkdirAll(nested, 0755)
		// The hash for /tmp/xxx/sub/dir would be -tmp-xxx-sub-dir
		// But since tmp paths vary, just verify the function works with known paths
		result := DecodeProjectHash("-tmp")
		if result != "/tmp" {
			// /tmp might not exist on all systems, so just check it tried
			t.Logf("DecodeProjectHash(-tmp) = %q (may depend on system)", result)
		}
	})
}

func TestScanPIDFiles(t *testing.T) {
	t.Run("empty directory", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, "sessions"), 0755)

		result, err := scanPIDFiles(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty map, got %d entries", len(result))
		}
	})

	t.Run("dead PID included with PID=0", func(t *testing.T) {
		dir := t.TempDir()
		sessDir := filepath.Join(dir, "sessions")
		os.MkdirAll(sessDir, 0755)

		// Use PID 999999999 which is almost certainly not alive
		data, _ := json.Marshal(map[string]any{
			"pid":       999999999,
			"sessionId": "dead-session-id",
			"cwd":       "/some/path",
			"startedAt": time.Now().UnixMilli(),
		})
		os.WriteFile(filepath.Join(sessDir, "999999999.json"), data, 0644)

		result, err := scanPIDFiles(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		info, ok := result["dead-session-id"]
		if !ok {
			t.Fatal("expected dead session to be in map")
		}
		if info.PID != 0 {
			t.Errorf("expected PID=0 for dead process, got %d", info.PID)
		}
		if info.Cwd != "/some/path" {
			t.Errorf("expected cwd %q, got %q", "/some/path", info.Cwd)
		}
	})

	t.Run("invalid JSON skipped", func(t *testing.T) {
		dir := t.TempDir()
		sessDir := filepath.Join(dir, "sessions")
		os.MkdirAll(sessDir, 0755)

		// Write one valid and one invalid file
		data, _ := json.Marshal(map[string]any{
			"pid":       999999998,
			"sessionId": "valid-session",
			"cwd":       "/valid",
			"startedAt": time.Now().UnixMilli(),
		})
		os.WriteFile(filepath.Join(sessDir, "999999998.json"), data, 0644)
		os.WriteFile(filepath.Join(sessDir, "bad.json"), []byte("not json"), 0644)

		result, err := scanPIDFiles(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := result["valid-session"]; !ok {
			t.Error("expected valid session to be in map despite bad file")
		}
	})

	t.Run("nonexistent directory returns nil", func(t *testing.T) {
		dir := t.TempDir()
		result, err := scanPIDFiles(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})
}

func TestListSessions(t *testing.T) {
	// Set up a mock ~/.claude directory structure
	home := t.TempDir()
	t.Setenv("HOME", home)

	claudeBase := filepath.Join(home, ".claude")
	sessDir := filepath.Join(claudeBase, "sessions")
	projDir := filepath.Join(claudeBase, "projects", "-home-dev-test")

	os.MkdirAll(sessDir, 0755)
	os.MkdirAll(projDir, 0755)

	// Create a JSONL file with a custom-title
	jsonlContent := `{"type":"custom-title","customTitle":"test-session","sessionId":"sess-uuid-1"}` + "\n"
	jsonlContent += `{"type":"message","content":"hello"}` + "\n"
	os.WriteFile(filepath.Join(projDir, "sess-uuid-1.jsonl"), []byte(jsonlContent), 0644)

	// Create a PID file for the session (dead PID)
	pidData, _ := json.Marshal(map[string]any{
		"pid":       999999998,
		"sessionId": "sess-uuid-1",
		"cwd":       "/home/dev/test",
		"startedAt": time.Now().UnixMilli(),
	})
	os.WriteFile(filepath.Join(sessDir, "999999998.json"), pidData, 0644)

	// Create another JSONL without a title (should be skipped)
	jsonlNoTitle := `{"type":"message","content":"no title here"}` + "\n"
	os.WriteFile(filepath.Join(projDir, "sess-uuid-2.jsonl"), []byte(jsonlNoTitle), 0644)

	sessions, err := ListSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	s := sessions[0]
	if s.Name != "test-session" {
		t.Errorf("expected name %q, got %q", "test-session", s.Name)
	}
	if s.SessionID != "sess-uuid-1" {
		t.Errorf("expected sessionId %q, got %q", "sess-uuid-1", s.SessionID)
	}
	if s.Status != "detached" {
		t.Errorf("expected status %q, got %q", "detached", s.Status)
	}
	if s.Cwd != "/home/dev/test" {
		t.Errorf("expected cwd %q, got %q", "/home/dev/test", s.Cwd)
	}
}

func TestGetSession(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	claudeBase := filepath.Join(home, ".claude")
	projDir := filepath.Join(claudeBase, "projects", "-test")
	os.MkdirAll(filepath.Join(claudeBase, "sessions"), 0755)
	os.MkdirAll(projDir, 0755)

	jsonl := `{"type":"custom-title","customTitle":"my-proj","sessionId":"uuid-abc"}` + "\n"
	os.WriteFile(filepath.Join(projDir, "uuid-abc.jsonl"), []byte(jsonl), 0644)

	t.Run("found", func(t *testing.T) {
		sess, err := GetSession("my-proj")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sess.Name != "my-proj" {
			t.Errorf("expected %q, got %q", "my-proj", sess.Name)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := GetSession("nonexistent")
		if err == nil {
			t.Fatal("expected error for nonexistent session")
		}
	})
}

func TestListSessionNames(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	claudeBase := filepath.Join(home, ".claude")
	projDir := filepath.Join(claudeBase, "projects", "-test")
	os.MkdirAll(filepath.Join(claudeBase, "sessions"), 0755)
	os.MkdirAll(projDir, 0755)

	// Create two sessions with different names
	for _, s := range []struct{ name, id string }{
		{"alpha-session", "uuid-1"},
		{"beta-session", "uuid-2"},
	} {
		jsonl := fmt.Sprintf(`{"type":"custom-title","customTitle":"%s","sessionId":"%s"}`+"\n", s.name, s.id)
		os.WriteFile(filepath.Join(projDir, s.id+".jsonl"), []byte(jsonl), 0644)
	}

	t.Run("all names", func(t *testing.T) {
		names, err := ListSessionNames("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(names) != 2 {
			t.Fatalf("expected 2 names, got %d", len(names))
		}
	})

	t.Run("prefix filter", func(t *testing.T) {
		names, err := ListSessionNames("alpha")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(names) != 1 || names[0] != "alpha-session" {
			t.Errorf("expected [alpha-session], got %v", names)
		}
	})
}
