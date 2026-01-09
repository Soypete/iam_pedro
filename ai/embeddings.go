package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
)

// EmbeddingGenerator handles generating embeddings for text
type EmbeddingGenerator struct {
	embedder *embeddings.EmbedderImpl
}

// NewEmbeddingGenerator creates a new embedding generator
// If llmPath is empty, it will use the default OpenAI endpoint
// embeddingModel should be something like "text-embedding-ada-002" for OpenAI
// or the model name for a local embedding server
func NewEmbeddingGenerator(llmPath string, embeddingModel string) (*EmbeddingGenerator, error) {
	if embeddingModel == "" {
		embeddingModel = "text-embedding-ada-002"
	}

	var opts []openai.Option
	opts = append(opts, openai.WithModel(embeddingModel))

	if llmPath != "" {
		if !strings.HasSuffix(llmPath, "/v1") {
			llmPath = llmPath + "/v1"
		}
		opts = append(opts, openai.WithBaseURL(llmPath))
	}

	llm, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client for embeddings: %w", err)
	}

	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	return &EmbeddingGenerator{
		embedder: embedder,
	}, nil
}

// GenerateEmbedding generates an embedding vector for a single text string
func (e *EmbeddingGenerator) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	vectors, err := e.embedder.EmbedQuery(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	return vectors, nil
}

// GenerateEmbeddings generates embedding vectors for multiple text strings
func (e *EmbeddingGenerator) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("texts slice cannot be empty")
	}

	vectors, err := e.embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	return vectors, nil
}

// FormatModerationMessage formats a moderation message for embedding
// This creates a consistent format for similarity search
func FormatModerationMessage(username, message string) string {
	return fmt.Sprintf("[%s]: %s", username, message)
}
