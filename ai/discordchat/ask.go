package discordchat

import (
	"context"

	"github.com/Soypete/twitch-llm-bot/database"
	"github.com/google/uuid"
)

// SingleMessageResponse is a response from the LLM model to a single message
func (b *Bot) SingleMessageResponse(ctx context.Context, msg database.TwitchMessage, messageID uuid.UUID) (string, error) {
	return "", nil
}
