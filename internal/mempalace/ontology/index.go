package ontology

import (
	"context"
	"math"

	"github.com/Soypete/twitch-llm-bot/faq"
)

type Index struct {
	terms     []string
	vectors   [][]float32
	dimension int
	embedder  *faq.EmbeddingService
}

func NewIndex(embedder *faq.EmbeddingService) *Index {
	return &Index{
		embedder: embedder,
	}
}

func (i *Index) Build(ctx context.Context, terms []string) error {
	if len(terms) == 0 {
		return nil
	}

	vectors, err := i.embedder.GenerateBatch(ctx, terms)
	if err != nil {
		return err
	}

	i.terms = terms
	i.vectors = vectors
	i.dimension = len(vectors[0])

	return nil
}

func (i *Index) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	queryVec, err := i.embedder.Generate(ctx, query)
	if err != nil {
		return nil, err
	}

	type scoredTerm struct {
		term  string
		score float32
	}

	var scores []scoredTerm
	for j := range i.terms {
		sim := cosineSimilarity(queryVec, i.vectors[j])
		scores = append(scores, scoredTerm{term: i.terms[j], score: sim})
	}

	for j := 0; j < len(scores)-1; j++ {
		for k := j + 1; k < len(scores); k++ {
			if scores[k].score > scores[j].score {
				scores[j], scores[k] = scores[k], scores[j]
			}
		}
	}

	if topK > len(scores) {
		topK = len(scores)
	}

	results := make([]SearchResult, topK)
	for j := 0; j < topK; j++ {
		results[j] = SearchResult{
			Term:  scores[j].term,
			Score: float64(scores[j].score),
		}
	}

	return results, nil
}

func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float64
	var normA, normB float64

	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return float32(dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)))
}

type SearchResult struct {
	Term  string
	Score float64
}

func (i *Index) TermCount() int {
	return len(i.terms)
}
