package discordchat

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Soypete/twitch-llm-bot/duckduckgo"
	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// WebSearchTool implements the tools.Tool interface for DuckDuckGo search
type WebSearchTool struct {
	ddgClient *duckduckgo.Client
}

// NewWebSearchTool creates a new WebSearchTool
func NewWebSearchTool(ddgClient *duckduckgo.Client) *WebSearchTool {
	return &WebSearchTool{ddgClient: ddgClient}
}

// Name returns the name of the tool
func (w *WebSearchTool) Name() string {
	return "web_search"
}

// Description returns a description of the tool
func (w *WebSearchTool) Description() string {
	return "Perform a web search to find current information"
}

// Call performs the web search
func (w *WebSearchTool) Call(ctx context.Context, input string) (string, error) {
	// Perform the search
	searchResult, err := w.ddgClient.Search(input)
	if err != nil {
		return "", err
	}

	// Parse the search result
	var ddgResponse duckduckgo.Response
	err = json.Unmarshal(searchResult, &ddgResponse)
	if err != nil {
		return "", fmt.Errorf("failed to parse search results: %w", err)
	}

	// Extract the first result or abstract
	if len(ddgResponse.Results) > 0 {
		return ddgResponse.Results[0].Text, nil
	}
	if ddgResponse.Abstract != "" {
		return ddgResponse.Abstract, nil
	}
	if len(ddgResponse.RelatedTopics) > 0 {
		return ddgResponse.RelatedTopics[0].Text, nil
	}

	return "No results found", nil
}

// CreateOpenAIFunctionsAgent creates an agent with web search capabilities
func CreateOpenAIFunctionsAgent(
	llm llms.Model, 
	ddgClient *duckduckgo.Client, 
	logger *logging.Logger,
) (tools.Tool, error) {
	// Create web search tool
	webSearchTool := NewWebSearchTool(ddgClient)
	
	return webSearchTool, nil
}