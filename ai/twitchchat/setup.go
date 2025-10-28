// package twitchchat is the implemtation of the chatter interface for twitch chat.
package twitchchat

import (
	"fmt"

	"github.com/Soypete/twitch-llm-bot/duckduckgo"
	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// Client is a client for interacting with the OpenAI LLM and the database.
type Client struct {
	llm         llms.Model
	chatHistory []llms.MessageContent
	modelName   string
	logger      *logging.Logger
	ddgClient   *duckduckgo.Client
}

// Setup creates a new twitch chat bot.
func Setup(llmPath string, modelName string, logger *logging.Logger) (*Client, error) {
	if logger == nil {
		logger = logging.Default()
	}

	logger.Info("setting up twitch chat LLM client", "path", llmPath)

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

	// Initialize DuckDuckGo client
	ddgClient := duckduckgo.NewClient()

	return &Client{
		llm:       llm,
		modelName: modelName,
		logger:    logger,
		ddgClient: ddgClient,
	}, nil
}
