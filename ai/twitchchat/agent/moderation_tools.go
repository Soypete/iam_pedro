package agent

import (
	"encoding/json"
	"fmt"

	"github.com/tmc/langchaingo/llms"
)

// Moderation tool names as constants
const (
	ToolNoAction            = "no_action"
	ToolWarnUser            = "warn_user"
	ToolTimeoutUser         = "timeout_user"
	ToolBanUser             = "ban_user"
	ToolUnbanUser           = "unban_user"
	ToolAddModerator        = "add_moderator"
	ToolRemoveModerator     = "remove_moderator"
	ToolAddVIP              = "add_vip"
	ToolRemoveVIP           = "remove_vip"
	ToolDeleteMessage       = "delete_message"
	ToolClearChat           = "clear_chat"
	ToolEmoteOnlyMode       = "emote_only_mode"
	ToolSubscriberOnlyMode  = "subscriber_only_mode"
	ToolFollowerOnlyMode    = "follower_only_mode"
	ToolSlowMode            = "slow_mode"
	ToolCreatePoll          = "create_poll"
	ToolEndPoll             = "end_poll"
	ToolCreatePrediction    = "create_prediction"
	ToolResolvePrediction   = "resolve_prediction"
	ToolCancelPrediction    = "cancel_prediction"
	ToolSendAnnouncement    = "send_announcement"
	ToolShoutout            = "shoutout"
)

// GetModerationToolDefinitions returns all moderation tool definitions for the LLM
func GetModerationToolDefinitions() []llms.Tool {
	return []llms.Tool{
		getNoActionTool(),
		getWarnUserTool(),
		getTimeoutUserTool(),
		getBanUserTool(),
		getUnbanUserTool(),
		getAddModeratorTool(),
		getRemoveModeratorTool(),
		getAddVIPTool(),
		getRemoveVIPTool(),
		getDeleteMessageTool(),
		getClearChatTool(),
		getEmoteOnlyModeTool(),
		getSubscriberOnlyModeTool(),
		getFollowerOnlyModeTool(),
		getSlowModeTool(),
		getCreatePollTool(),
		getEndPollTool(),
		getCreatePredictionTool(),
		getResolvePredictionTool(),
		getCancelPredictionTool(),
		getSendAnnouncementTool(),
		getShoutoutTool(),
	}
}

// GetCoreModerationToolDefinitions returns only the core moderation tools
// (excludes polls, predictions, and announcements which require broadcaster token)
func GetCoreModerationToolDefinitions() []llms.Tool {
	return []llms.Tool{
		getNoActionTool(),
		getWarnUserTool(),
		getTimeoutUserTool(),
		getBanUserTool(),
		getUnbanUserTool(),
		getDeleteMessageTool(),
		getClearChatTool(),
		getEmoteOnlyModeTool(),
		getSubscriberOnlyModeTool(),
		getFollowerOnlyModeTool(),
		getSlowModeTool(),
	}
}

func getNoActionTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolNoAction,
			Description: "Take no moderation action. Use this when the message does not require any moderation.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"reason": map[string]any{
						"type":        "string",
						"description": "Brief explanation of why no action is needed",
					},
				},
				"required": []string{"reason"},
			},
		},
	}
}

func getWarnUserTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolWarnUser,
			Description: "Send a warning message to a user in chat. Use for minor first-time violations.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"username": map[string]any{
						"type":        "string",
						"description": "The username of the user to warn",
					},
					"message": map[string]any{
						"type":        "string",
						"description": "The warning message to send",
					},
				},
				"required": []string{"username", "message"},
			},
		},
	}
}

func getTimeoutUserTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolTimeoutUser,
			Description: "Temporarily ban a user from chat. Use for moderate violations or repeated minor violations.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"username": map[string]any{
						"type":        "string",
						"description": "The username of the user to timeout",
					},
					"duration_seconds": map[string]any{
						"type":        "integer",
						"description": "Duration of the timeout in seconds (1-1209600, max 2 weeks)",
						"minimum":     1,
						"maximum":     1209600,
					},
					"reason": map[string]any{
						"type":        "string",
						"description": "The reason for the timeout",
					},
				},
				"required": []string{"username", "duration_seconds", "reason"},
			},
		},
	}
}

func getBanUserTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolBanUser,
			Description: "Permanently ban a user from chat. Use only for severe violations like hate speech, harassment, or spam bots.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"username": map[string]any{
						"type":        "string",
						"description": "The username of the user to ban",
					},
					"reason": map[string]any{
						"type":        "string",
						"description": "The reason for the ban",
					},
				},
				"required": []string{"username", "reason"},
			},
		},
	}
}

func getUnbanUserTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolUnbanUser,
			Description: "Remove a ban or timeout from a user. Use to reverse a previous moderation action.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"username": map[string]any{
						"type":        "string",
						"description": "The username of the user to unban",
					},
				},
				"required": []string{"username"},
			},
		},
	}
}

func getAddModeratorTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolAddModerator,
			Description: "Promote a user to moderator status. Use with extreme caution.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"username": map[string]any{
						"type":        "string",
						"description": "The username of the user to promote",
					},
				},
				"required": []string{"username"},
			},
		},
	}
}

func getRemoveModeratorTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolRemoveModerator,
			Description: "Remove moderator status from a user. Use with extreme caution.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"username": map[string]any{
						"type":        "string",
						"description": "The username of the moderator to demote",
					},
				},
				"required": []string{"username"},
			},
		},
	}
}

func getAddVIPTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolAddVIP,
			Description: "Add VIP status to a user.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"username": map[string]any{
						"type":        "string",
						"description": "The username of the user to give VIP status",
					},
				},
				"required": []string{"username"},
			},
		},
	}
}

func getRemoveVIPTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolRemoveVIP,
			Description: "Remove VIP status from a user.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"username": map[string]any{
						"type":        "string",
						"description": "The username of the VIP to demote",
					},
				},
				"required": []string{"username"},
			},
		},
	}
}

func getDeleteMessageTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolDeleteMessage,
			Description: "Delete a specific message from chat. Use for messages that violate rules but don't warrant a timeout.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"message_id": map[string]any{
						"type":        "string",
						"description": "The ID of the message to delete",
					},
				},
				"required": []string{"message_id"},
			},
		},
	}
}

func getClearChatTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolClearChat,
			Description: "Clear all messages from chat. Use only in extreme situations like raids or mass spam.",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}
}

func getEmoteOnlyModeTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolEmoteOnlyMode,
			Description: "Enable or disable emote-only mode. Users can only send emotes when enabled.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"enabled": map[string]any{
						"type":        "boolean",
						"description": "Whether to enable (true) or disable (false) emote-only mode",
					},
				},
				"required": []string{"enabled"},
			},
		},
	}
}

func getSubscriberOnlyModeTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolSubscriberOnlyMode,
			Description: "Enable or disable subscriber-only mode. Only subscribers can chat when enabled.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"enabled": map[string]any{
						"type":        "boolean",
						"description": "Whether to enable (true) or disable (false) subscriber-only mode",
					},
				},
				"required": []string{"enabled"},
			},
		},
	}
}

func getFollowerOnlyModeTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolFollowerOnlyMode,
			Description: "Enable or disable follower-only mode. Only followers can chat when enabled.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"enabled": map[string]any{
						"type":        "boolean",
						"description": "Whether to enable (true) or disable (false) follower-only mode",
					},
					"duration_minutes": map[string]any{
						"type":        "integer",
						"description": "Minimum follow time in minutes (0-129600). Only required when enabling.",
						"minimum":     0,
						"maximum":     129600,
					},
				},
				"required": []string{"enabled"},
			},
		},
	}
}

func getSlowModeTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolSlowMode,
			Description: "Enable or disable slow mode. Limits how often users can send messages.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"enabled": map[string]any{
						"type":        "boolean",
						"description": "Whether to enable (true) or disable (false) slow mode",
					},
					"delay_seconds": map[string]any{
						"type":        "integer",
						"description": "Time between messages in seconds (3-120). Only required when enabling.",
						"minimum":     3,
						"maximum":     120,
					},
				},
				"required": []string{"enabled"},
			},
		},
	}
}

func getCreatePollTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolCreatePoll,
			Description: "Create a poll for viewers to vote on. Requires broadcaster token.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": map[string]any{
						"type":        "string",
						"description": "The poll question (max 60 characters)",
						"maxLength":   60,
					},
					"choices": map[string]any{
						"type":        "array",
						"description": "Poll options (2-5 choices, max 25 characters each)",
						"items": map[string]any{
							"type":      "string",
							"maxLength": 25,
						},
						"minItems": 2,
						"maxItems": 5,
					},
					"duration_seconds": map[string]any{
						"type":        "integer",
						"description": "How long the poll runs in seconds (15-1800)",
						"minimum":     15,
						"maximum":     1800,
					},
				},
				"required": []string{"title", "choices", "duration_seconds"},
			},
		},
	}
}

func getEndPollTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolEndPoll,
			Description: "End an active poll. Requires broadcaster token.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"poll_id": map[string]any{
						"type":        "string",
						"description": "The ID of the poll to end",
					},
					"status": map[string]any{
						"type":        "string",
						"description": "How to end the poll: 'terminated' shows results, 'archived' hides results",
						"enum":        []string{"terminated", "archived"},
					},
				},
				"required": []string{"poll_id", "status"},
			},
		},
	}
}

func getCreatePredictionTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolCreatePrediction,
			Description: "Create a prediction for viewers to bet channel points on. Requires broadcaster token.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": map[string]any{
						"type":        "string",
						"description": "The prediction question (max 45 characters)",
						"maxLength":   45,
					},
					"outcomes": map[string]any{
						"type":        "array",
						"description": "Possible outcomes (2-10 outcomes, max 25 characters each)",
						"items": map[string]any{
							"type":      "string",
							"maxLength": 25,
						},
						"minItems": 2,
						"maxItems": 10,
					},
					"duration_seconds": map[string]any{
						"type":        "integer",
						"description": "How long users can make predictions in seconds (30-1800)",
						"minimum":     30,
						"maximum":     1800,
					},
				},
				"required": []string{"title", "outcomes", "duration_seconds"},
			},
		},
	}
}

func getResolvePredictionTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolResolvePrediction,
			Description: "Resolve a prediction with a winning outcome. Requires broadcaster token.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"prediction_id": map[string]any{
						"type":        "string",
						"description": "The ID of the prediction to resolve",
					},
					"winning_outcome_id": map[string]any{
						"type":        "string",
						"description": "The ID of the winning outcome",
					},
				},
				"required": []string{"prediction_id", "winning_outcome_id"},
			},
		},
	}
}

func getCancelPredictionTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolCancelPrediction,
			Description: "Cancel a prediction and refund all channel points. Requires broadcaster token.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"prediction_id": map[string]any{
						"type":        "string",
						"description": "The ID of the prediction to cancel",
					},
				},
				"required": []string{"prediction_id"},
			},
		},
	}
}

func getSendAnnouncementTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolSendAnnouncement,
			Description: "Send a highlighted announcement message to chat.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"message": map[string]any{
						"type":        "string",
						"description": "The announcement message (max 500 characters)",
						"maxLength":   500,
					},
					"color": map[string]any{
						"type":        "string",
						"description": "The color of the announcement",
						"enum":        []string{"blue", "green", "orange", "purple", "primary"},
					},
				},
				"required": []string{"message"},
			},
		},
	}
}

func getShoutoutTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolShoutout,
			Description: "Give a shoutout to another streamer, showing their channel info in chat.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"username": map[string]any{
						"type":        "string",
						"description": "The username of the streamer to shoutout",
					},
				},
				"required": []string{"username"},
			},
		},
	}
}

// ModerationToolCallArgs represents generic parsed arguments from a moderation tool call
type ModerationToolCallArgs struct {
	ToolName string
	Args     map[string]interface{}
}

// ParseModerationToolCall parses a tool call and returns the tool name and arguments
func ParseModerationToolCall(toolCall llms.ToolCall) (*ModerationToolCallArgs, error) {
	var args map[string]interface{}
	if toolCall.FunctionCall.Arguments != "" {
		if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args); err != nil {
			return nil, fmt.Errorf("failed to parse tool call arguments: %w", err)
		}
	}

	return &ModerationToolCallArgs{
		ToolName: toolCall.FunctionCall.Name,
		Args:     args,
	}, nil
}

// IsModerationTool checks if a tool name is a moderation tool
func IsModerationTool(toolName string) bool {
	switch toolName {
	case ToolNoAction, ToolWarnUser, ToolTimeoutUser, ToolBanUser, ToolUnbanUser,
		ToolAddModerator, ToolRemoveModerator, ToolAddVIP, ToolRemoveVIP,
		ToolDeleteMessage, ToolClearChat, ToolEmoteOnlyMode, ToolSubscriberOnlyMode,
		ToolFollowerOnlyMode, ToolSlowMode, ToolCreatePoll, ToolEndPoll,
		ToolCreatePrediction, ToolResolvePrediction, ToolCancelPrediction,
		ToolSendAnnouncement, ToolShoutout:
		return true
	default:
		return false
	}
}
