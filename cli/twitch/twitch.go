package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"

	"github.com/Soypete/twitch-llm-bot/ai/twitchchat"
	database "github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/faq"
	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/Soypete/twitch-llm-bot/metrics"
	twitchirc "github.com/Soypete/twitch-llm-bot/twitch"
)

func main() {
	var model string
	var logLevel string
	var streamConfig string
	var faqConfig string

	flag.StringVar(&model, "model", os.Getenv("MODEL"), "The model to use for the LLM")
	flag.StringVar(&logLevel, "errorLevel", "info", "Log level (debug, info, warn, error)")
	flag.StringVar(&streamConfig, "streamConfig", "", "Path to stream context config file (e.g., 'configs/streams/golang-nov-2025.yaml')")
	flag.StringVar(&faqConfig, "faqConfig", "", "Path to FAQ config file for semantic FAQ responses (e.g., 'configs/faq/entries.yaml')")
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
	_ = os.Setenv("OPENAI_API_KEY", "test")
	llmPath := os.Getenv("LLAMA_CPP_PATH")
	twitchllm, err := twitchchat.SetupWithStreamConfig(llmPath, model, streamConfig, logger)
	if err != nil {
		logger.Error("failed to setup twitch LLM", "error", err.Error())
		os.Exit(1)
	}

	var irc *twitchirc.IRC
	// setup twitch IRC
	irc, err = twitchirc.SetupTwitchIRC(wg, twitchllm, model, db, logger)
	if err != nil {
		logger.Error("failed to setup twitch IRC", "error", err.Error())
		stop <- os.Interrupt
	}

	// Setup FAQ service if config is provided
	if faqConfig != "" {
		logger.Info("setting up FAQ service", "config", faqConfig)
		faqService, err := setupFAQService(db, llmPath, model, faqConfig, logger)
		if err != nil {
			logger.Error("failed to setup FAQ service", "error", err.Error())
			// Continue without FAQ - it's optional
		} else {
			// Create FAQ processor and attach to IRC
			faqProcessor := twitchirc.NewFAQProcessor(faqService, irc.GetAsyncResponseChannel(), logger)
			irc.SetFAQProcessor(faqProcessor)
			logger.Info("FAQ service enabled and attached to Twitch IRC")
		}
	}

	// Register auth health endpoint
	server.RegisterAuthHealthHandler(irc.AuthHealthHandler())
	logger.Debug("auth health endpoint registered at /healthz/auth")

	logger.Info("starting twitch IRC connection")
	// long running function
	err = irc.ConnectIRC(ctx, wg)
	if err != nil {
		logger.Error("failed to connect to twitch IRC", "error", err.Error())
		stop <- os.Interrupt
	}

	go func() {
		err = irc.Client.Connect()
		if err != nil {
			logger.Error("twitch client connection failed", "error", err.Error())
			stop <- os.Interrupt
		}
	}()
	signal.Notify(stop, os.Interrupt)
	logger.Info("Press Ctrl+C to exit")
	Shutdown(ctx, wg, irc, stop, logger)
}

// setupFAQService initializes the FAQ service from a config file
func setupFAQService(db *database.Postgres, llmPath, chatModel, configPath string, logger *logging.Logger) (*faq.Service, error) {
	// Load FAQ config
	config, err := faq.LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	logger.Info("loaded FAQ config",
		"entries", len(config.Entries),
		"embeddingModel", config.EmbeddingModel,
		"threshold", config.SimilarityThreshold,
	)

	// Create FAQ service
	serviceConfig := faq.ServiceConfig{
		LLMPath:             llmPath,
		EmbeddingModel:      config.EmbeddingModel,
		ChatModel:           chatModel,
		SimilarityThreshold: config.SimilarityThreshold,
		UsePerUserCooldown:  true, // Enable per-user cooldowns
		Logger:              logger,
	}

	return faq.NewService(db.DB(), serviceConfig)
}

// Shutdown cancels the context and logs a message.
// TODO: this needs to be handled with an os signal
func Shutdown(ctx context.Context, wg *sync.WaitGroup, irc *twitchirc.IRC, stop chan os.Signal, logger *logging.Logger,
) {
	<-stop
	ctx.Done()

	if irc != nil {
		err := irc.Client.Disconnect()
		if err != nil {
			logger.Error("error disconnecting twitch client", "error", err.Error())
		}
	}
	// wg.Done()
	logger.Info("Shutting down")
	os.Exit(0)
}
