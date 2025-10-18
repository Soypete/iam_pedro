package discordchat

import (
	"context"
	"fmt"
	"strings"
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
	shouldCallTool   bool
}

func (m *mockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, opts ...llms.CallOption) (*llms.ContentResponse, error) {
	// Check if the last human message looks like it needs a web search
	// This simulates the LLM deciding to call the web_search tool
	needsWebSearch := false
	searchQuery := "golang best practices"

	if len(messages) > 0 {
		lastMsg := messages[len(messages)-1]
		if lastMsg.Role == llms.ChatMessageTypeHuman {
			for _, part := range lastMsg.Parts {
				if textPart, ok := part.(llms.TextContent); ok {
					text := string(textPart.Text)
					// If the message contains "execute web search" in the test input, trigger tool call
					if strings.Contains(text, "execute web search") {
						needsWebSearch = true
						// Extract what comes after "execute web search"
						parts := strings.Split(text, "execute web search")
						if len(parts) > 1 {
							searchQuery = strings.TrimSpace(parts[1])
						}
						break
					}
				}
			}
		}
	}

	// If web search is needed, return a tool call
	if needsWebSearch || m.shouldCallTool {
		return &llms.ContentResponse{
			Choices: []*llms.ContentChoice{
				{
					ToolCalls: []llms.ToolCall{
						{
							FunctionCall: &llms.FunctionCall{
								Name:      "web_search",
								Arguments: fmt.Sprintf(`{"query":"%s"}`, searchQuery),
							},
						},
					},
				},
			},
		}, nil
	}

	// Otherwise return normal content
	content := "Mock LLM response"
	if m.responseOverride != "" {
		content = m.responseOverride
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