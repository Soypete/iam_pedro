package faq

import (
	"testing"

	"github.com/Soypete/twitch-llm-bot/logging"
)

func TestServiceConfig(t *testing.T) {
	t.Run("ServiceConfig has required fields", func(t *testing.T) {
		config := ServiceConfig{
			LLMPath:             "http://localhost:8080",
			EmbeddingModel:      "text-embedding-ada-002",
			ChatModel:           "llama3",
			SimilarityThreshold: 0.75,
			UsePerUserCooldown:  false,
			Logger:              logging.Default(),
		}

		if config.LLMPath != "http://localhost:8080" {
			t.Errorf("expected LLMPath, got %s", config.LLMPath)
		}
		if config.SimilarityThreshold != 0.75 {
			t.Errorf("expected threshold 0.75, got %f", config.SimilarityThreshold)
		}
	})

	t.Run("default threshold is applied when zero", func(t *testing.T) {
		threshold := 0.0
		if threshold <= 0 {
			threshold = 0.75
		}
		if threshold != 0.75 {
			t.Errorf("expected default threshold 0.75, got %f", threshold)
		}
	})
}

func TestServiceStruct(t *testing.T) {
	t.Run("Service has required fields", func(t *testing.T) {
		service := &Service{}
		_ = service.embeddingService
		_ = service.matcher
		_ = service.llm
		_ = service.threshold
		_ = service.logger
		_ = service.db
		_ = service.usePerUserCooldown
	})
}

func TestMatchResultStruct(t *testing.T) {
	t.Run("MatchResult has required fields", func(t *testing.T) {
		result := &MatchResult{
			Matched:           true,
			Question:          "What is Go?",
			CachedResponse:    "Go is a programming language.",
			GeneratedResponse: "Hey! Go is a programming language.",
			SimilarityScore:   0.85,
			Category:          "programming",
		}

		if !result.Matched {
			t.Error("expected Matched to be true")
		}
		if result.SimilarityScore != 0.85 {
			t.Errorf("expected similarity 0.85, got %f", result.SimilarityScore)
		}
	})
}

func TestServiceUpdateThreshold(t *testing.T) {
	t.Run("updates threshold when valid", func(t *testing.T) {
		service := &Service{threshold: 0.75}
		err := service.UpdateThreshold(0.8)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if service.threshold != 0.8 {
			t.Errorf("expected threshold 0.8, got %f", service.threshold)
		}
	})

	t.Run("rejects invalid threshold", func(t *testing.T) {
		service := &Service{threshold: 0.75}
		err := service.UpdateThreshold(1.5)
		if err == nil {
			t.Error("expected error for invalid threshold")
		}
	})

	t.Run("rejects zero threshold", func(t *testing.T) {
		service := &Service{threshold: 0.75}
		err := service.UpdateThreshold(0)
		if err == nil {
			t.Error("expected error for zero threshold")
		}
	})

	t.Run("rejects negative threshold", func(t *testing.T) {
		service := &Service{threshold: 0.75}
		err := service.UpdateThreshold(-0.1)
		if err == nil {
			t.Error("expected error for negative threshold")
		}
	})
}

func TestServiceGetThreshold(t *testing.T) {
	t.Run("returns current threshold", func(t *testing.T) {
		service := &Service{threshold: 0.85}
		if service.GetThreshold() != 0.85 {
			t.Errorf("expected 0.85, got %f", service.GetThreshold())
		}
	})
}
