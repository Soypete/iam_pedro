// package twitchchat is the implemtation of the chatter interface for twitch chat.
package twitchchat

import (
	"fmt"

	"github.com/Soypete/twitch-llm-bot/database"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// Client is a client for interacting with the OpenAI LLM and the database.
type Client struct {
	llm         llms.Model
	db          database.ResponseWriter
	modelName   string
	chatHistory []llms.MessageContent
}

// Setup creates a new twitch chat bot.
func Setup(db database.Postgres, modelName string, llmPath string) (*Client, error) {
	opts := []openai.Option{
		openai.WithBaseURL(llmPath),
	}
	llm, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI LLM: %w", err)
	}

	return &Client{
		llm:       llm,
		db:        &db,
		modelName: modelName,
	}, nil
}
