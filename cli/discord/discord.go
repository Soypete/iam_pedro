package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"

	"github.com/Soypete/twitch-llm-bot/ai/discordchat"
	database "github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/discord"
	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/Soypete/twitch-llm-bot/metrics"
)

func main() {
	var model string
	var logLevel string

	flag.StringVar(&model, "model", "", "The model to use for the LLM")
	flag.StringVar(&logLevel, "errorLevel", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	// Initialize logger
	logger := logging.NewLogger(logging.LogLevel(logLevel), os.Stdout)

	ctx := context.Background()
	stop := make(chan os.Signal, 1)
	wg := &sync.WaitGroup{}

	// listen and serve for metrics server.
	// TODO: change these configs to file
	server := metrics.SetupServer()
	go server.Run()

	// setup postgres connection
	// change these configs to file
	db, err := database.NewPostgres(logger)
	if err != nil {
		logger.Error("failed to connect to postgres", "error", err.Error())
		os.Exit(1)
	}

	// setup llm connection
	//  we are not actually connecting to openai, but we are using their api spec to connect to our own model via llama.cpp
	os.Setenv("OPENAI_API_KEY", "test")
	llmPath := os.Getenv("LLAMA_CPP_PATH")

	// Debug: Log environment variables
	logger.Info("environment configuration",
		"model", model,
		"llmPath", llmPath,
		"hasDiscordToken", os.Getenv("DISCORD_TOKEN") != "",
		"hasDatabaseURL", os.Getenv("DATABASE_URL") != "",
	)

	if llmPath == "" {
		logger.Error("LLAMA_CPP_PATH environment variable is not set")
		os.Exit(1)
	}

	var session discord.Client
	discordllm, err := discordchat.Setup(db, model, llmPath, logger)
	if err != nil {
		logger.Error("failed to setup discord LLM", "error", err.Error())
		os.Exit(1)
	}

	session, err = discord.Setup(discordllm, db, logger)
	if err != nil {
		logger.Error("failed to setup discord session", "error", err.Error())
		stop <- os.Interrupt
	}

	signal.Notify(stop, os.Interrupt)
	logger.Info("Press Ctrl+C to exit")
	Shutdown(ctx, wg, session, stop, logger)
}

// Shutdown cancels the context and logs a message.
// TODO: this needs to be handled with an os signal
func Shutdown(ctx context.Context, wg *sync.WaitGroup, session discord.Client, stop chan os.Signal, logger *logging.Logger,
) {
	<-stop
	ctx.Done()

	if session.Session != nil {
		err := session.Session.Close()
		if err != nil {
			logger.Error("error closing discord session", "error", err.Error())
		}
	}

	// wg.Done()
	logger.Info("Shutting down")
	os.Exit(0)
}
