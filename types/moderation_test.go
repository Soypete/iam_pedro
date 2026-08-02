package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestModActionStruct(t *testing.T) {
	t.Run("ModAction has required fields", func(t *testing.T) {
		action := ModAction{
			ID:                    uuid.New(),
			CreatedAt:             time.Now(),
			TriggerMessageID:      "msg-123",
			TriggerUsername:       "user1",
			TriggerMessageContent: "bad content",
			LLMModel:              "llama3",
			LLMReasoning:          "Reasoning text",
			ToolCallName:          "ban",
			ToolCallParams:        json.RawMessage(`{"user": "baduser"}`),
			TargetUsername:        "baduser",
			TargetUserID:          "uid-123",
			Success:               true,
			ChannelID:             "ch-123",
			ChannelName:           "soypetetech",
		}

		if action.TriggerMessageID != "msg-123" {
			t.Errorf("expected msg-123, got %s", action.TriggerMessageID)
		}
		if !action.Success {
			t.Error("expected Success to be true")
		}
	})
}

func TestModerationContextStruct(t *testing.T) {
	t.Run("ModerationContext has required fields", func(t *testing.T) {
		ctx := ModerationContext{
			Message: TwitchMessage{
				Username: "user1",
				Text:     "test message",
			},
			MessageID:      "msg-123",
			RecentMessages: []TwitchMessage{},
			ChannelRules:   []string{"Be nice", "No spam"},
			ChannelID:      "ch-123",
			ChannelName:    "soypetetech",
		}

		if ctx.MessageID != "msg-123" {
			t.Errorf("expected msg-123, got %s", ctx.MessageID)
		}
		if len(ctx.ChannelRules) != 2 {
			t.Errorf("expected 2 rules, got %d", len(ctx.ChannelRules))
		}
	})
}
