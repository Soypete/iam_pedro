package langchain

import (
	"context"
	"log"
	"os"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/vectorstores/pgvector"
)

func (c *Client) MakeVectorStore() error {
	e, err := embeddings.NewEmbedder(c.llm)
	if err != nil {
		log.Fatal(err)
	}

	// Create a new pgvector store.
	ctx := context.Background()
	// why have two database connections?
	store, err := pgvector.New(
		ctx,
		pgvector.WithConnectionURL(os.Getenv("POSTGRES_VECTOR_URL")), // used just for retrieval
		pgvector.WithEmbedder(e),
	)
	if err != nil {
		log.Fatal(err)
	}

	c.vectorStore = store
	return nil
}
