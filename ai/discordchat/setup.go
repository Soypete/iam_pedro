// package discordchat is the implemtation of the chatter interface for discord.
package discordchat

import (
	"context"
	"fmt"

	"github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/llms"
	"github.com/Soypete/twitch-llm-bot/llms/llamacpp"
	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/Soypete/twitch-llm-bot/types"
)

// Bot is a client for interacting with the HTTP LLM and the database.
type Bot struct {
	llm       llms.Model
	db        database.ResponseWriter
	modelName string
	logger    *logging.Logger
}

// LLM is an interface that defines the methods for interacting with the LLM from discord.
type LLM interface {
	// ai.Chatter
	Start20Questions(context.Context, types.Discord20QuestionsGame) (string, error)
	Play20Questions(context.Context, string, []llms.MessageContent) (string, error)
	SingleMessageResponse(context.Context, types.DiscordAskMessage) (string, error)
}

// Setup creates a new discord chat bot.
func Setup(db database.ResponseWriter, modelName string, llmPath string, logger *logging.Logger) (*Bot, error) {
	if logger == nil {
		logger = logging.Default()
	}

	logger.Info("setting up discord chat LLM bot", "model", modelName, "path", llmPath)

	llm, err := llamacpp.New(llmPath, llamacpp.WithModel(modelName))
	if err != nil {
		logger.Error("failed to create LlamaCpp client", "error", err.Error())
		return nil, fmt.Errorf("failed to create LlamaCpp client: %w", err)
	}

	return &Bot{
		llm:       llm,
		db:        db,
		modelName: modelName,
		logger:    logger,
	}, nil
}
