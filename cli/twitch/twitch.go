package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"

	"github.com/Soypete/twitch-llm-bot/ai/twitchchat"
	database "github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/Soypete/twitch-llm-bot/metrics"
	twitchirc "github.com/Soypete/twitch-llm-bot/twitch"
)

func main() {
	var model string
	var logLevel string

	flag.StringVar(&model, "model", os.Getenv("MODEL"), "The model to use for the LLM")
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
	_ = os.Setenv("OPENAI_API_KEY", "test")
	llmPath := os.Getenv("LLAMA_CPP_PATH")
	twitchllm, err := twitchchat.Setup(llmPath, model, logger)
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
