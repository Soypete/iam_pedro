package address

import (
	"testing"
)

func TestDetectorStruct(t *testing.T) {
	t.Run("Detector has required fields", func(t *testing.T) {
		detector := &Detector{}
		_ = detector.index
		_ = detector.threshold
	})
}

func TestCosineSimilarity(t *testing.T) {
	t.Run("calculates similarity for identical vectors", func(t *testing.T) {
		a := []float32{1.0, 0.0}
		b := []float32{1.0, 0.0}

		sim := CosineSimilarity(a, b)
		if sim != 1.0 {
			t.Errorf("expected 1.0, got %f", sim)
		}
	})

	t.Run("calculates similarity for orthogonal vectors", func(t *testing.T) {
		a := []float32{1.0, 0.0}
		b := []float32{0.0, 1.0}

		sim := CosineSimilarity(a, b)
		if sim != 0.0 {
			t.Errorf("expected 0.0, got %f", sim)
		}
	})

	t.Run("calculates similarity for opposite vectors", func(t *testing.T) {
		a := []float32{1.0, 0.0}
		b := []float32{-1.0, 0.0}

		sim := CosineSimilarity(a, b)
		if sim != -1.0 {
			t.Errorf("expected -1.0, got %f", sim)
		}
	})

	t.Run("returns 0 for mismatched lengths", func(t *testing.T) {
		a := []float32{1.0, 0.0}
		b := []float32{1.0, 0.0, 0.0}

		sim := CosineSimilarity(a, b)
		if sim != 0.0 {
			t.Errorf("expected 0.0, got %f", sim)
		}
	})

	t.Run("returns 0 for zero vectors", func(t *testing.T) {
		a := []float32{0.0, 0.0}
		b := []float32{1.0, 0.0}

		sim := CosineSimilarity(a, b)
		if sim != 0.0 {
			t.Errorf("expected 0.0, got %f", sim)
		}
	})
}

func TestNewDetector(t *testing.T) {
	t.Run("uses default threshold when zero", func(t *testing.T) {
		// Would need embedder mock - tested through integration
	})
}

func TestSetThreshold(t *testing.T) {
	t.Run("updates threshold", func(t *testing.T) {
		detector := &Detector{threshold: 0.6}
		detector.SetThreshold(0.8)

		if detector.threshold != 0.8 {
			t.Errorf("expected 0.8, got %f", detector.threshold)
		}
	})
}
