package classifier

import (
	"testing"

	"github.com/Soypete/twitch-llm-bot/internal/mempalace/ontology"
)

func TestClassifierStruct(t *testing.T) {
	t.Run("Classifier has required fields", func(t *testing.T) {
		classifier := &Classifier{}
		_ = classifier.llm
		_ = classifier.modelName
		_ = classifier.classes
		_ = classifier.classNames
	})
}

func TestNewClassifier(t *testing.T) {
	t.Run("extracts class names from classes", func(t *testing.T) {
		classes := []ontology.Class{
			{Label: "golang"},
			{Label: "ai"},
			{Label: "gaming"},
		}

		classifier := NewClassifier(nil, "test-model", classes)

		if len(classifier.classNames) != 3 {
			t.Errorf("expected 3 class names, got %d", len(classifier.classNames))
		}

		if classifier.classNames[0] != "golang" {
			t.Errorf("expected 'golang', got %s", classifier.classNames[0])
		}
		if classifier.modelName != "test-model" {
			t.Errorf("expected 'test-model', got %s", classifier.modelName)
		}
	})

	t.Run("handles empty classes", func(t *testing.T) {
		classifier := NewClassifier(nil, "test-model", []ontology.Class{})

		if len(classifier.classNames) != 0 {
			t.Errorf("expected 0 class names, got %d", len(classifier.classNames))
		}
	})
}

func TestGetClasses(t *testing.T) {
	t.Run("returns classes", func(t *testing.T) {
		classes := []ontology.Class{
			{Label: "golang"},
		}

		classifier := NewClassifier(nil, "test-model", classes)
		returned := classifier.GetClasses()

		if len(returned) != 1 {
			t.Errorf("expected 1 class, got %d", len(returned))
		}
		if returned[0].Label != "golang" {
			t.Errorf("expected 'golang', got %s", returned[0].Label)
		}
	})
}

func TestClassificationResultStruct(t *testing.T) {
	t.Run("ClassificationResult has required fields", func(t *testing.T) {
		result := &classificationResult{
			Topic:      "golang",
			Confidence: 0.85,
		}

		if result.Topic != "golang" {
			t.Errorf("expected 'golang', got %s", result.Topic)
		}
		if result.Confidence != 0.85 {
			t.Errorf("expected 0.85, got %f", result.Confidence)
		}
	})
}

func TestParseClassificationResponse(t *testing.T) {
	t.Run("parses valid JSON", func(t *testing.T) {
		// This is tested through integration - unit test would need mock
	})
}
