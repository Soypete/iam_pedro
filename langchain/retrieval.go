package langchain

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/pgvector"
)

func (c *Client) MakeVectorStore() error {
	opts := []openai.Option{
		openai.WithBaseURL("http://127.0.0.1:8080"),
	}
	llm, err := openai.New(opts...)
	if err != nil {
		return fmt.Errorf("failed to create OpenAI LLM: %w", err)
	}
	e, err := embeddings.NewEmbedder(llm)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	store, err := pgvector.New(
		ctx,
		pgvector.WithConnectionURL(os.Getenv("POSTGRES_URL")), // used just for retrieval
		pgvector.WithEmbedder(e),
	)
	if err != nil {
		log.Fatal(err)
	}

	// add speakers
	f, err := os.Open("./test-langchain/speakers.csv")
	if err != nil {
		return fmt.Errorf("failed to open speakers file: %w", err)
	}
	r := csv.NewReader(f)

	records, err := r.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read speakers file: %w", err)
	}
	var speakerDocs []schema.Document
	for i, row := range records {
		// skip header
		if i == 0 {
			continue
		}
		speakerDocs = append(speakerDocs, schema.Document{PageContent: row[0] + " is speaking about " + row[2] + "."})
	}
	_, err = store.AddDocuments(ctx, speakerDocs)
	if err != nil {
		return fmt.Errorf("failed to add speaker documents to store: %w", err)
	}
	docs, err := store.SimilaritySearch(ctx, "speaking about Parducky ", 3, vectorstores.WithScoreThreshold(0.80))
	// docs, err := store.SimilaritySearch(ctx, "who is speaking at 10am?", 3, vectorstores.WithScoreThreshold(0.80), vectorstores.WithFilters(filter))
	if err != nil {
		return fmt.Errorf("failed to search for similar documents: %w", err)
	}
	fmt.Println(docs)
	return nil
}

// {
// 	PageContent: "Sponsors",
// 	Metadata: map[string]any{},
// },
// {
// 	PageContent: "Talks",
// 	Metadata: map[string]any{},
// },
// {
// 	PageContent: "Location",
// 	Metadata: map[string]any{},
// },
// {
// 	PageContent: "CodeOfConduct",
// 	Metadata: map[string]any{},
// },
// {
// 	PageContent: "Website",
// 	Mlskjfalskjdflkajetadata: map[string]any{},
// },
