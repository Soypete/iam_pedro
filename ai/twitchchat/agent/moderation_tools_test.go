package agent

import (
	"testing"

	"github.com/tmc/langchaingo/llms"
)

func TestGetModerationToolDefinitions(t *testing.T) {
	tools := GetModerationToolDefinitions()

	if len(tools) == 0 {
		t.Error("expected at least one moderation tool")
	}

	// Check that all tools have required fields
	for _, tool := range tools {
		if tool.Type != "function" {
			t.Errorf("tool type should be 'function', got %s", tool.Type)
		}
		if tool.Function == nil {
			t.Error("tool function should not be nil")
		}
		if tool.Function.Name == "" {
			t.Error("tool function name should not be empty")
		}
		if tool.Function.Description == "" {
			t.Errorf("tool %s should have a description", tool.Function.Name)
		}
	}
}

func TestGetCoreModerationToolDefinitions(t *testing.T) {
	tools := GetCoreModerationToolDefinitions()

	if len(tools) == 0 {
		t.Error("expected at least one core moderation tool")
	}

	// Core tools should not include broadcaster-only tools
	broadcasterTools := map[string]bool{
		ToolCreatePoll:        true,
		ToolEndPoll:           true,
		ToolCreatePrediction:  true,
		ToolResolvePrediction: true,
		ToolCancelPrediction:  true,
	}

	for _, tool := range tools {
		if broadcasterTools[tool.Function.Name] {
			t.Errorf("core tools should not include broadcaster-only tool: %s", tool.Function.Name)
		}
	}
}

func TestParseModerationToolCall(t *testing.T) {
	tests := []struct {
		name     string
		toolCall llms.ToolCall
		wantName string
		wantArgs map[string]interface{}
		wantErr  bool
	}{
		{
			name: "timeout_user tool call",
			toolCall: llms.ToolCall{
				FunctionCall: &llms.FunctionCall{
					Name:      "timeout_user",
					Arguments: `{"username": "testuser", "duration_seconds": 300, "reason": "spam"}`,
				},
			},
			wantName: "timeout_user",
			wantArgs: map[string]interface{}{
				"username":         "testuser",
				"duration_seconds": float64(300),
				"reason":           "spam",
			},
			wantErr: false,
		},
		{
			name: "no_action tool call",
			toolCall: llms.ToolCall{
				FunctionCall: &llms.FunctionCall{
					Name:      "no_action",
					Arguments: `{"reason": "message is fine"}`,
				},
			},
			wantName: "no_action",
			wantArgs: map[string]interface{}{
				"reason": "message is fine",
			},
			wantErr: false,
		},
		{
			name: "ban_user tool call",
			toolCall: llms.ToolCall{
				FunctionCall: &llms.FunctionCall{
					Name:      "ban_user",
					Arguments: `{"username": "baduser", "reason": "hate speech"}`,
				},
			},
			wantName: "ban_user",
			wantArgs: map[string]interface{}{
				"username": "baduser",
				"reason":   "hate speech",
			},
			wantErr: false,
		},
		{
			name: "empty arguments",
			toolCall: llms.ToolCall{
				FunctionCall: &llms.FunctionCall{
					Name:      "clear_chat",
					Arguments: "",
				},
			},
			wantName: "clear_chat",
			wantArgs: nil,
			wantErr:  false,
		},
		{
			name: "invalid json",
			toolCall: llms.ToolCall{
				FunctionCall: &llms.FunctionCall{
					Name:      "timeout_user",
					Arguments: `{invalid json}`,
				},
			},
			wantName: "",
			wantArgs: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseModerationToolCall(tt.toolCall)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseModerationToolCall() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if result.ToolName != tt.wantName {
				t.Errorf("ToolName = %v, want %v", result.ToolName, tt.wantName)
			}

			if tt.wantArgs != nil {
				for key, wantVal := range tt.wantArgs {
					gotVal, ok := result.Args[key]
					if !ok {
						t.Errorf("missing arg %s", key)
						continue
					}
					if gotVal != wantVal {
						t.Errorf("arg %s = %v, want %v", key, gotVal, wantVal)
					}
				}
			}
		})
	}
}

func TestIsModerationTool(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		want     bool
	}{
		{"no_action is moderation tool", ToolNoAction, true},
		{"timeout_user is moderation tool", ToolTimeoutUser, true},
		{"ban_user is moderation tool", ToolBanUser, true},
		{"delete_message is moderation tool", ToolDeleteMessage, true},
		{"clear_chat is moderation tool", ToolClearChat, true},
		{"create_poll is moderation tool", ToolCreatePoll, true},
		{"shoutout is moderation tool", ToolShoutout, true},
		{"web_search is not moderation tool", "web_search", false},
		{"random is not moderation tool", "random_tool", false},
		{"empty is not moderation tool", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsModerationTool(tt.toolName); got != tt.want {
				t.Errorf("IsModerationTool(%q) = %v, want %v", tt.toolName, got, tt.want)
			}
		})
	}
}

func TestToolConstants(t *testing.T) {
	// Verify tool constants match their expected values
	expectedTools := map[string]string{
		"ToolNoAction":           "no_action",
		"ToolWarnUser":           "warn_user",
		"ToolTimeoutUser":        "timeout_user",
		"ToolBanUser":            "ban_user",
		"ToolUnbanUser":          "unban_user",
		"ToolDeleteMessage":      "delete_message",
		"ToolClearChat":          "clear_chat",
		"ToolEmoteOnlyMode":      "emote_only_mode",
		"ToolSubscriberOnlyMode": "subscriber_only_mode",
		"ToolFollowerOnlyMode":   "follower_only_mode",
		"ToolSlowMode":           "slow_mode",
	}

	actualTools := map[string]string{
		"ToolNoAction":           ToolNoAction,
		"ToolWarnUser":           ToolWarnUser,
		"ToolTimeoutUser":        ToolTimeoutUser,
		"ToolBanUser":            ToolBanUser,
		"ToolUnbanUser":          ToolUnbanUser,
		"ToolDeleteMessage":      ToolDeleteMessage,
		"ToolClearChat":          ToolClearChat,
		"ToolEmoteOnlyMode":      ToolEmoteOnlyMode,
		"ToolSubscriberOnlyMode": ToolSubscriberOnlyMode,
		"ToolFollowerOnlyMode":   ToolFollowerOnlyMode,
		"ToolSlowMode":           ToolSlowMode,
	}

	for name, expected := range expectedTools {
		if actual, ok := actualTools[name]; !ok || actual != expected {
			t.Errorf("%s = %q, want %q", name, actual, expected)
		}
	}
}
