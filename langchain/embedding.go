package langchain

import (
	"context"
	"fmt"

	"github.com/pgvector/pgvector-go"
)

// CreateEmbedding creates an embedding from the string prompt. The embedding from langchain is a 2D array of float32.
func (c Client) CreateEmbedding(ctx context.Context, injection []string) ([]pgvector.Vector, error) {

	vectors, err := c.embedder.EmbedDocuments(ctx, injection)
	if err != nil {
		return nil, fmt.Errorf("failed to embed documents: %w", err)
	}

	var embeddings []pgvector.Vector // TODO: maybe we make this an array of length len(vectors) for performance
	for i := range vectors {
		embeddings = append(embeddings, pgvector.NewVector(vectors[i]))
	}

	return embeddings, nil
}
