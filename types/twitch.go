package types

import (
	"time"

	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"
)

// TwitchMessage represents a message sent in Twitch chat. Contains all the Metadata
// need to related information to context and the llm  calls.
type TwitchMessage struct {
	Username   string             `db:"username"`
	Text       string             `db:"message"`
	IsCommand  bool               `db:"isCommand"`
	StopReason string             `db:"stop_reason"`
	Time       time.Time          `db:"created_at"`
	UUID       uuid.UUID          `db:"uuid"`
	WebSearch  *WebSearchRequest  `db:"-"` // Not stored in database
}

// WebSearchRequest contains information needed for async web search
type WebSearchRequest struct {
	Query       string
	OriginalMsg TwitchMessage
	ChatHistory []llms.MessageContent
}
