package faq

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVectorToString(t *testing.T) {
	tests := []struct {
		name   string
		vector []float32
		want   string
	}{
		{
			name:   "empty vector",
			vector: []float32{},
			want:   "[]",
		},
		{
			name:   "single element",
			vector: []float32{0.5},
			want:   "[0.500000]",
		},
		{
			name:   "multiple elements",
			vector: []float32{0.1, 0.2, 0.3},
			want:   "[0.100000,0.200000,0.300000]",
		},
		{
			name:   "negative values",
			vector: []float32{-0.5, 0.5, -1.0},
			want:   "[-0.500000,0.500000,-1.000000]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VectorToString(tt.vector)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestEmbeddingDimension(t *testing.T) {
	// Verify the constant is set correctly for text-embedding-3-small
	assert.Equal(t, 1536, EmbeddingDimension)
}
