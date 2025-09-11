//go:build ignore

package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/bwmarrin/discordgo"
	v2 "github.com/gempir/go-twitch-irc/v2"
)

type TestClient struct {
	twitchClient  *v2.Client
	discordClient *discordgo.Session
	logger        *logging.Logger
	testMessages  []string
}

func NewTestClient(logger *logging.Logger) *TestClient {
	return &TestClient{
		logger: logger,
		testMessages: []string{
			"Hey Pedro, how are you doing?",
			"Pedro can you help me with Go?",
			"What's the weather like pedro?",
			"pedro, tell me a joke",
			"How do I use the bot Pedro?",
			"Pedro explain containers",
			"pedro what is kubernetes?",
			"Can you help pedro?",
		},
	}
}

func (tc *TestClient) Run(ctx context.Context) {
	tc.logger.Info("Starting test client")

	// Test with simulated Twitch IRC
	tc.simulateTwitchMessages(ctx)
}

func (tc *TestClient) simulateTwitchMessages(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			tc.logger.Info("Test client stopping")
			return
		case <-ticker.C:
			msg := tc.testMessages[rand.Intn(len(tc.testMessages))]
			tc.logger.Info("Simulating Twitch message", "message", msg)
			
			// In a real test, this would send to Pedro's Twitch channel
			// For now, we just log the message
			fmt.Printf("TWITCH_MSG: testuser: %s\n", msg)
		}
	}
}

func (tc *TestClient) SendLoadTestMessages(count int, duration time.Duration) {
	tc.logger.Info("Starting load test", "messageCount", count, "duration", duration)
	
	interval := duration / time.Duration(count)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	sent := 0
	start := time.Now()
	
	for sent < count {
		msg := fmt.Sprintf("Load test message %d - Pedro help!", sent+1)
		fmt.Printf("LOAD_TEST_MSG: testuser%d: %s\n", sent+1, msg)
		tc.logger.Debug("Sent load test message", "count", sent+1)
		sent++
		<-ticker.C
	}
	
	tc.logger.Info("Load test completed", "duration", time.Since(start), "messagesSent", sent)
}