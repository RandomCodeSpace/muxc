package store

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

func (s *Store) CreateSession(session *Session) error {
	return s.db.Create(session).Error
}

func (s *Store) GetSession(name string) (*Session, error) {
	var session Session
	err := s.db.Preload("Tags").Preload("History").Where("name = ?", name).First(&session).Error
	if err != nil {
		return nil, fmt.Errorf("session %q not found: %w", name, err)
	}
	return &session, nil
}

func (s *Store) GetSessionByID(sessionID string) (*Session, error) {
	var session Session
	err := s.db.Preload("Tags").Preload("History").Where("session_id = ?", sessionID).First(&session).Error
	if err != nil {
		return nil, fmt.Errorf("session with id %q not found: %w", sessionID, err)
	}
	return &session, nil
}

func (s *Store) ListSessions(status string, tag string, includeArchived bool) ([]Session, error) {
	var sessions []Session
	q := s.db.Preload("Tags").Preload("History")

	if status != "" {
		q = q.Where("status = ?", status)
	} else if !includeArchived {
		q = q.Where("status IN ?", []string{"active", "detached"})
	}

	if tag != "" {
		q = q.Where("id IN (?)",
			s.db.Model(&Tag{}).Select("session_id").Where("value = ?", tag),
		)
	}

	q = q.Order("accessed_at DESC")

	if err := q.Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (s *Store) ListSessionNames(prefix string) ([]string, error) {
	var names []string
	q := s.db.Model(&Session{}).Select("name")
	if prefix != "" {
		q = q.Where("name LIKE ?", prefix+"%")
	}
	q = q.Order("name ASC")
	if err := q.Pluck("name", &names).Error; err != nil {
		return nil, err
	}
	return names, nil
}

func (s *Store) UpdateSession(session *Session) error {
	return s.db.Save(session).Error
}

func (s *Store) DeleteSession(name string) error {
	result := s.db.Where("name = ?", name).Delete(&Session{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *Store) AddTag(sessionID uint, value string) error {
	tag := Tag{SessionID: sessionID, Value: value}
	return s.db.Create(&tag).Error
}

func (s *Store) RemoveTag(sessionID uint, value string) error {
	result := s.db.Where("session_id = ? AND value = ?", sessionID, value).Delete(&Tag{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("tag %q not found on session %d", value, sessionID)
	}
	return nil
}

func (s *Store) AppendHistory(sessionID uint, event string, details string) error {
	entry := HistoryEntry{
		SessionID: sessionID,
		Timestamp: time.Now(),
		Event:     event,
		Details:   details,
	}
	return s.db.Create(&entry).Error
}

func (s *Store) GetActiveSessions() ([]Session, error) {
	var sessions []Session
	err := s.db.Preload("Tags").Where("status = ?", "active").Find(&sessions).Error
	if err != nil {
		return nil, err
	}
	return sessions, nil
}
