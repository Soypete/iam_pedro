package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"

	"github.com/Soypete/twitch-llm-bot/ai/discordchat"
	"github.com/Soypete/twitch-llm-bot/ai/twitchchat"
	database "github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/discord"
	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/Soypete/twitch-llm-bot/metrics"
	twitchirc "github.com/Soypete/twitch-llm-bot/twitch"
)

func main() {
	var model string
	var startDiscord bool
	var startTwitch bool
	var logLevel string

	flag.StringVar(&model, "model", "", "The model to use for the LLM")
	flag.BoolVar(&startDiscord, "discordMode", false, "Start the discord bot")
	flag.BoolVar(&startTwitch, "twitchMode", true, "Start the twitch bot")
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
	twitchllm, err := twitchchat.Setup(llmPath, logger)
	if err != nil {
		logger.Error("failed to setup twitch LLM", "error", err.Error())
		os.Exit(1)
	}

	var session discord.Client
	if startDiscord {
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
	}

	var irc *twitchirc.IRC
	if startTwitch {
		// setup twitch IRC
		irc, err = twitchirc.SetupTwitchIRC(wg, twitchllm, model, db, logger)
		if err != nil {
			logger.Error("failed to setup twitch IRC", "error", err.Error())
			stop <- os.Interrupt
		}
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
	}
	signal.Notify(stop, os.Interrupt)
	logger.Info("Press Ctrl+C to exit")
	Shutdown(ctx, wg, irc, session, stop, logger)
}

// Shutdown cancels the context and logs a message.
// TODO: this needs to be handled with an os signal
func Shutdown(ctx context.Context, wg *sync.WaitGroup,
	irc *twitchirc.IRC, session discord.Client, stop chan os.Signal, logger *logging.Logger,
) {
	<-stop
	ctx.Done()

	if session.Session != nil {
		err := session.Session.Close()
		if err != nil {
			logger.Error("error closing discord session", "error", err.Error())
		}
	}

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
