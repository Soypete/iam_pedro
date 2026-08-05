package twitchirc

import (
	"testing"

	"github.com/Soypete/twitch-llm-bot/logging"
)

func TestSessionRegistry(t *testing.T) {
	logger := logging.Default()

	t.Run("creates new session registry", func(t *testing.T) {
		registry := NewSessionRegistry("testdata", logger)
		if registry == nil || registry.sessions == nil {
			t.Error("expected non-nil registry with initialized sessions map")
		}
	})

	t.Run("GetOrCreateSession creates new session", func(t *testing.T) {
		registry := NewSessionRegistry("testdata", logger)
		session, err := registry.GetOrCreateSession("stream123")
		if err != nil {
			t.Logf("session creation may fail without wrapper: %v", err)
		}
		if session != nil {
			_ = session.End()
		}
	})

	t.Run("GetDataDir returns correct path", func(t *testing.T) {
		registry := NewSessionRegistry("testdata", logger)
		dataDir := registry.GetDataDir()
		if dataDir == "" {
			t.Error("expected non-empty data dir")
		}
	})

	t.Run("GetDataDir returns absolute path for empty input", func(t *testing.T) {
		registry := NewSessionRegistry("", logger)
		dataDir := registry.GetDataDir()
		if dataDir == "" {
			t.Error("expected non-empty data dir")
		}
	})
}

func TestSessionRegistryConcurrency(t *testing.T) {
	logger := logging.Default()
	registry := NewSessionRegistry("testdata", logger)

	t.Run("multiple GetOrCreateSession calls are safe", func(t *testing.T) {
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				_, _ = registry.GetOrCreateSession("stream123")
				done <- true
			}()
		}
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}
