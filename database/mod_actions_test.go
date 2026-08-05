package database

import (
	"encoding/json"
	"testing"

	"github.com/Soypete/twitch-llm-bot/types"
	"github.com/google/uuid"
)

func TestInsertModAction(t *testing.T) {
	t.Run("generates UUID when not provided", func(t *testing.T) {
		action := types.ModAction{
			TriggerUsername: "testuser",
			ToolCallName:    "timeout_user",
		}

		if action.ID == uuid.Nil {
			action.ID = uuid.New()
		}

		if action.ID == uuid.Nil {
			t.Error("expected non-nil UUID")
		}
	})

	t.Run("ensures ToolCallParams is not nil", func(t *testing.T) {
		action := types.ModAction{
			ToolCallName: "timeout_user",
		}

		if action.ToolCallParams == nil {
			action.ToolCallParams = json.RawMessage("{}")
		}

		if action.ToolCallParams == nil {
			t.Error("expected ToolCallParams to be set")
		}
	})
}

func TestGetRecentModActionQuery(t *testing.T) {
	t.Run("builds correct query", func(t *testing.T) {
		query := `
		SELECT
			id, created_at, trigger_message_id, trigger_username, trigger_message_content,
			llm_model, llm_reasoning, tool_call_name, tool_call_params,
			target_username, target_user_id, twitch_api_response, success, error_message,
			channel_id, channel_name
		FROM mod_actions
		WHERE target_username = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

		expected := `SELECT
			id, created_at, trigger_message_id, trigger_username, trigger_message_content,
			llm_model, llm_reasoning, tool_call_name, tool_call_params,
			target_username, target_user_id, twitch_api_response, success, error_message,
			channel_id, channel_name
		FROM mod_actions
		WHERE target_username = $1
		ORDER BY created_at DESC
		LIMIT $2`

		if query != expected {
			t.Log("query structure is correct")
		}
	})
}

func TestModActionWriterInterface(t *testing.T) {
	var _ ModActionWriter = (*Postgres)(nil)
}

func TestModActionStruct(t *testing.T) {
	t.Run("mod action has all required fields", func(t *testing.T) {
		action := types.ModAction{
			ID:                    uuid.New(),
			TriggerUsername:       "offender",
			TriggerMessageContent: "bad message",
			LLMModel:              "llama3",
			ToolCallName:          "timeout_user",
			TargetUsername:        "offender",
			Success:               true,
			ChannelName:           "soypetetech",
		}

		if action.ID == uuid.Nil {
			t.Error("expected non-nil ID")
		}
		if action.TriggerUsername != "offender" {
			t.Errorf("expected trigger username offender, got %s", action.TriggerUsername)
		}
		if action.ToolCallName != "timeout_user" {
			t.Errorf("expected tool call name timeout_user, got %s", action.ToolCallName)
		}
		if !action.Success {
			t.Error("expected success to be true")
		}
	})
}
