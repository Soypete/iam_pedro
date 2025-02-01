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
	twitchirc "github.com/Soypete/twitch-llm-bot/twitch"
)

func main() {

	var model string
	flag.StringVar(&model, "model", "Mistral_7B_v0.1.4", "The model to use for the LLM")
	flag.Parse()

	ctx := context.Background()
	stop := make(chan os.Signal, 1)
	wg := &sync.WaitGroup{}

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
	discordllm, err := discordchat.Setup(db, model, llmPath)
	if err != nil {
		log.Fatalln(err)
	}

	session, err := discord.Setup(discordllm)
	if err != nil {
		fmt.Println(err)
		stop <- os.Interrupt
	}

	go Shutdown(ctx, wg, session, stop)
	wg.Add(1)

	// setup twitch IRC
	irc, err := twitchirc.SetupTwitchIRC(wg, twitchllm, db)
	if err != nil {
		fmt.Println(err)
		stop <- os.Interrupt
	}
	log.Println("starting twitch IRC connection")
	// long running function
	err = irc.ConnectIRC(ctx)
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

	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	wg.Wait()
}

// Shutdown cancels the context and logs a message.
// TODO: this needs to be handled with an os signal
func Shutdown(ctx context.Context, wg *sync.WaitGroup, session discord.Client, stop chan os.Signal) {
	<-stop
	ctx.Done()
	session.Session.Close()
	wg.Done()
	log.Println("Shutting down")
}
