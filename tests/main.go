//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/Soypete/twitch-llm-bot/logging"
)

func main() {
	logger := logging.NewLogger(logging.LogLevel("info"), os.Stdout)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	var wg sync.WaitGroup

	// Start test framework
	testFramework := NewTestFramework(logger)
	
	// Run integration tests
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := testFramework.RunIntegrationTests(ctx)
		if err != nil {
			logger.Error("Integration tests failed", "error", err)
		}
		testFramework.PrintResults()
	}()

	// Start continuous test client
	testClient := NewTestClient(logger)
	wg.Add(1)
	go func() {
		defer wg.Done()
		testClient.Run(ctx)
	}()

	fmt.Println("Pedro Integration Test Suite Started")
	fmt.Println("Press Ctrl+C to stop")

	<-stop
	logger.Info("Shutting down test suite")
	cancel()
	wg.Wait()
}