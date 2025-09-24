package discordchat

import (
	"context"
	"testing"

	"github.com/Soypete/twitch-llm-bot/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tmc/langchaingo/llms"
)

// MockLLM is a mock implementation of the llms.Model interface
type MockLLM struct {
	mock.Mock
}

func (m *MockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	args := m.Called(ctx, messages, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llms.ContentResponse), args.Error(1)
}

func (m *MockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	args := m.Called(ctx, prompt, options)
	return args.String(0), args.Error(1)
}

func TestConversationResponse(t *testing.T) {
	tests := []struct {
		name            string
		messages        []types.DiscordAskMessage
		newMessage      string
		mockResponse    *llms.ContentResponse
		mockError       error
		expectedResult  string
		expectedError   bool
	}{
		{
			name: "successful conversation with history",
			messages: []types.DiscordAskMessage{
				{
					Message:     "Hello Pedro",
					Username:    "testuser",
					IsFromPedro: false,
				},
				{
					Message:     "Hello! How can I help you?",
					Username:    "Pedro",
					IsFromPedro: true,
				},
			},
			newMessage: "What's the weather like?",
			mockResponse: &llms.ContentResponse{
				Choices: []*llms.ContentChoice{
					{
						Content: "I'm a chat bot and don't have access to weather data, but I can help with other questions!",
					},
				},
			},
			mockError:      nil,
			expectedResult: "I'm a chat bot and don't have access to weather data, but I can help with other questions!",
			expectedError:  false,
		},
		{
			name:       "empty conversation history",
			messages:   []types.DiscordAskMessage{},
			newMessage: "Hello Pedro",
			mockResponse: &llms.ContentResponse{
				Choices: []*llms.ContentChoice{
					{
						Content: "Hello! I'm Pedro, how can I help you today?",
					},
				},
			},
			mockError:      nil,
			expectedResult: "Hello! I'm Pedro, how can I help you today?",
			expectedError:  false,
		},
		{
			name: "empty response from LLM",
			messages: []types.DiscordAskMessage{
				{
					Message:     "Test",
					Username:    "user",
					IsFromPedro: false,
				},
			},
			newMessage: "Another test",
			mockResponse: &llms.ContentResponse{
				Choices: []*llms.ContentChoice{
					{
						Content: "",
					},
				},
			},
			mockError:      nil,
			expectedResult: "Sorry, I cannot respond to that. Please try again",
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := new(MockLLM)
			bot := &Bot{
				llm:    mockLLM,
				logger: nil, // Will use default logger
			}

			// Set up mock expectations
			if tt.mockResponse != nil || tt.mockError != nil {
				mockLLM.On("GenerateContent", mock.Anything, mock.Anything, mock.Anything).
					Return(tt.mockResponse, tt.mockError)
			}

			// Execute
			result, err := bot.ConversationResponse(context.Background(), tt.messages, tt.newMessage)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestSingleMessageResponse(t *testing.T) {
	tests := []struct {
		name           string
		message        types.DiscordAskMessage
		mockResponse   *llms.ContentResponse
		mockError      error
		expectedResult string
		expectedError  bool
	}{
		{
			name: "successful single message",
			message: types.DiscordAskMessage{
				Message:  "What is Go?",
				Username: "testuser",
			},
			mockResponse: &llms.ContentResponse{
				Choices: []*llms.ContentChoice{
					{
						Content: "Go is a programming language developed by Google.",
					},
				},
			},
			mockError:      nil,
			expectedResult: "Go is a programming language developed by Google.",
			expectedError:  false,
		},
		{
			name: "empty response handling",
			message: types.DiscordAskMessage{
				Message:  "Invalid input",
				Username: "testuser",
			},
			mockResponse: &llms.ContentResponse{
				Choices: []*llms.ContentChoice{
					{
						Content: "",
					},
				},
			},
			mockError:      nil,
			expectedResult: "sorry, I cannot respond to @testuser. Please try again",
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := new(MockLLM)
			bot := &Bot{
				llm:    mockLLM,
				logger: nil, // Will use default logger
			}

			// Set up mock expectations
			if tt.mockResponse != nil || tt.mockError != nil {
				mockLLM.On("GenerateContent", mock.Anything, mock.Anything, mock.Anything).
					Return(tt.mockResponse, tt.mockError)
			}

			// Execute
			result, err := bot.SingleMessageResponse(context.Background(), tt.message)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}