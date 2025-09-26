package discordchat

import (
	"context"
	"testing"

	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebSearchTool(t *testing.T) {
	// Create a mock DuckDuckGo client
	mockDDGClient := &duckduckgo.Client{
		BaseURL: "https://api.duckduckgo.com",
		HTTPClient: &mockHTTPClient{
			mockSearch: func(query string) ([]byte, error) {
				return []byte(`{
					"Abstract": "Golang best practices overview",
					"Results": [
						{
							"Text": "A comprehensive guide to writing clean Go code"
						}
					]
				}`), nil
			},
		},
	}

	// Create the web search tool
	webSearchTool := NewWebSearchTool(mockDDGClient)

	// Test tool metadata
	assert.Equal(t, "web_search", webSearchTool.Name())
	assert.NotEmpty(t, webSearchTool.Description())

	// Perform search
	result, err := webSearchTool.Call(context.Background(), "golang best practices")

	// Assertions
	require.NoError(t, err)
	assert.Contains(t, result, "A comprehensive guide to writing clean Go code")
}

func TestCreateOpenAIFunctionsAgent(t *testing.T) {
	// Setup dependencies
	logger := logging.Default()
	ddgClient := &duckduckgo.Client{
		BaseURL: "https://api.duckduckgo.com",
		HTTPClient: &mockHTTPClient{
			mockSearch: func(query string) ([]byte, error) {
				return []byte(`{
					"Abstract": "Golang best practices overview",
					"Results": [
						{
							"Text": "A comprehensive guide to writing clean Go code"
						}
					]
				}`), nil
			},
		},
	}

	// Mock LLM 
	mockLLM := &mockLLM{}

	// Create the agent
	agent, err := CreateOpenAIFunctionsAgent(mockLLM, ddgClient, logger)

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, agent)

	// Check that the agent is a web search tool
	assert.Equal(t, "web_search", agent.Name())
	assert.NotEmpty(t, agent.Description())
}

// mockHTTPClient is a mock implementation of HTTP client for DuckDuckGo
type mockHTTPClient struct {
	mockSearch func(query string) ([]byte, error)
}

func (m *mockHTTPClient) Search(query string) ([]byte, error) {
	return m.mockSearch(query)
}

// mockLLM is a mock implementation of the LLM interface for testing
type mockLLM struct{}

func (m *mockLLM) GenerateContent(ctx context.Context, messages any, opts ...any) (any, error) {
	return struct {
		Choices []struct {
			Content string
		}
	}{
		Choices: []struct {
			Content string
		}{
			{
				Content: "Mock LLM response",
			},
		},
	}, nil
}

func (m *mockLLM) Call(ctx context.Context, prompt string, opts ...any) (string, error) {
	return "Mock LLM response", nil
}