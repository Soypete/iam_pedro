//go:build ignore

package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Soypete/twitch-llm-bot/logging"
)

type LoadTester struct {
	logger         *logging.Logger
	messagesSent   int64
	messagesRecv   int64
	responseTimes  []time.Duration
	errors         []error
	mu             sync.Mutex
}

func NewLoadTester(logger *logging.Logger) *LoadTester {
	return &LoadTester{
		logger: logger,
	}
}

func (lt *LoadTester) RunLoadTest(ctx context.Context, config LoadTestConfig) error {
	lt.logger.Info("Starting load test", 
		"messageCount", config.MessageCount, 
		"duration", config.Duration,
		"concurrent", config.Concurrent)

	start := time.Now()
	var wg sync.WaitGroup

	// Channel to control message sending rate
	msgChan := make(chan int, config.MessageCount)
	for i := 0; i < config.MessageCount; i++ {
		msgChan <- i
	}
	close(msgChan)

	// Start concurrent workers
	for i := 0; i < config.Concurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			lt.worker(ctx, workerID, msgChan)
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	lt.printLoadTestResults(duration, config)
	return nil
}

func (lt *LoadTester) worker(ctx context.Context, workerID int, msgChan <-chan int) {
	for {
		select {
		case <-ctx.Done():
			return
		case msgID, ok := <-msgChan:
			if !ok {
				return
			}
			lt.sendTestMessage(ctx, workerID, msgID)
		}
	}
}

func (lt *LoadTester) sendTestMessage(ctx context.Context, workerID, msgID int) {
	start := time.Now()
	
	message := fmt.Sprintf("Pedro load test message %d from worker %d", msgID, workerID)
	
	// Simulate message sending and response
	fmt.Printf("LOAD_TEST: worker-%d: %s\n", workerID, message)
	
	// Simulate processing time
	time.Sleep(time.Duration(10+msgID%50) * time.Millisecond)
	
	responseTime := time.Since(start)
	
	atomic.AddInt64(&lt.messagesSent, 1)
	atomic.AddInt64(&lt.messagesRecv, 1)
	
	lt.mu.Lock()
	lt.responseTimes = append(lt.responseTimes, responseTime)
	lt.mu.Unlock()
}

func (lt *LoadTester) printLoadTestResults(duration time.Duration, config LoadTestConfig) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	sent := atomic.LoadInt64(&lt.messagesSent)
	recv := atomic.LoadInt64(&lt.messagesRecv)
	
	fmt.Println("\n=== Load Test Results ===")
	fmt.Printf("Total Duration: %v\n", duration)
	fmt.Printf("Messages Sent: %d\n", sent)
	fmt.Printf("Messages Received: %d\n", recv)
	fmt.Printf("Success Rate: %.2f%%\n", float64(recv)/float64(sent)*100)
	fmt.Printf("Messages/Second: %.2f\n", float64(sent)/duration.Seconds())
	
	if len(lt.responseTimes) > 0 {
		var total time.Duration
		var min, max time.Duration = time.Hour, 0
		
		for _, rt := range lt.responseTimes {
			total += rt
			if rt < min {
				min = rt
			}
			if rt > max {
				max = rt
			}
		}
		
		avg := total / time.Duration(len(lt.responseTimes))
		fmt.Printf("Response Time - Min: %v, Max: %v, Avg: %v\n", min, max, avg)
	}
	
	if len(lt.errors) > 0 {
		fmt.Printf("Errors: %d\n", len(lt.errors))
		for i, err := range lt.errors {
			if i < 5 { // Show first 5 errors
				fmt.Printf("  %v\n", err)
			}
		}
		if len(lt.errors) > 5 {
			fmt.Printf("  ... and %d more errors\n", len(lt.errors)-5)
		}
	}
}