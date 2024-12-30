package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync"

	database "github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/langchain"
	"github.com/Soypete/twitch-llm-bot/metrics"
	twitchirc "github.com/Soypete/twitch-llm-bot/twitch"
)

func main() {

	var model string
	flag.StringVar(&model, "model", "Mistral_7B_v0.1.4", "The model to use for the LLM")
	flag.Parse()

	// listen and serve for metrics server.
	// TODO: change these configs to file
	server := metrics.SetupServer()
	go server.Run()

	ctx := context.Background()
	// setup postgres connection
	// change these configs to file
	db, err := database.NewPostgres()
	if err != nil {
		log.Fatalln(err)
	}
	// setup llm connection
	llm, err := langchain.Setup(db, model)
	if err != nil {
		log.Fatalln(err)
	}
	// TODO: audit waitgroup
	wg := sync.WaitGroup{}
	// setup twitch IRC
	irc, err := twitchirc.SetupTwitchIRC(wg, llm, db)
	if err != nil {
		Shutdown(ctx, &wg)
		log.Fatalln(err)
	}
	log.Println("starting twitch IRC connection")
	// long running function
	err = irc.ConnectIRC(ctx)
	if err != nil {
		Shutdown(ctx, &wg)
		panic(err)
	}

	//TODO: make channel for when twitch chat is active
	// TODO: why is this not in a goroutine?
	err = irc.Client.Connect()
	if err != nil {
		Shutdown(ctx, &wg)
		panic(fmt.Errorf("failed to connect to twitch IRC: %w", err))
	}
}

// Shutdown cancels the context and logs a message.
// TODO: this needs to be handled with an os signal
func Shutdown(ctx context.Context, wg *sync.WaitGroup) {
	ctx.Done()
	log.Println("Shutting down")
}
