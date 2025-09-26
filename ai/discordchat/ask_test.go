package discordchat

import (
	"context"
	"testing"

	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/Soypete/twitch-llm-bot/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func TestSingleMessageResponse(t *testing.T) {
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