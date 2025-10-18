package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"time"

	"github.com/Soypete/twitch-llm-bot/keepalive"
	"github.com/Soypete/twitch-llm-bot/logging"
)

func main() {
	var (
		discordBotURL string
		twitchBotURL  string
		discordToken  string
		logLevel      string
		checkInterval int
		alertInterval int
	)

	flag.StringVar(&discordBotURL, "discord-bot-url", "http://localhost:6060/healthz", "Discord bot health endpoint")
	flag.StringVar(&twitchBotURL, "twitch-bot-url", "", "Twitch bot health endpoint (optional)")
	flag.StringVar(&discordToken, "discord-token", "", "Discord bot token for alerts")
	flag.StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.IntVar(&checkInterval, "check-interval", 60, "Health check interval in seconds")
	flag.IntVar(&alertInterval, "alert-interval", 3600, "Alert repeat interval in seconds (default: 1 hour)")
	flag.Parse()

	// Initialize logger
	logger := logging.NewLogger(logging.LogLevel(logLevel), os.Stdout)

	// Validate required flags
	if discordToken == "" {
		logger.Error("discord-token is required")
		os.Exit(1)
	}

	// Create Discord alerter (channel and user are hardcoded in the alerter)
	alerter, err := keepalive.NewDiscordAlerter(discordToken, logger)
	if err != nil {
		logger.Error("failed to create Discord alerter", "error", err.Error())
		os.Exit(1)
	}
	defer func() {
		if err := alerter.Close(); err != nil {
			logger.Error("failed to close Discord alerter", "error", err.Error())
		}
	}()

	// Configure services to monitor
	services := []keepalive.ServiceConfig{
		{
			Name:      "Discord Bot",
			HealthURL: discordBotURL,
		},
	}

	// Add Twitch bot if URL is provided
	if twitchBotURL != "" {
		services = append(services, keepalive.ServiceConfig{
			Name:      "Twitch Bot",
			HealthURL: twitchBotURL,
		})
	}

	// Add VLLM/llama.cpp if LLAMA_CPP_PATH is set
	llamaCppPath := os.Getenv("LLAMA_CPP_PATH")
	if llamaCppPath != "" {
		// Convert llama.cpp URL to health endpoint
		// e.g., http://127.0.0.1:8080 -> http://127.0.0.1:8080/health
		vllmHealthURL := llamaCppPath
		if vllmHealthURL[len(vllmHealthURL)-1] != '/' {
			vllmHealthURL += "/"
		}
		vllmHealthURL += "health"

		services = append(services, keepalive.ServiceConfig{
			Name:      "VLLM/llama.cpp",
			HealthURL: vllmHealthURL,
		})
	}

	// Create keepalive service
	kas := keepalive.NewKeepAliveService(
		services,
		time.Duration(checkInterval)*time.Second,
		time.Duration(alertInterval)*time.Second,
		alerter,
		logger,
	)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown gracefully
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	go func() {
		<-stop
		logger.Info("Received interrupt signal, shutting down...")
		cancel()
	}()

	logger.Info("Starting KeepAlive service",
		"check_interval", checkInterval,
		"alert_interval", alertInterval,
		"monitored_services", len(services))

	// Start the keepalive service
	if err := kas.Start(ctx); err != nil && err != context.Canceled {
		logger.Error("keepalive service error", "error", err.Error())
		os.Exit(1)
	}

	logger.Info("KeepAlive service stopped")
}
