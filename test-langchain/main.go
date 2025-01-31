//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/Soypete/twitch-llm-bot/discord"
	"github.com/tmc/langchaingo/llms/openai"
)

func main() {
	ctx := context.Background()
	// TODO: add options
	opts := []openai.Option{
		openai.WithBaseURL("http://127.0.0.1:8080"),
	}
	llm, err := openai.New(opts...)
	if err != nil {
		log.Fatal(err)
	}
	prompt := "What color is the sky?"
	response, err := llm.Call(ctx, prompt)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response)

	session, err := discord.Setup()
	if err != nil {
		log.Fatal(err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop

	err = session.Session.Close()
	if err != nil {
		log.Fatal(err)
	}
}
