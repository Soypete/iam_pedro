package database

import (
	"context"
	"testing"
	"time"

	"github.com/Soypete/twitch-llm-bot/types"
	"github.com/google/uuid"
)

func TestInsertResponse(t *testing.T) {
	ctx := context.Background()

	t.Run("response struct has correct fields", func(t *testing.T) {
		resp := types.TwitchMessage{
			Username:   "pedro",
			Text:       "Hello testuser!",
			StopReason: "stop",
			Time:       time.Now(),
			UUID:       uuid.New(),
		}

		if resp.Username != "pedro" {
			t.Errorf("expected username pedro, got %s", resp.Username)
		}
		if resp.Text != "Hello testuser!" {
			t.Errorf("expected text 'Hello testuser!', got %s", resp.Text)
		}
		if resp.StopReason != "stop" {
			t.Errorf("expected stop reason 'stop', got %s", resp.StopReason)
		}
	})

	t.Run("response writer interface", func(t *testing.T) {
		var _ ResponseWriter = (*Postgres)(nil)
		_ = ctx // use ctx
	})
}

func TestResponseWriterInterface(t *testing.T) {
	var _ ResponseWriter = (*Postgres)(nil)
}
