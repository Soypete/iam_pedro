// Package ontology provides SKOS thesaurus parsing and cosine-similarity
// indexing for topic classification.
//
// It loads TTL (Turtle) files containing skos:Concept definitions with
// prefLabel, altLabel, and definition. The loader builds an in-memory
// index of all labels (including alternatives) for fast cosine-similarity
// matching against chat messages.
//
// Default ontology: testdata/twitch_topics.ttl (~25 programming/tech topics)
package ontology

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

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
	defer func() { _ = file.Close() }()

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
			currentClass = Class{}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error scanning ontology file: %w", err)
	}

	return nil
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
	for _, part := range parts {
		part = strings.TrimSuffix(part, ";")
		part = strings.TrimSuffix(part, ".")
		if strings.HasPrefix(part, "sw:") || strings.HasPrefix(part, "schema:") {
			return part
		}
	}
	if len(parts) >= 3 {
		return parts[2]
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
	var terms []string
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

func ParseTTL(filepath string) ([]Class, error) {
	loader := NewLoader()
	if err := loader.LoadTTL(filepath); err != nil {
		return nil, err
	}
	return loader.GetClasses(), nil
}

func (l *Loader) GetClassByLabel(label string) *Class {
	for _, c := range l.classes {
		if c.Label == label {
			return &c
		}
	}
	return nil
}
