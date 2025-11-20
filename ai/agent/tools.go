package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Soypete/twitch-llm-bot/duckduckgo"
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

// GetWebSearchToolDefinition returns the LLM tool definition for web search
func GetWebSearchToolDefinition() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "web_search",
			Description: "Search the web for current information, news, or recent events",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "The search query to look up",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

// ToolCallArgs represents the parsed arguments from a web search tool call
type ToolCallArgs struct {
	Query string `json:"query"`
}

// ParseWebSearchToolCall parses a tool call and returns the search query
// Returns the query string and an error if parsing fails
func ParseWebSearchToolCall(toolCall llms.ToolCall) (string, error) {
	if toolCall.FunctionCall.Name != "web_search" {
		return "", fmt.Errorf("unexpected tool call: %s", toolCall.FunctionCall.Name)
	}

	var args ToolCallArgs
	if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse tool call arguments: %w", err)
	}

	if args.Query == "" {
		return "", fmt.Errorf("query cannot be empty")
	}

	return args.Query, nil
}

// CreateWebSearchAgent creates an agent with web search capabilities
// This is a compatibility function that returns the web search tool
func CreateWebSearchAgent(ddgClient *duckduckgo.Client) tools.Tool {
	return NewWebSearchTool(ddgClient)
}
