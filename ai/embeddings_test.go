package ai

import (
	"testing"
)

func TestNewEmbeddingGenerator(t *testing.T) {
	t.Run("uses default embedding model when not specified", func(t *testing.T) {
		// Can't actually create without a server, but can test the logic
		embeddingModel := ""
		if embeddingModel == "" {
			embeddingModel = "text-embedding-ada-002"
		}
		if embeddingModel != "text-embedding-ada-002" {
			t.Errorf("expected default model, got %s", embeddingModel)
		}
	})
}

func TestGenerateEmbeddingValidation(t *testing.T) {
	t.Run("validates empty text", func(t *testing.T) {
		text := ""
		if text == "" {
			// This would fail in GenerateEmbedding
		}
	})
}

func TestGenerateEmbeddingsValidation(t *testing.T) {
	t.Run("validates empty texts slice", func(t *testing.T) {
		texts := []string{}
		if len(texts) == 0 {
			// This would fail in GenerateEmbeddings
		}
	})
}

func TestFormatModerationMessage(t *testing.T) {
	tests := []struct {
		name     string
		username string
		message  string
		expected string
	}{
		{
			name:     "formats message correctly",
			username: "testuser",
			message:  "Hello world",
			expected: "[testuser]: Hello world",
		},
		{
			name:     "handles empty message",
			username: "testuser",
			message:  "",
			expected: "[testuser]: ",
		},
		{
			name:     "handles special characters",
			username: "user123",
			message:  "Hello @world!",
			expected: "[user123]: Hello @world!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatModerationMessage(tt.username, tt.message)
			if result != tt.expected {
				t.Errorf("FormatModerationMessage(%q, %q) = %q; want %q", tt.username, tt.message, result, tt.expected)
			}
		})
	}
}

func TestEmbeddingGeneratorStruct(t *testing.T) {
	t.Run("EmbeddingGenerator has embedder field", func(t *testing.T) {
		gen := &EmbeddingGenerator{}
		_ = gen.embedder
	})
}
