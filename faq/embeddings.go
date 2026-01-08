package faq

import (
	"context"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
)

// EmbeddingDimension is the dimension of OpenAI text-embedding-3-small vectors
const EmbeddingDimension = 1536

// EmbeddingService generates embeddings for text using an OpenAI-compatible API
type EmbeddingService struct {
	embedder  *embeddings.EmbedderImpl
	modelName string
}

// NewEmbeddingService creates a new embedding service
// llmPath is the base URL for the OpenAI-compatible API (e.g., http://localhost:8080)
// modelName is the embedding model to use (e.g., text-embedding-3-small)
func NewEmbeddingService(llmPath string, modelName string) (*EmbeddingService, error) {
	if llmPath == "" {
		return nil, fmt.Errorf("llmPath cannot be empty")
	}
	if modelName == "" {
		return nil, fmt.Errorf("modelName cannot be empty")
	}

	// Ensure the path ends with /v1 for OpenAI-compatible API
	if !strings.HasSuffix(llmPath, "/v1") {
		llmPath = llmPath + "/v1"
	}

	opts := []openai.Option{
		openai.WithBaseURL(llmPath),
		openai.WithEmbeddingModel(modelName),
	}

	llm, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client for embeddings: %w", err)
	}

	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	return &EmbeddingService{
		embedder:  embedder,
		modelName: modelName,
	}, nil
}

// Generate creates an embedding vector for the given text
func (s *EmbeddingService) Generate(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	vectors, err := s.embedder.EmbedDocuments(ctx, []string{text})
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if len(vectors) == 0 {
		return nil, fmt.Errorf("no embedding returned from model")
	}

	return vectors[0], nil
}

// GenerateBatch creates embedding vectors for multiple texts
// This is more efficient than calling Generate multiple times
func (s *EmbeddingService) GenerateBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("texts cannot be empty")
	}

	vectors, err := s.embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings batch: %w", err)
	}

	if len(vectors) != len(texts) {
		return nil, fmt.Errorf("embedding count mismatch: expected %d, got %d", len(texts), len(vectors))
	}

	return vectors, nil
}

// ModelName returns the name of the embedding model being used
func (s *EmbeddingService) ModelName() string {
	return s.modelName
}

// Dimension returns the dimension of the embedding vectors
func (s *EmbeddingService) Dimension() int {
	return EmbeddingDimension
}

// VectorToString converts an embedding vector to a PostgreSQL-compatible string format
// Format: [0.1, 0.2, 0.3, ...]
func VectorToString(vector []float32) string {
	if len(vector) == 0 {
		return "[]"
	}

	var sb strings.Builder
	sb.WriteString("[")
	for i, v := range vector {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("%f", v))
	}
	sb.WriteString("]")
	return sb.String()
}
