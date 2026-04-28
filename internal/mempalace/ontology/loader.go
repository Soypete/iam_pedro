package ontology

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

type Embedder interface {
	Generate(ctx context.Context, text string) ([]float32, error)
	GenerateBatch(ctx context.Context, texts []string) ([][]float32, error)
}

type Class struct {
	URI        string
	Label      string
	AltLabels  []string
	Definition string
	SubClassOf string
}

type Loader struct {
	classes map[string]Class
}

func NewLoader() *Loader {
	return &Loader{
		classes: make(map[string]Class),
	}
}

func (l *Loader) LoadTTL(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open ontology file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentURI string
	var currentClass Class

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.Contains(line, "a skos:Concept") || strings.Contains(line, "a owl:Class") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				currentURI = strings.TrimSuffix(parts[0], " a")
				currentClass = Class{URI: currentURI}
			}
			continue
		}

		if currentURI != "" && strings.Contains(line, "skos:prefLabel") {
			label := l.extractLabel(line)
			currentClass.Label = label
		}

		if currentURI != "" && strings.Contains(line, "skos:altLabel") {
			altLabel := l.extractLabel(line)
			if altLabel != "" {
				currentClass.AltLabels = append(currentClass.AltLabels, altLabel)
			}
		}

		if currentURI != "" && strings.Contains(line, "skos:definition") {
			def := l.extractLabel(line)
			currentClass.Definition = def
		}

		if currentURI != "" && strings.Contains(line, "rdfs:subClassOf") {
			subClassOf := l.extractURI(line)
			currentClass.SubClassOf = subClassOf
		}

		if currentURI != "" && (line == "." || strings.HasSuffix(line, " .")) {
			if currentClass.Label != "" {
				l.classes[currentURI] = currentClass
			}
			currentURI = ""
		}
	}

	return scanner.Err()
}

func (l *Loader) extractLabel(line string) string {
	start := strings.Index(line, "\"")
	if start == -1 {
		return ""
	}
	end := strings.Index(line[start+1:], "\"")
	if end == -1 {
		return ""
	}
	return line[start+1 : start+1+end]
}

func (l *Loader) extractURI(line string) string {
	parts := strings.Fields(line)
	for _, p := range parts {
		if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") || strings.HasPrefix(p, "sw:") {
			return strings.TrimSuffix(p, " .")
		}
	}
	return ""
}

func (l *Loader) GetClasses() []Class {
	classes := make([]Class, 0, len(l.classes))
	for _, c := range l.classes {
		classes = append(classes, c)
	}
	return classes
}

func (l *Loader) GetSearchTerms() []string {
	terms := make([]string, 0)
	seen := make(map[string]bool)
	for _, c := range l.classes {
		if c.Label != "" && !seen[c.Label] {
			terms = append(terms, c.Label)
			seen[c.Label] = true
		}
		for _, alt := range c.AltLabels {
			if !seen[alt] {
				terms = append(terms, alt)
				seen[alt] = true
			}
		}
	}
	return terms
}

func (l *Loader) GetClassByLabel(label string) *Class {
	for _, c := range l.classes {
		if c.Label == label {
			return &c
		}
	}
	return nil
}

type Index struct {
	terms     []string
	vectors   [][]float32
	dimension int
	embedder  Embedder
	classes   map[string]Class
}

func NewIndex(embedder Embedder) (*Index, error) {
	if embedder == nil {
		return nil, fmt.Errorf("embedder cannot be nil")
	}
	return &Index{
		embedder: embedder,
		classes:  make(map[string]Class),
	}, nil
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

func (i *Index) LoadTTL(ctx context.Context, filepath string) error {
	loader := NewLoader()
	if err := loader.LoadTTL(filepath); err != nil {
		return fmt.Errorf("failed to load ontology: %w", err)
	}

	for _, c := range loader.classes {
		i.classes[c.URI] = c
	}

	var terms []string
	seen := make(map[string]bool)
	for _, c := range i.classes {
		if c.Label != "" && !seen[c.Label] {
			terms = append(terms, c.Label)
			seen[c.Label] = true
		}
		for _, alt := range c.AltLabels {
			if !seen[alt] {
				terms = append(terms, alt)
				seen[alt] = true
			}
		}
	}

	vectors, err := i.embedder.GenerateBatch(ctx, terms)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	i.terms = terms
	i.vectors = vectors
	i.dimension = len(vectors[0])

	return nil
}

func (i *Index) resolvePrefix(uri string) string {
	prefixes := map[string]string{
		"sw:": "http://soypete.tech/twitch/topics/",
	}
	for prefix, ns := range prefixes {
		if strings.HasPrefix(uri, prefix) {
			return ns + strings.TrimPrefix(uri, prefix)
		}
	}
	return uri
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

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return float32(dotProduct / (normA * normB))
}

type SearchResult struct {
	Term  string
	Score float64
}

func (i *Index) TermCount() int {
	return len(i.terms)
}

func (i *Index) GetClasses() []Class {
	classes := make([]Class, 0, len(i.classes))
	for _, c := range i.classes {
		classes = append(classes, c)
	}
	return classes
}

func (i *Index) ClassLabels() []string {
	labels := make([]string, 0, len(i.classes))
	for _, c := range i.classes {
		if c.Label != "" {
			labels = append(labels, c.Label)
		}
	}
	return labels
}

func shortURI(uri string) string {
	if idx := strings.LastIndex(uri, "#"); idx != -1 {
		return uri[idx+1:]
	}
	if idx := strings.LastIndex(uri, "/"); idx != -1 {
		return uri[idx+1:]
	}
	return uri
}

func stripLanguageTag(s string) string {
	if idx := strings.Index(s, "@"); idx != -1 {
		return s[:idx]
	}
	return s
}
