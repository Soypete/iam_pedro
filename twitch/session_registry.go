package twitchirc

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Soypete/twitch-llm-bot/logging"
)

type SessionRegistry struct {
	sessions map[string]*PalaceSession
	mu       sync.RWMutex
	dataDir  string
	logger   *logging.Logger
}

func NewSessionRegistry(dataDir string, logger *logging.Logger) *SessionRegistry {
	return &SessionRegistry{
		sessions: make(map[string]*PalaceSession),
		dataDir:  dataDir,
		logger:   logger,
	}
}

func (r *SessionRegistry) GetOrCreateSession(streamID string) (*PalaceSession, error) {
	r.mu.RLock()
	session, exists := r.sessions[streamID]
	r.mu.RUnlock()

	if exists {
		return session, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if session, exists = r.sessions[streamID]; exists {
		return session, nil
	}

	sessionID := r.generateSessionID(streamID)
	session, err := NewPalaceSession(sessionID, r.dataDir, r.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create palace session: %w", err)
	}

	r.sessions[streamID] = session
	r.logger.Info("created new palace session", "streamID", streamID, "sessionID", sessionID)

	return session, nil
}

func (r *SessionRegistry) generateSessionID(streamID string) string {
	now := time.Now()
	dateStr := now.Format("2006-01-02")
	channel := strings.ToLower(streamID)
	return fmt.Sprintf("%s_%s", dateStr, channel)
}

func (r *SessionRegistry) EndSession(streamID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.sessions[streamID]
	if !exists {
		return nil
	}

	if err := session.End(); err != nil {
		r.logger.Error("failed to end palace session", "streamID", streamID, "error", err.Error())
		return err
	}

	delete(r.sessions, streamID)
	r.logger.Info("ended palace session", "streamID", streamID)

	return nil
}

func (r *SessionRegistry) EndAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for streamID, session := range r.sessions {
		if err := session.End(); err != nil {
			r.logger.Error("failed to end palace session", "streamID", streamID, "error", err.Error())
		}
	}

	r.sessions = make(map[string]*PalaceSession)
	return nil
}

func (r *SessionRegistry) GetDataDir() string {
	if r.dataDir == "" {
		home, _ := filepath.Abs("data")
		return home
	}
	return r.dataDir
}
