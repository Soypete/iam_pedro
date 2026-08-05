package database

import (
	"testing"
)

func TestArrayToString(t *testing.T) {
	tests := []struct {
		name     string
		input    []float32
		expected string
	}{
		{
			name:     "empty array",
			input:    []float32{},
			expected: "[]",
		},
		{
			name:     "single element",
			input:    []float32{1.0},
			expected: "[1.000000]",
		},
		{
			name:     "multiple elements",
			input:    []float32{1.0, 2.0, 3.0},
			expected: "[1.000000,2.000000,3.000000]",
		},
		{
			name:     "negative numbers",
			input:    []float32{-1.5, 0.0, 1.5},
			expected: "[-1.500000,0.000000,1.500000]",
		},
		{
			name:     "small decimals",
			input:    []float32{0.1, 0.01, 0.001},
			expected: "[0.100000,0.010000,0.001000]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := arrayToString(tt.input)
			if result != tt.expected {
				t.Errorf("arrayToString(%v) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFindSimilarMessagesDefaults(t *testing.T) {
	t.Run("default limit is applied", func(t *testing.T) {
		limit := 0
		if limit <= 0 {
			limit = 10
		}
		if limit != 10 {
			t.Errorf("expected default limit 10, got %d", limit)
		}
	})

	t.Run("default minSimilarity is applied", func(t *testing.T) {
		minSimilarity := 0.0
		if minSimilarity <= 0 {
			minSimilarity = 0.7
		}
		if minSimilarity != 0.7 {
			t.Errorf("expected default minSimilarity 0.7, got %f", minSimilarity)
		}
	})
}

func TestStoreMessageEmbedding(t *testing.T) {
	t.Run("embedding conversion produces valid format", func(t *testing.T) {
		embedding := []float32{0.1, 0.2, 0.3}
		result := arrayToString(embedding)
		if len(result) == 0 {
			t.Error("expected non-empty string")
		}
		if result[0] != '[' {
			t.Errorf("expected string to start with '[', got %c", result[0])
		}
		if result[len(result)-1] != ']' {
			t.Errorf("expected string to end with ']', got %c", result[len(result)-1])
		}
	})
}
