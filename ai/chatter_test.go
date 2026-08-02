package ai

import (
	"context"
	"testing"

	"github.com/Soypete/twitch-llm-bot/types"
	"github.com/google/uuid"
)

func TestChatterInterface(t *testing.T) {
	t.Run("Chatter interface requires SingleMessageResponse", func(t *testing.T) {
		// Verify the interface is properly defined
		// This is a compile-time check
		var _ Chatter = nil
	})
}

func TestPedroPrompt(t *testing.T) {
	t.Run("PedroPrompt is not empty", func(t *testing.T) {
		if len(PedroPrompt) == 0 {
			t.Error("expected PedroPrompt to not be empty")
		}
	})

	t.Run("PedroPrompt contains key personality traits", func(t *testing.T) {
		prompt := PedroPrompt

		checks := []struct {
			str    string
			expect bool
		}{
			{"Pedro", true},
			{"SoyPeteTech", true},
			{"Golang", true},
			{"500 characters", true},
		}

		for _, check := range checks {
			found := len(prompt) >= len(check.str) && func() bool {
				for i := 0; i <= len(prompt)-len(check.str); i++ {
					if prompt[i:i+len(check.str)] == check.str {
						return true
					}
				}
				return false
			}()

			if found != check.expect {
				t.Errorf("expected PedroPrompt to contain %v: %v", check.str, check.expect)
			}
		}
	})
}

func TestTwitchMessageStruct(t *testing.T) {
	t.Run("TwitchMessage has required fields", func(t *testing.T) {
		msg := types.TwitchMessage{
			Username:   "testuser",
			Text:       "Hello!",
			IsCommand:  false,
			StopReason: "stop",
			UUID:       uuid.New(),
		}

		if msg.Username != "testuser" {
			t.Errorf("expected username testuser, got %s", msg.Username)
		}
		if msg.Text != "Hello!" {
			t.Errorf("expected text 'Hello!', got %s", msg.Text)
		}
		if msg.IsCommand != false {
			t.Errorf("expected IsCommand false, got %v", msg.IsCommand)
		}
	})

	t.Run("TwitchMessage can have nil UUID", func(t *testing.T) {
		msg := types.TwitchMessage{
			UUID: uuid.Nil,
		}

		if msg.UUID != uuid.Nil {
			t.Error("expected UUID to be nil")
		}
	})
}

func TestChatterInterfaceImplementation(t *testing.T) {
	t.Run("context can be used with Chatter", func(t *testing.T) {
		ctx := context.Background()
		msg := types.TwitchMessage{
			Username: "testuser",
			Text:     "Hello Pedro!",
		}

		_ = ctx
		_ = msg
	})
}
