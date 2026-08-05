package ontology

import (
	"testing"
)

func TestClassStruct(t *testing.T) {
	t.Run("Class has required fields", func(t *testing.T) {
		class := &Class{
			URI:        "http://example.org/golang",
			Label:      "Go",
			AltLabels:  []string{"Golang", "GoLang"},
			Definition: "A programming language",
			SubClassOf: "http://example.org/language",
		}

		if class.Label != "Go" {
			t.Errorf("expected 'Go', got %s", class.Label)
		}
		if len(class.AltLabels) != 2 {
			t.Errorf("expected 3 alt labels, got %d", len(class.AltLabels))
		}
	})
}

func TestLoaderStruct(t *testing.T) {
	t.Run("Loader has classes map", func(t *testing.T) {
		loader := &Loader{}
		_ = loader.classes
	})
}

func TestNewLoader(t *testing.T) {
	t.Run("creates loader with empty classes", func(t *testing.T) {
		loader := NewLoader()
		if loader.classes == nil {
			t.Error("expected non-nil classes map")
		}
		if len(loader.classes) != 0 {
			t.Errorf("expected 0 classes, got %d", len(loader.classes))
		}
	})
}

func TestIndexStruct(t *testing.T) {
	t.Run("Index has required fields", func(t *testing.T) {
		index := &Index{}
		_ = index.terms
		_ = index.vectors
		_ = index.dimension
		_ = index.embedder
		_ = index.classes
	})
}

func TestNewIndex(t *testing.T) {
	t.Run("returns error for nil embedder", func(t *testing.T) {
		_, err := NewIndex(nil)
		if err == nil {
			t.Error("expected error for nil embedder")
		}
	})

	t.Run("creates index for valid embedder", func(t *testing.T) {
		// Would need mock embedder
	})
}

func TestSearchResultStruct(t *testing.T) {
	t.Run("SearchResult has required fields", func(t *testing.T) {
		result := &SearchResult{
			Term:  "golang",
			Score: 0.85,
		}

		if result.Term != "golang" {
			t.Errorf("expected 'golang', got %s", result.Term)
		}
		if result.Score != 0.85 {
			t.Errorf("expected 0.85, got %f", result.Score)
		}
	})
}

func TestCosineSimilarity(t *testing.T) {
	t.Run("calculates similarity for identical vectors", func(t *testing.T) {
		a := []float32{1.0, 0.0}
		b := []float32{1.0, 0.0}

		sim := cosineSimilarity(a, b)
		if sim != 1.0 {
			t.Errorf("expected 1.0, got %f", sim)
		}
	})

	t.Run("returns 0 for mismatched lengths", func(t *testing.T) {
		a := []float32{1.0, 0.0}
		b := []float32{1.0, 0.0, 0.0}

		sim := cosineSimilarity(a, b)
		if sim != 0.0 {
			t.Errorf("expected 0.0, got %f", sim)
		}
	})

	t.Run("returns 0 for zero vectors", func(t *testing.T) {
		a := []float32{0.0, 0.0}
		b := []float32{1.0, 0.0}

		sim := cosineSimilarity(a, b)
		if sim != 0.0 {
			t.Errorf("expected 0.0, got %f", sim)
		}
	})
}

func TestGetClasses(t *testing.T) {
	t.Run("returns classes from loader", func(t *testing.T) {
		loader := NewLoader()
		classes := loader.GetClasses()

		if len(classes) != 0 {
			t.Errorf("expected 0 classes, got %d", len(classes))
		}
	})
}

func TestGetSearchTerms(t *testing.T) {
	t.Run("returns terms from classes", func(t *testing.T) {
		loader := NewLoader()
		terms := loader.GetSearchTerms()

		if len(terms) != 0 {
			t.Errorf("expected 0 terms, got %d", len(terms))
		}
	})
}

func TestTermCount(t *testing.T) {
	t.Run("returns term count", func(t *testing.T) {
		index := &Index{terms: []string{"a", "b", "c"}}
		if index.TermCount() != 3 {
			t.Errorf("expected 3, got %d", index.TermCount())
		}
	})
}

func TestGetClassesFromIndex(t *testing.T) {
	t.Run("returns classes from index", func(t *testing.T) {
		index := &Index{
			classes: map[string]Class{
				"http://example.org/go": {Label: "Go"},
			},
		}
		classes := index.GetClasses()
		if len(classes) != 1 {
			t.Errorf("expected 1 class, got %d", len(classes))
		}
	})
}

func TestClassLabels(t *testing.T) {
	t.Run("returns class labels", func(t *testing.T) {
		index := &Index{
			classes: map[string]Class{
				"http://example.org/go":   {Label: "Go"},
				"http://example.org/java": {Label: "Java"},
			},
		}
		labels := index.ClassLabels()
		if len(labels) != 2 {
			t.Errorf("expected 2 labels, got %d", len(labels))
		}
	})

	t.Run("excludes empty labels", func(t *testing.T) {
		index := &Index{
			classes: map[string]Class{
				"http://example.org/go":      {Label: "Go"},
				"http://example.org/unknown": {},
			},
		}
		labels := index.ClassLabels()
		if len(labels) != 1 {
			t.Errorf("expected 1 label, got %d", len(labels))
		}
	})
}

func TestGetClassByLabel(t *testing.T) {
	t.Run("finds class by label", func(t *testing.T) {
		loader := NewLoader()
		loader.classes["http://example.org/go"] = Class{Label: "Go"}

		class := loader.GetClassByLabel("Go")
		if class == nil || class.Label != "Go" {
			t.Error("expected to find class with label 'Go'")
		}
	})

	t.Run("returns nil for missing label", func(t *testing.T) {
		loader := NewLoader()
		class := loader.GetClassByLabel("Missing")
		if class != nil {
			t.Error("expected nil for missing class")
		}
	})
}
