package discordchat

import (
	"context"
	"testing"

	"github.com/Soypete/twitch-llm-bot/duckduckgo"
	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func TestWebSearchTool(t *testing.T) {
	// Create a mock DuckDuckGo client
	mockDDGClient := duckduckgo.NewClient()

	// Create the web search tool
	webSearchTool := NewWebSearchTool(mockDDGClient)

	// Test tool metadata
	assert.Equal(t, "web_search", webSearchTool.Name())
	assert.NotEmpty(t, webSearchTool.Description())
}

func TestCreateOpenAIFunctionsAgent(t *testing.T) {
	// Setup dependencies
	logger := logging.Default()
	ddgClient := duckduckgo.NewClient()

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

// mockLLM is a mock implementation of the LLM interface for testing
type mockLLM struct {
	responseOverride string
}

func (m *mockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, opts ...llms.CallOption) (*llms.ContentResponse, error) {
	content := "Mock LLM response"

	// If there's a response override, use it
	if m.responseOverride != "" {
		content = m.responseOverride
	} else if len(messages) > 0 {
		// Check only the last human message for web search triggers
		lastMsg := messages[len(messages)-1]
		if lastMsg.Role == llms.ChatMessageTypeHuman {
			for _, part := range lastMsg.Parts {
				if textPart, ok := part.(llms.TextContent); ok {
					text := string(textPart.Text)
					// If input message contains "execute web search", return that phrase
					// to trigger the web search logic in the bot
					if len(text) >= 18 {
						// Check if it contains the trigger phrase
						for i := 0; i <= len(text)-18; i++ {
							if text[i:i+18] == "execute web search" {
								// Return the trigger phrase so the bot knows to search
								content = "execute web search " + text[i+18:]
								break
							}
						}
					}
				}
			}
		}
	}

	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{
				Content: content,
			},
		},
	}, nil
}

func (m *mockLLM) Call(ctx context.Context, prompt string, opts ...llms.CallOption) (string, error) {
	return "Mock LLM response", nil
}