// package discordchat is the implemtation of the chatter interface for discord.
package discordchat

import (
	"context"
	"fmt"

	"github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/Soypete/twitch-llm-bot/types"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// Bot is a client for interacting with the OpenAI LLM and the database.
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

	// Ensure the path ends with /v1 for OpenAI-compatible API
	if llmPath != "" && llmPath[len(llmPath)-3:] != "/v1" {
		llmPath = llmPath + "/v1"
		logger.Info("appended /v1 to LLM path", "fullPath", llmPath)
	}

	opts := []openai.Option{
		openai.WithBaseURL(llmPath),
		openai.WithModel(modelName),
	}
	llm, err := openai.New(opts...)
	if err != nil {
		logger.Error("failed to create OpenAI LLM", "error", err.Error())
		return nil, fmt.Errorf("failed to create OpenAI LLM: %w", err)
	}

	return &Bot{
		llm:       llm,
		db:        db,
		modelName: modelName,
		logger:    logger,
	}, nil
}
