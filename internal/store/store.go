package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Store struct {
	dir string
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating sessions directory: %w", err)
	}
	return &Store{dir: dir}, nil
}

func (s *Store) Close() error { return nil }

func (s *Store) sessionPath(name string) string {
	return filepath.Join(s.dir, name+".json")
}

func (s *Store) readSession(name string) (*Session, error) {
	data, err := os.ReadFile(s.sessionPath(name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session %q not found", name)
		}
		return nil, err
	}
	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("parsing session %q: %w", name, err)
	}
	return &sess, nil
}

func (s *Store) writeSession(sess *Session) error {
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.sessionPath(sess.Name) + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.sessionPath(sess.Name))
}

func (s *Store) allSessions() ([]Session, error) {
	entries, err := filepath.Glob(filepath.Join(s.dir, "*.json"))
	if err != nil {
		return nil, err
	}
	var sessions []Session
	for _, path := range entries {
		name := strings.TrimSuffix(filepath.Base(path), ".json")
		sess, err := s.readSession(name)
		if err != nil {
			continue
		}
		sessions = append(sessions, *sess)
	}
	return sessions, nil
}

// CreateSession writes a new session file. Returns an error if the name is taken.
func (s *Store) CreateSession(sess *Session) error {
	if _, err := os.Stat(s.sessionPath(sess.Name)); err == nil {
		return fmt.Errorf("session %q already exists", sess.Name)
	}
	return s.writeSession(sess)
}

func (s *Store) GetSession(name string) (*Session, error) {
	return s.readSession(name)
}

func (s *Store) GetSessionByID(sessionID string) (*Session, error) {
	sessions, err := s.allSessions()
	if err != nil {
		return nil, err
	}
	for _, sess := range sessions {
		if sess.SessionID == sessionID {
			return &sess, nil
		}
	}
	return nil, fmt.Errorf("session with id %q not found", sessionID)
}

func (s *Store) ListSessions(status string, tag string, includeArchived bool) ([]Session, error) {
	sessions, err := s.allSessions()
	if err != nil {
		return nil, err
	}
	var filtered []Session
	for _, sess := range sessions {
		if status != "" && sess.Status != status {
			continue
		}
		if status == "" && !includeArchived && sess.Status == "archived" {
			continue
		}
		if tag != "" {
			found := false
			for _, t := range sess.Tags {
				if t == tag {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		filtered = append(filtered, sess)
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].AccessedAt.After(filtered[j].AccessedAt)
	})
	return filtered, nil
}

func (s *Store) ListSessionNames(prefix string) ([]string, error) {
	entries, err := filepath.Glob(filepath.Join(s.dir, prefix+"*.json"))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, path := range entries {
		name := strings.TrimSuffix(filepath.Base(path), ".json")
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func (s *Store) UpdateSession(sess *Session) error {
	return s.writeSession(sess)
}

func (s *Store) DeleteSession(name string) error {
	path := s.sessionPath(name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("session %q not found", name)
	}
	return os.Remove(path)
}

func (s *Store) RenameSession(oldName, newName string) error {
	sess, err := s.readSession(oldName)
	if err != nil {
		return err
	}
	if _, err := os.Stat(s.sessionPath(newName)); err == nil {
		return fmt.Errorf("session %q already exists", newName)
	}
	sess.Name = newName
	if err := s.writeSession(sess); err != nil {
		return err
	}
	return os.Remove(s.sessionPath(oldName))
}

func (s *Store) AddTag(name string, value string) error {
	sess, err := s.readSession(name)
	if err != nil {
		return err
	}
	for _, t := range sess.Tags {
		if t == value {
			return fmt.Errorf("tag %q already exists on session %q", value, name)
		}
	}
	sess.Tags = append(sess.Tags, value)
	return s.writeSession(sess)
}

func (s *Store) RemoveTag(name string, value string) error {
	sess, err := s.readSession(name)
	if err != nil {
		return err
	}
	idx := -1
	for i, t := range sess.Tags {
		if t == value {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("tag %q not found on session %q", value, name)
	}
	sess.Tags = append(sess.Tags[:idx], sess.Tags[idx+1:]...)
	return s.writeSession(sess)
}

func (s *Store) AppendHistory(name string, event string, details string) error {
	sess, err := s.readSession(name)
	if err != nil {
		return err
	}
	sess.History = append(sess.History, HistoryEntry{
		Timestamp: time.Now(),
		Event:     event,
		Details:   details,
	})
	return s.writeSession(sess)
}

func (s *Store) GetActiveSessions() ([]Session, error) {
	return s.ListSessions("active", "", true)
}
