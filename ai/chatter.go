// package ai defines the interface that Pedro will have to implement the functions that will be used in the chat.

package ai

import (
	"context"

	"github.com/Soypete/twitch-llm-bot/database"
	"github.com/google/uuid"
)

// Chattter is the interface that defines the functions that Pedro will have. The interface is implemented with functionally for each connection.
type Chatter interface {
	SingleMessageResponse(ctx context.Context, msg database.TwitchMessage, messageID uuid.UUID) (string, error)
	Play20Questions(ctx context.Context, msg database.TwitchMessage, messageID uuid.UUID) (string, error)
	End20Questions()
}
