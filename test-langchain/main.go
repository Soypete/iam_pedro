package main

import (
	"log"

	"github.com/Soypete/twitch-llm-bot/langchain"
)

func main() {
	llm := langchain.Client{}

	err := llm.MakeVectorStore()
	if err != nil {
		log.Fatalln(err)
	}
}
