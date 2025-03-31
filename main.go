package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/Soypete/twitch-llm-bot/ai/discordchat"
	"github.com/Soypete/twitch-llm-bot/ai/twitchchat"
	database "github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/discord"
	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/Soypete/twitch-llm-bot/secrets"
	twitchirc "github.com/Soypete/twitch-llm-bot/twitch"
)

func main() {

	var model string
	var startDiscord bool
	var startTwitch bool
	flag.StringVar(&model, "model", "Mistral_7B_v0.1.4", "The model to use for the LLM")
	flag.BoolVar(&startDiscord, "discordMode", false, "Start the discord bot")
	flag.BoolVar(&startTwitch, "twitchMode", true, "Start the twitch bot")
	flag.Parse()

	ctx := context.Background()

	stop := make(chan os.Signal, 1)
	wg := &sync.WaitGroup{}

	// setup secrets from 1password
	println("Loading secrets")
	secrets.Init()

	// listen and serve for metrics server.
	// TODO: change these configs to file
	server := metrics.SetupServer()
	go server.Run()

	// setup postgres connection
	// change these configs to file
	db, err := database.NewPostgres()
	if err != nil {
		log.Fatalln(err)
	}

	// setup llm connection
	//  we are not actually connecting to openai, but we are using their api spec to connect to our own model via llama.cpp
	os.Setenv("OPENAI_API_KEY", "none")
	llmPath := os.Getenv("LLAMA_CPP_PATH")
	twitchllm, err := twitchchat.Setup(db, model, llmPath)
	if err != nil {
		log.Fatalln(err)
	}

	var session discord.Client
	if startDiscord {
		discordllm, err := discordchat.Setup(db, model, llmPath)
		if err != nil {
			log.Fatalln(err)
		}

		session, err = discord.Setup(discordllm, db)
		if err != nil {
			fmt.Println(err)
			stop <- os.Interrupt
		}

	}

	var irc *twitchirc.IRC
	if startTwitch {
		// setup twitch IRC
		irc, err = twitchirc.SetupTwitchIRC(wg, twitchllm, db)
		if err != nil {
			fmt.Println(err)
			stop <- os.Interrupt
		}
		log.Println("starting twitch IRC connection")
		// long running function
		err = irc.ConnectIRC(ctx, wg)
		if err != nil {
			fmt.Println(err)
			stop <- os.Interrupt
		}

		go func() {
			err = irc.Client.Connect()
			if err != nil {
				fmt.Println(err)
				stop <- os.Interrupt
			}
		}()
	}
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	Shutdown(ctx, wg, irc, session, stop)
}

// Shutdown cancels the context and logs a message.
// TODO: this needs to be handled with an os signal
func Shutdown(ctx context.Context, wg *sync.WaitGroup,
	irc *twitchirc.IRC, session discord.Client, stop chan os.Signal) {
	<-stop
	ctx.Done()

	if session.Session != nil {
		log.Println(session.Session.Close()) // print error if any
	}

	if irc != nil {
		log.Println(irc.Client.Disconnect()) // print error if any
	}
	// wg.Done()
	log.Println("Shutting down")
	os.Exit(0)
}
