package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/Soypete/twitch-llm-bot/keepalive"
	"github.com/Soypete/twitch-llm-bot/logging"
)

// getEnv gets an environment variable with a default fallback
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an integer environment variable with a default fallback
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func main() {
	// Read configuration from environment variables
	discordBotURL := getEnv("DISCORD_BOT_URL", "http://localhost:6060/healthz")
	twitchBotURL := getEnv("TWITCH_BOT_URL", "")
	discordToken := getEnv("DISCORD_SECRET", "")
	discordUserID := getEnv("DISCORD_ALERT_USER_ID", "") // Discord user ID for mentions
	logLevel := getEnv("LOG_LEVEL", "info")
	checkInterval := getEnvInt("CHECK_INTERVAL", 60)
	alertInterval := getEnvInt("ALERT_INTERVAL", 3600)

	// Initialize logger
	logger := logging.NewLogger(logging.LogLevel(logLevel), os.Stdout)

	// Validate required token
	if discordToken == "" {
		logger.Error("DISCORD_SECRET environment variable is required")
		os.Exit(1)
	}

	// Create Discord alerter
	alerter, err := keepalive.NewDiscordAlerter(discordToken, discordUserID, logger)
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
