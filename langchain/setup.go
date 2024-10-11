package langchain

import (
	"context"
	"fmt"

	"github.com/Soypete/twitch-llm-bot/database"
	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/vectorstores"
)

type Inferencer interface {
	SingleMessageResponse(ctx context.Context, msg database.TwitchMessage, messageID uuid.UUID) (string, error)
	GenerateTimer(ctx context.Context) (string, error)
}

type Client struct {
	llm         llms.Model
	db          database.ResponseWriter
	modelName   string
	vectorStore vectorstores.VectorStore
}

func Setup(db database.Postgres, modelName string) (*Client, error) {
	opts := []openai.Option{
		openai.WithBaseURL("http://127.0.0.1:8080"),
	}
	llm, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI LLM: %w", err)
	}

	c := &Client{
		llm:       llm,
		db:        &db,
		modelName: modelName,
	}

	err = c.MakeVectorStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create vector store: %w", err)
	}
	return c, nil
}
