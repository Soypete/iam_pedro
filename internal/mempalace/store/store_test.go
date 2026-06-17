package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Soypete/twitch-llm-bot/internal/mempalace/ontology"
)

func TestStore_Init(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("MEMPALACE_DATA_DIR", tmpDir)
	defer os.Unsetenv("MEMPALACE_DATA_DIR")

	s := NewStore()
	classes, err := ontology.ParseTTL(filepath.Join("..", "ontology", "testdata", "twitch_topics.ttl"))
	if err != nil {
		t.Fatalf("failed to parse TTL: %v", err)
	}

	err = s.Init("test-stream-123", classes)
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	defer s.Close()

	if s.streamID != "test-stream-123" {
		t.Errorf("expected streamID 'test-stream-123', got '%s'", s.streamID)
	}
}

func TestStore_WriteMessage(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("MEMPALACE_DATA_DIR", tmpDir)
	defer os.Unsetenv("MEMPALACE_DATA_DIR")

	s := NewStore()
	classes, err := ontology.ParseTTL(filepath.Join("..", "ontology", "testdata", "twitch_topics.ttl"))
	if err != nil {
		t.Fatalf("failed to parse TTL: %v", err)
	}

	err = s.Init("test-stream-123", classes)
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	defer s.Close()

	msg := Message{
		ID:         "msg-123",
		StreamID:   "test-stream-123",
		Username:   "testuser",
		Message:    "Testing Go concurrency",
		Timestamp:  time.Now(),
		Topic:      "Go",
		Confidence: 0.9,
	}

	err = s.WriteMessage(nil, msg)
	if err != nil {
		t.Fatalf("failed to write message: %v", err)
	}
}

func TestStore_Query(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("MEMPALACE_DATA_DIR", tmpDir)
	defer os.Unsetenv("MEMPALACE_DATA_DIR")

	s := NewStore()
	classes, err := ontology.ParseTTL(filepath.Join("..", "ontology", "testdata", "twitch_topics.ttl"))
	if err != nil {
		t.Fatalf("failed to parse TTL: %v", err)
	}

	err = s.Init("test-stream-123", classes)
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	defer s.Close()

	msg := Message{
		ID:         "msg-456",
		StreamID:   "test-stream-123",
		Username:   "gopher",
		Message:    "I love learning Go channels",
		Timestamp:  time.Now(),
		Topic:      "Go",
		Confidence: 0.85,
	}

	err = s.WriteMessage(nil, msg)
	if err != nil {
		t.Fatalf("failed to write message: %v", err)
	}

	results, err := s.Query(nil, QueryOpts{
		Topic: "Go",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected results, got none")
	}

	if results[0].Username != "gopher" {
		t.Errorf("expected username 'gopher', got '%s'", results[0].Username)
	}
}

func TestStore_QueryByText(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("MEMPALACE_DATA_DIR", tmpDir)
	defer os.Unsetenv("MEMPALACE_DATA_DIR")

	s := NewStore()
	classes, err := ontology.ParseTTL(filepath.Join("..", "ontology", "testdata", "twitch_topics.ttl"))
	if err != nil {
		t.Fatalf("failed to parse TTL: %v", err)
	}

	err = s.Init("test-stream-123", classes)
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	defer s.Close()

	msg := Message{
		ID:         "msg-789",
		StreamID:   "test-stream-123",
		Username:   "searcher",
		Message:    "How do variadic functions work in Go?",
		Timestamp:  time.Now(),
		Topic:      "Go",
		Confidence: 0.9,
	}

	err = s.WriteMessage(nil, msg)
	if err != nil {
		t.Fatalf("failed to write message: %v", err)
	}

	results, err := s.Query(nil, QueryOpts{
		QueryText: "variadic",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected results for text search, got none")
	}
}

func TestSanitizeTableName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Go", "go"},
		{"LLM Engineering", "llm_engineering"},
		{"CICD", "cicd"},
		{"Open Source", "open_source"},
		{"TypeScript", "typescript"},
	}

	for _, tt := range tests {
		result := sanitizeTableName(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeTableName(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}
