package store

import "time"

type Session struct {
	ID         uint      `gorm:"primaryKey"`
	Name       string    `gorm:"uniqueIndex;size:64;not null"`
	SessionID  string    `gorm:"uniqueIndex;not null"`
	ClaudePID  int       `gorm:"default:0"`
	Cwd        string    `gorm:"not null"`
	Status     string    `gorm:"index;not null;default:detached"`
	ClaudeArgs string    `gorm:"type:text"`
	Notes      string    `gorm:"type:text"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
	AccessedAt time.Time
	Tags       []Tag          `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
	History    []HistoryEntry `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
}

type Tag struct {
	ID        uint   `gorm:"primaryKey"`
	SessionID uint   `gorm:"uniqueIndex:idx_session_tag;not null"`
	Value     string `gorm:"uniqueIndex:idx_session_tag;not null"`
}

type HistoryEntry struct {
	ID        uint      `gorm:"primaryKey"`
	SessionID uint      `gorm:"index;not null"`
	Timestamp time.Time `gorm:"autoCreateTime"`
	Event     string    `gorm:"not null"`
	Details   string
}
