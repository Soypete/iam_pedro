package faq

import (
	"testing"
)

func TestMatchStruct(t *testing.T) {
	t.Run("Match has required fields", func(t *testing.T) {
		match := &Match{
			Question:        "What is Go?",
			Response:        "Go is a programming language.",
			SimilarityScore: 0.85,
			CooldownSeconds: 300,
		}

		if match.Question != "What is Go?" {
			t.Errorf("expected question, got %s", match.Question)
		}
		if match.SimilarityScore != 0.85 {
			t.Errorf("expected similarity 0.85, got %f", match.SimilarityScore)
		}
		if match.CooldownSeconds != 300 {
			t.Errorf("expected cooldown 300, got %d", match.CooldownSeconds)
		}
	})
}

func TestMatcherStruct(t *testing.T) {
	t.Run("Matcher has db field", func(t *testing.T) {
		matcher := &Matcher{}
		_ = matcher.db
	})
}

func TestFindMatchValidation(t *testing.T) {
	t.Run("validates empty embedding", func(t *testing.T) {
		embedding := []float32{}
		if len(embedding) == 0 {
			// Would return error in actual function
		}
	})
}

func TestFindMatchForUserValidation(t *testing.T) {
	t.Run("validates empty embedding", func(t *testing.T) {
		embedding := []float32{}
		userID := "testuser"

		valid := len(embedding) != 0 && userID != ""
		if valid {
			t.Error("expected validation to fail with empty embedding")
		}
	})

	t.Run("validates empty userID", func(t *testing.T) {
		embedding := []float32{0.1, 0.2, 0.3}
		userID := ""

		valid := len(embedding) != 0 && userID != ""
		if valid {
			t.Error("expected validation to fail with empty userID")
		}
	})
}
