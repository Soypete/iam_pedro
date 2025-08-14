// package twitchchat is the implemtation of the chatter interface for twitch chat.
package twitchchat

import (
	"fmt"

	"github.com/Soypete/twitch-llm-bot/llms"
	"github.com/Soypete/twitch-llm-bot/llms/llamacpp"
	"github.com/Soypete/twitch-llm-bot/logging"
)

// Client is a client for interacting with the LLM and the database.
type Client struct {
	llm         llms.Model
	chatHistory []llms.MessageContent
	logger      *logging.Logger
}

// Setup creates a new twitch chat bot.
func Setup(llmPath string, logger *logging.Logger) (*Client, error) {
	if logger == nil {
		logger = logging.Default()
	}

	logger.Info("setting up twitch chat LLM client", "path", llmPath)

	llm, err := llamacpp.New(llmPath)
	if err != nil {
		logger.Error("failed to create LlamaCpp client", "error", err.Error())
		return nil, fmt.Errorf("failed to create LlamaCpp client: %w", err)
	}

	return &Client{
		llm:    llm,
		logger: logger,
	}, nil
}
