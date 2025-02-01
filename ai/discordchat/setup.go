// package discordchat is the implemtation of the chatter interface for discord.
package discordchat

import (
	"fmt"

	"github.com/Soypete/twitch-llm-bot/database"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// Bot is a client for interacting with the OpenAI LLM and the database.
type Bot struct {
	llm       llms.Model
	db        database.ResponseWriter
	modelName string
}

// Setup creates a new discord chat bot.
func Setup(db database.Postgres, modelName string, llmPath string) (*Bot, error) {
	opts := []openai.Option{
		openai.WithBaseURL(llmPath),
	}
	llm, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI LLM: %w", err)
	}

	return &Bot{
		llm:       llm,
		db:        &db,
		modelName: modelName,
	}, nil
}
