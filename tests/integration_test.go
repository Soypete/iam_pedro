//go:build ignore

package main

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Soypete/twitch-llm-bot/ai/twitchchat"
	"github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/logging"
	twitchirc "github.com/Soypete/twitch-llm-bot/twitch"
	"github.com/Soypete/twitch-llm-bot/types"
)

func TestPedroTwitchIntegration(t *testing.T) {
	logger := logging.NewLogger(logging.LogLevel("debug"), os.Stdout)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Setup test database
	db, err := database.NewPostgres(logger)
	if err != nil {
		t.Skipf("Skipping integration test - no database available: %v", err)
	}

	// Setup LLM (if available)
	os.Setenv("OPENAI_API_KEY", "test")
	llmPath := os.Getenv("LLAMA_CPP_PATH")
	if llmPath == "" {
		t.Skip("Skipping integration test - no LLAMA_CPP_PATH set")
	}

	twitchllm, err := twitchchat.Setup(llmPath, logger)
	if err != nil {
		t.Skipf("Skipping integration test - LLM setup failed: %v", err)
	}

	var wg sync.WaitGroup

	// Setup Pedro's Twitch IRC
	irc, err := twitchirc.SetupTwitchIRC(&wg, twitchllm, "test-model", db, logger)
	if err != nil {
		t.Skipf("Skipping integration test - Twitch IRC setup failed: %v", err)
	}

	// Test message handling
	testMessage := types.TwitchMessage{
		Username:  "testuser",
		Text:      "Hey Pedro, this is a test message!",
		IsCommand: false,
		Time:      time.Now(),
	}

	// Insert test message
	messageID, err := db.InsertMessage(ctx, testMessage)
	if err != nil {
		t.Fatalf("Failed to insert test message: %v", err)
	}

	// Test LLM response
	resp, err := twitchllm.SingleMessageResponse(ctx, testMessage, messageID)
	if err != nil {
		t.Fatalf("Failed to get LLM response: %v", err)
	}

	if resp.Text == "" {
		t.Error("Expected non-empty response from LLM")
	}

	t.Logf("Test passed - received response: %s", resp.Text)
}

func TestPedroLoadHandling(t *testing.T) {
	logger := logging.NewLogger(logging.LogLevel("info"), os.Stdout)
	
	// Create test client
	testClient := NewTestClient(logger)
	
	// Test load generation
	start := time.Now()
	testClient.SendLoadTestMessages(10, 5*time.Second)
	duration := time.Since(start)
	
	if duration > 10*time.Second {
		t.Errorf("Load test took too long: %v", duration)
	}
	
	t.Logf("Load test completed in %v", duration)
}

func BenchmarkMessageProcessing(b *testing.B) {
	logger := logging.NewLogger(logging.LogLevel("error"), os.Stdout)
	testClient := NewTestClient(logger)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := fmt.Sprintf("Pedro test message %d", i)
		fmt.Printf("BENCHMARK_MSG: testuser: %s\n", msg)
	}
}