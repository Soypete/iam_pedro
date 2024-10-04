package main

import (
	"context"
	"fmt"
	"log"

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

	embedings, err := llm.CreateEmbedding(ctx, []string{prompt})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(embedings)
}
