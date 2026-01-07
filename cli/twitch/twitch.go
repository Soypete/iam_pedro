package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"

	"github.com/Soypete/twitch-llm-bot/ai"
	"github.com/Soypete/twitch-llm-bot/ai/twitchchat"
	database "github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/Soypete/twitch-llm-bot/metrics"
	twitchirc "github.com/Soypete/twitch-llm-bot/twitch"
)

func main() {
	var model string
	var logLevel string
	var streamConfig string
	var modConfigPath string
	var enableModeration bool
	var dryRun bool

	flag.StringVar(&model, "model", os.Getenv("MODEL"), "The model to use for the LLM")
	flag.StringVar(&logLevel, "errorLevel", "info", "Log level (debug, info, warn, error)")
	flag.StringVar(&streamConfig, "streamConfig", "", "Path to stream context config file (e.g., 'configs/streams/golang-nov-2025.yaml')")
	flag.StringVar(&modConfigPath, "modConfig", "", "Path to moderation config file (e.g., 'configs/moderation.yaml')")
	flag.BoolVar(&enableModeration, "enableModeration", false, "Enable chat moderation system")
	flag.BoolVar(&dryRun, "modDryRun", false, "Run moderation in dry-run mode (log actions without executing)")
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

	// Load moderation config if enabled
	var modConfig *ai.ModerationConfig
	if enableModeration || modConfigPath != "" {
		if modConfigPath != "" {
			modConfig, err = ai.LoadModerationConfig(modConfigPath)
			if err != nil {
				logger.Error("failed to load moderation config", "error", err.Error())
				os.Exit(1)
			}
			logger.Info("loaded moderation config", "path", modConfigPath)
		} else {
			// Use default config if no path provided but moderation is enabled
			modConfig = ai.DefaultModerationConfig()
			logger.Info("using default moderation config")
		}

		// Override enabled and dry-run from flags
		if enableModeration {
			modConfig.Enabled = true
		}
		if dryRun {
			modConfig.DryRun = true
		}

		logger.Info("moderation configuration",
			"enabled", modConfig.Enabled,
			"dryRun", modConfig.DryRun,
			"sensitivity", modConfig.SensitivityLevel,
		)
	}

	var irc *twitchirc.IRC
	// setup twitch IRC with optional moderation
	if modConfig != nil && modConfig.Enabled {
		irc, err = twitchirc.SetupTwitchIRCWithModeration(wg, twitchllm, model, db, db, modConfig, logger)
	} else {
		irc, err = twitchirc.SetupTwitchIRC(wg, twitchllm, model, db, logger)
	}
	if err != nil {
		logger.Error("failed to setup twitch IRC", "error", err.Error())
		stop <- os.Interrupt
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
