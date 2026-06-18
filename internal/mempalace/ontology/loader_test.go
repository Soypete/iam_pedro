package ontology

import (
	"os"
	"path/filepath"
	"testing"
)

func testDataPath() string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "testdata", "twitch_topics.ttl")
}

func TestLoader_LoadTTL(t *testing.T) {
	loader := NewLoader()

	err := loader.LoadTTL(testDataPath())
	if err != nil {
		t.Fatalf("failed to load TTL: %v", err)
	}

	classes := loader.GetClasses()
	if len(classes) == 0 {
		t.Fatal("expected classes, got none")
	}

	found := false
	for _, c := range classes {
		if c.Label == "Go" {
			found = true
			if len(c.AltLabels) != 2 {
				t.Errorf("expected 2 alt labels for Go, got %d", len(c.AltLabels))
			}
		}
	}
	if !found {
		t.Error("expected to find Go class")
	}
}

func TestLoader_GetSearchTerms(t *testing.T) {
	loader := NewLoader()

	err := loader.LoadTTL(testDataPath())
	if err != nil {
		t.Fatalf("failed to load TTL: %v", err)
	}

	terms := loader.GetSearchTerms()
	if len(terms) == 0 {
		t.Fatal("expected search terms, got none")
	}

	found := false
	for _, term := range terms {
		if term == "Go" || term == "Golang" || term == "Gopher" {
			found = true
		}
	}
	if !found {
		t.Error("expected to find Go-related terms")
	}
}

func TestLoader_GetClassByLabel(t *testing.T) {
	loader := NewLoader()

	err := loader.LoadTTL(testDataPath())
	if err != nil {
		t.Fatalf("failed to load TTL: %v", err)
	}

	class := loader.GetClassByLabel("Go")
	if class == nil {
		t.Fatal("expected to find Go class")
	}

	if class.Label != "Go" {
		t.Errorf("expected label 'Go', got '%s'", class.Label)
	}

	notFound := loader.GetClassByLabel("NonExistent")
	if notFound != nil {
		t.Error("expected nil for non-existent class")
	}
}

func TestLoader_FileNotFound(t *testing.T) {
	loader := NewLoader()
	err := loader.LoadTTL("nonexistent.ttl")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float32
	}{
		{
			name:     "identical vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{1, 0, 0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{0, 1, 0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{-1, 0, 0},
			expected: -1.0,
		},
		{
			name:     "3d vectors",
			a:        []float32{1, 2, 3},
			b:        []float32{4, 5, 6},
			expected: 0.974,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			if tt.expected == 0 {
				if result != 0 {
					t.Errorf("expected 0, got %f", result)
				}
			} else {
				diff := result - tt.expected
				if diff < 0 {
					diff = -diff
				}
				if diff > 0.01 {
					t.Errorf("expected ~%f, got %f", tt.expected, result)
				}
			}
		})
	}
}

func TestLoader_LoadTTL_FileNotFound(t *testing.T) {
	loader := NewLoader()
	err := loader.LoadTTL("/tmp/does_not_exist_12345.ttl")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
