//go:build ignore

package main

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/Soypete/twitch-llm-bot/logging"
)

type BenchmarkSuite struct {
	logger *logging.Logger
}

func NewBenchmarkSuite(logger *logging.Logger) *BenchmarkSuite {
	return &BenchmarkSuite{logger: logger}
}

func (bs *BenchmarkSuite) RunBenchmarks(ctx context.Context) {
	fmt.Println("\n=== Pedro Performance Benchmarks ===")
	
	// Memory usage benchmark
	bs.benchmarkMemoryUsage()
	
	// Message processing throughput
	bs.benchmarkMessageThroughput(ctx)
	
	// Concurrent user simulation
	bs.benchmarkConcurrentUsers(ctx)
}

func (bs *BenchmarkSuite) benchmarkMemoryUsage() {
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	// Simulate Pedro's typical workload
	testClient := NewTestClient(bs.logger)
	testFramework := NewTestFramework(bs.logger)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Run some tests to measure memory
	testFramework.RunIntegrationTests(ctx)
	testClient.SendLoadTestMessages(100, 5*time.Second)
	
	runtime.GC()
	runtime.ReadMemStats(&m2)
	
	fmt.Printf("Memory Usage:\n")
	fmt.Printf("  Heap: %d KB\n", (m2.HeapAlloc-m1.HeapAlloc)/1024)
	fmt.Printf("  Total Alloc: %d KB\n", (m2.TotalAlloc-m1.TotalAlloc)/1024)
	fmt.Printf("  Sys: %d KB\n", (m2.Sys-m1.Sys)/1024)
	fmt.Printf("  GC Cycles: %d\n", m2.NumGC-m1.NumGC)
}

func (bs *BenchmarkSuite) benchmarkMessageThroughput(ctx context.Context) {
	fmt.Printf("\nMessage Throughput Benchmark:\n")
	
	messageRates := []int{10, 50, 100, 200}
	
	for _, rate := range messageRates {
		start := time.Now()
		testClient := NewTestClient(bs.logger)
		
		// Send messages at target rate
		testClient.SendLoadTestMessages(rate, 5*time.Second)
		
		duration := time.Since(start)
		actualRate := float64(rate) / duration.Seconds()
		
		fmt.Printf("  Target: %d msg/5s, Actual: %.2f msg/s\n", rate, actualRate)
	}
}

func (bs *BenchmarkSuite) benchmarkConcurrentUsers(ctx context.Context) {
	fmt.Printf("\nConcurrent Users Benchmark:\n")
	
	userCounts := []int{5, 10, 25, 50}
	
	for _, userCount := range userCounts {
		start := time.Now()
		var wg sync.WaitGroup
		
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		
		for i := 0; i < userCount; i++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()
				bs.simulateUser(ctx, userID)
			}(i)
		}
		
		wg.Wait()
		cancel()
		
		duration := time.Since(start)
		fmt.Printf("  %d users: %.2fs\n", userCount, duration.Seconds())
	}
}

func (bs *BenchmarkSuite) simulateUser(ctx context.Context, userID int) {
	messages := []string{
		fmt.Sprintf("Pedro help user %d", userID),
		fmt.Sprintf("Hey pedro, user %d here", userID),
		fmt.Sprintf("pedro explain something to user %d", userID),
	}
	
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	msgCount := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if msgCount >= len(messages) {
				return
			}
			fmt.Printf("USER_%d: %s\n", userID, messages[msgCount])
			msgCount++
		}
	}
}