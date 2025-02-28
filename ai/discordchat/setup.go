// package discordchat is the implemtation of the chatter interface for discord.
package discordchat

import (
	"fmt"

	"github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/logging"
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

// Setup creates a new discord chat bot.
func Setup(db database.ResponseWriter, modelName string, llmPath string, logger *logging.Logger) (*Bot, error) {
	if logger == nil {
		logger = logging.Default()
	}
	
	logger.Info("setting up discord chat LLM bot", "model", modelName, "path", llmPath)
	
	opts := []openai.Option{
		openai.WithBaseURL(llmPath),
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
