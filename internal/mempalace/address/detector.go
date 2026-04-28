package address

import (
	"context"
	"math"

	"github.com/Soypete/twitch-llm-bot/internal/mempalace/ontology"
	"github.com/Soypete/twitch-llm-bot/metrics"
)

type Detector struct {
	index     *ontology.Index
	threshold float32
}

func NewDetector(embedder ontology.Embedder, threshold float32) (*Detector, error) {
	if threshold == 0 {
		threshold = 0.6
	}

	exemplars := []string{
		"hey pedro",
		"pedro can you",
		"what do you think pedro",
		"pedro what is",
		"pedro help me",
		"@pedro",
		"@Pedro",
		"hey Pedro",
		"Pedro can you help",
		"ping pedro",
	}

	index, err := ontology.NewIndex(embedder)
	if err != nil {
		return nil, err
	}

	err = index.Build(context.Background(), exemplars)
	if err != nil {
		return nil, err
	}

	return &Detector{
		index:     index,
		threshold: threshold,
	}, nil
}

func (d *Detector) IsAddressed(msg string) (bool, float32) {
	results, err := d.index.Search(context.Background(), msg, 1)
	if err != nil {
		return false, 0
	}

	if len(results) == 0 {
		return false, 0
	}

	score := float32(results[0].Score)
	metrics.MempalacePedroAddressScore.Observe(float64(score))

	return score >= d.threshold, score
}

func (d *Detector) SetThreshold(threshold float32) {
	d.threshold = threshold
}

func CosineSimilarity(a, b []float32) float32 {
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
