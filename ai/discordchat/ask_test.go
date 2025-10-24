package discordchat

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Soypete/twitch-llm-bot/duckduckgo"
	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/Soypete/twitch-llm-bot/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func TestSingleMessageResponse(t *testing.T) {
	// Setup dependencies
	logger := logging.Default()
	ddgClient := duckduckgo.NewClient()

	// Create bot with mocked dependencies
	bot := &Bot{
		llm:         &mockLLM{},
		modelName:   "test-model",
		logger:      logger,
		ddgClient:   ddgClient,
		agent:       &mockWebSearchTool{},
		chatHistory: []llms.MessageContent{},
	}

	testCases := []struct {
		name           string
		inputMessage   types.DiscordAskMessage
		expectedOutput string
		webSearchCall  bool
	}{
		{
			name: "Normal message response",
			inputMessage: types.DiscordAskMessage{
				Username: "testuser",
				Message:  "Hello Pedro!",
				ThreadID: "thread123",
			},
			expectedOutput: "Mock LLM response",
			webSearchCall:  false,
		},
		{
			name: "Web search trigger",
			inputMessage: types.DiscordAskMessage{
				Username: "testuser",
				Message:  "execute web search golang best practices",
				ThreadID: "thread456",
			},
			expectedOutput: "one second and I will look that up for you :thinking:",
			webSearchCall:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Perform the test
			resp, err := bot.SingleMessageResponse(context.Background(), tc.inputMessage)

			// Assertions
			require.NoError(t, err)
			require.NotNil(t, resp)

			if tc.webSearchCall {
				assert.Equal(t, tc.expectedOutput, resp.Text)
				require.NotNil(t, resp.WebSearch)
				assert.Contains(t, resp.WebSearch.Query, "golang best practices")
			} else {
				assert.Equal(t, tc.expectedOutput, resp.Text)
			}
		})
	}
}

func TestExecuteWebSearch(t *testing.T) {
	// Setup dependencies
	logger := logging.Default()
	ddgClient := duckduckgo.NewClient()

	// Create bot with mocked dependencies
	bot := &Bot{
		llm:         &mockLLM{},
		modelName:   "test-model",
		logger:      logger,
		ddgClient:   ddgClient,
		agent:       &mockWebSearchTool{},
		chatHistory: []llms.MessageContent{},
	}

	// Prepare web search request
	searchRequest := &types.WebSearchRequest{
		Query: "golang best practices",
		ChatHistory: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Tell me about Golang best practices"),
		},
	}

	// Perform web search
	result, err := bot.ExecuteWebSearch(context.Background(), searchRequest)

	// Assertions
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

// mockWebSearchTool is a mock implementation of the web search tool
type mockWebSearchTool struct{}

func (m *mockWebSearchTool) Name() string {
	return "web_search"
}

func (m *mockWebSearchTool) Description() string {
	return "Mock web search tool for testing"
}

func (m *mockWebSearchTool) Call(ctx context.Context, input string) (string, error) {
	return "Mock search result for: " + input, nil
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