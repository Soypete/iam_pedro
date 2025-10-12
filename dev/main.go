package main

// curl -H "User-Agent: twitch-llm-bot/1.0" \
//        "https://api.duckduckgo.com/?q=Go%20programming%20language&format=json&no_html=1&skip_disambig=1"
import (
	"fmt"
	"log"

	"github.com/Soypete/twitch-llm-bot/duckduckgo"
)

func main() {
	// Create a new DuckDuckGo client
	client := duckduckgo.NewClient()

	// Test search query
	query := "What is today's date?"
	fmt.Printf("Searching for: %s\n\n", query)

	// Perform the search
	response, err := client.Search(query)
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}

	fmt.Println(string(response))
}
