// package discordchat is the implemtation of the chatter interface for discord.
package discordchat

import (
	"context"
	"fmt"

	"github.com/Soypete/twitch-llm-bot/ai/agent"
	"github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/duckduckgo"
	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/Soypete/twitch-llm-bot/types"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/tools"
)

// Bot is a client for interacting with the OpenAI LLM and the database.
type Bot struct {
	llm         llms.Model
	agent       tools.Tool
	db          database.ResponseWriter
	modelName   string
	logger      *logging.Logger
	ddgClient   *duckduckgo.Client
	chatHistory []llms.MessageContent
}

// LLM is an interface that defines the methods for interacting with the LLM from discord.
type LLM interface {
	// ai.Chatter
	Start20Questions(context.Context, types.Discord20QuestionsGame) (string, error)
	Play20Questions(context.Context, string, []llms.MessageContent) (string, error)
	SingleMessageResponse(context.Context, types.DiscordAskMessage) (*types.DiscordResponse, error)
	ThreadMessageResponse(context.Context, types.DiscordAskMessage, []llms.MessageContent) (string, error)
	ExecuteWebSearch(context.Context, *types.WebSearchRequest) (string, error)
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

	// Initialize DuckDuckGo client
	ddgClient := duckduckgo.NewClient()

	// Create web search agent using shared agent package
	webSearchAgent := agent.CreateWebSearchAgent(ddgClient)

	return &Bot{
		llm:         llm,
		agent:       webSearchAgent,
		db:          db,
		modelName:   modelName,
		logger:      logger,
		ddgClient:   ddgClient,
		chatHistory: []llms.MessageContent{},
	}, nil
}
