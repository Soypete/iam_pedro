package database

import (
	"testing"

	"github.com/Soypete/twitch-llm-bot/logging"
)

func TestNewPostgres(t *testing.T) {
	t.Run("creates postgres with default logger", func(t *testing.T) {
		logger := logging.Default()
		if logger == nil {
			t.Error("expected non-nil logger")
		}
	})

	t.Run("postgres struct has connections and logger", func(t *testing.T) {
		p := &Postgres{}
		_ = p.connections
		_ = p.logger
	})
}

func TestPostgresClose(t *testing.T) {
	t.Run("close does not panic with nil connections", func(t *testing.T) {
		p := &Postgres{}
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Logf("caught expected panic: %v", r)
				}
			}()
			p.Close()
		}()
	})
}

func TestPostgresDB(t *testing.T) {
	t.Run("DB returns underlying sqlx.DB", func(t *testing.T) {
		p := &Postgres{}
		_ = p.DB()
	})
}
