package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	database "github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/langchain"
	twitchirc "github.com/Soypete/twitch-llm-bot/twitch"
)

func main() {
	// read in json file with helpful links and prompts
	// TODOL pass in file path as a flag
	prompts, err := os.ReadFile("links.txt")
	if err != nil {
		log.Fatalln(err)
	}

	// TODO: 120 second timeout is to short. we need a better way to handle this
	ctx := context.Background()
	// setup postgres connection
	db, err := database.NewPostgres()
	if err != nil {
		log.Fatalln(err)
	}
	// setup llm connection
	llm, err := langchain.Setup(db)
	if err != nil {
		log.Fatalln(err)
	}
	if llm == nil {
		log.Fatalln("llm is nil")
	}

	// TODO: audit waitgroup
	wg := sync.WaitGroup{}
	// setup twitch IRC
	irc, err := twitchirc.SetupTwitchIRC(wg, llm, db)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("starting twitch IRC connection")
	// long running function
	err = irc.ConnectIRC(ctx)
	if err != nil {
		panic(err)
	}
	// TODO: break out of the main function
	go func() {
		log.Println("Starting prompt loop")
		// replaces nightbot timers
		// once every 30 minutes prompt the llm to generate a message
		// that message will have the context
		for {
			timeout := 30 * time.Minute
			time.Sleep(timeout)
			// generate prompts
			resp, err := llm.GenerateTimer(ctx, string(prompts))
			if err != nil {
				log.Println(err)
				continue
			}
			if resp == "" {
				log.Println("empty response")
				continue
			}
			// send message to twitch
			irc.Client.Say("soypetetech", resp)
		}
	}()

	// TODO: why is this not in a goroutine?
	err = irc.Client.Connect()
	if err != nil {
		panic(fmt.Errorf("failed to connect to twitch IRC: %w", err))
	}
}

func Shutdown(ctx context.Context, wg *sync.WaitGroup) {
	ctx.Done()
	log.Println("Shutting down")

}
