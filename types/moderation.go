package types

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ModAction represents a moderation action taken by the bot
type ModAction struct {
	ID                    uuid.UUID       `db:"id"`
	CreatedAt             time.Time       `db:"created_at"`
	TriggerMessageID      string          `db:"trigger_message_id"`
	TriggerUsername       string          `db:"trigger_username"`
	TriggerMessageContent string          `db:"trigger_message_content"`
	LLMModel              string          `db:"llm_model"`
	LLMReasoning          string          `db:"llm_reasoning"`
	ToolCallName          string          `db:"tool_call_name"`
	ToolCallParams        json.RawMessage `db:"tool_call_params"`
	TargetUsername        string          `db:"target_username"`
	TargetUserID          string          `db:"target_user_id"`
	TwitchAPIResponse     json.RawMessage `db:"twitch_api_response"`
	Success               bool            `db:"success"`
	ErrorMessage          string          `db:"error_message"`
	ChannelID             string          `db:"channel_id"`
	ChannelName           string          `db:"channel_name"`
}

// ModerationContext contains context for the LLM to make moderation decisions
type ModerationContext struct {
	Message        TwitchMessage
	MessageID      string
	RecentMessages []TwitchMessage
	ChannelRules   []string
	ChannelID      string
	ChannelName    string
}

// ModerationDecision represents the LLM's decision on whether to moderate
type ModerationDecision struct {
	ShouldAct    bool
	ToolCall     string
	ToolParams   map[string]interface{}
	Reasoning    string
	TargetUserID string
}

// TimeoutUserParams represents parameters for timeout_user tool
type TimeoutUserParams struct {
	Username        string `json:"username"`
	DurationSeconds int    `json:"duration_seconds"`
	Reason          string `json:"reason"`
}

// BanUserParams represents parameters for ban_user tool
type BanUserParams struct {
	Username string `json:"username"`
	Reason   string `json:"reason"`
}

// UnbanUserParams represents parameters for unban_user tool
type UnbanUserParams struct {
	Username string `json:"username"`
}

// ModeratorParams represents parameters for add_moderator/remove_moderator tools
type ModeratorParams struct {
	Username string `json:"username"`
}

// VIPParams represents parameters for add_vip/remove_vip tools
type VIPParams struct {
	Username string `json:"username"`
}

// DeleteMessageParams represents parameters for delete_message tool
type DeleteMessageParams struct {
	MessageID string `json:"message_id"`
}

// ChatModeParams represents parameters for chat mode tools
type ChatModeParams struct {
	Enabled         bool `json:"enabled"`
	DurationMinutes int  `json:"duration_minutes,omitempty"`
	DelaySeconds    int  `json:"delay_seconds,omitempty"`
}

// PollParams represents parameters for create_poll tool
type PollParams struct {
	Title           string   `json:"title"`
	Choices         []string `json:"choices"`
	DurationSeconds int      `json:"duration_seconds"`
}

// EndPollParams represents parameters for end_poll tool
type EndPollParams struct {
	PollID string `json:"poll_id"`
	Status string `json:"status"` // "archived" or "terminated"
}

// PredictionParams represents parameters for create_prediction tool
type PredictionParams struct {
	Title           string   `json:"title"`
	Outcomes        []string `json:"outcomes"`
	DurationSeconds int      `json:"duration_seconds"`
}

// ResolvePredictionParams represents parameters for resolve_prediction tool
type ResolvePredictionParams struct {
	PredictionID     string `json:"prediction_id"`
	WinningOutcomeID string `json:"winning_outcome_id"`
}

// CancelPredictionParams represents parameters for cancel_prediction tool
type CancelPredictionParams struct {
	PredictionID string `json:"prediction_id"`
}

// AnnouncementParams represents parameters for send_announcement tool
type AnnouncementParams struct {
	Message string `json:"message"`
	Color   string `json:"color"` // blue, green, orange, purple, primary
}

// ShoutoutParams represents parameters for shoutout tool
type ShoutoutParams struct {
	Username string `json:"username"`
}

// NoActionParams represents parameters for no_action tool
type NoActionParams struct {
	Reason string `json:"reason"`
}

// WarnUserParams represents parameters for warn_user tool (chat message warning)
type WarnUserParams struct {
	Username string `json:"username"`
	Message  string `json:"message"`
}
