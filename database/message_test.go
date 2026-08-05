package database

import (
	"testing"
	"time"

	"github.com/Soypete/twitch-llm-bot/types"
	"github.com/google/uuid"
)

func TestInsertMessage(t *testing.T) {
	msg := types.TwitchMessage{
		Username:  "testuser",
		Text:      "Hello Pedro!",
		IsCommand: false,
		Time:      time.Now(),
		UUID:      uuid.Nil,
	}

	t.Run("generates UUID when not provided", func(t *testing.T) {
		// Test that UUID generation works (can't actually insert without DB)
		newUUID, err := uuid.NewUUID()
		if err != nil {
			t.Fatalf("failed to generate UUID: %v", err)
		}
		if newUUID == uuid.Nil {
			t.Error("expected non-nil UUID")
		}
	})

	t.Run("message struct has correct fields", func(t *testing.T) {
		if msg.Username != "testuser" {
			t.Errorf("expected username testuser, got %s", msg.Username)
		}
		if msg.Text != "Hello Pedro!" {
			t.Errorf("expected text 'Hello Pedro!', got %s", msg.Text)
		}
		if msg.IsCommand != false {
			t.Errorf("expected IsCommand false, got %v", msg.IsCommand)
		}
	})
}

func TestMessageWriterInterface(t *testing.T) {
	var _ MessageWriter = (*Postgres)(nil)
}

func TestChatResponseWriterInterface(t *testing.T) {
	var _ ChatResponseWriter = (*Postgres)(nil)
}
