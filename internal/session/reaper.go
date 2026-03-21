package session

import (
	"fmt"
	"time"

	"github.com/RandomCodeSpace/muxc/internal/store"
)

// ReapDeadSessions scans active sessions and transitions dead ones to detached.
func ReapDeadSessions(s *store.Store) error {
	sessions, err := s.GetActiveSessions()
	if err != nil {
		return err
	}
	for _, sess := range sessions {
		if sess.ClaudePID > 0 && !CheckPID(sess.ClaudePID) {
			sess.Status = "detached"
			oldPID := sess.ClaudePID
			sess.ClaudePID = 0
			sess.AccessedAt = time.Now().UTC()
			if err := s.UpdateSession(&sess); err != nil {
				return err
			}
			s.AppendHistory(sess.Name, "detached", fmt.Sprintf("pid=%d (process died)", oldPID))
		}
	}
	return nil
}
