package twitchchat

import (
	"context"
	"fmt"
	"testing"

	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"
)

type mockLLM struct{}

func (m *mockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, opts ...llms.CallOption) (*llms.ContentResponse, error) {
	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{
				Content: "Hello World",
			},
		},
	}, nil
}

func (m *mockLLM) Call(ctx context.Context, prompt string, opts ...llms.CallOption) (string, error) {
	return "", nil
}

func TestClient_callLLM(t *testing.T) {
	type args struct {
		ctx       context.Context
		injection []string
	}
	tests := []struct {
		name    string
		c       Client
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "make prompt",
			c: Client{
				llm:    &mockLLM{},
				logger: logging.Default(),
			},
			args: args{
				ctx:       context.Background(),
				injection: []string{"Hello", "World"},
			},
			want:    "Hello World",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.c.callLLM(tt.args.ctx, tt.args.injection, uuid.New())
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.createPrompt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Client.createPrompt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_manageChatHistory(t *testing.T) {
	var ch []llms.MessageContent
	ch = append(ch, llms.TextParts(llms.ChatMessageTypeHuman, "Hello World"))
	ch = append(ch, llms.TextParts(llms.ChatMessageTypeHuman, "my name is Scott"))
	ch = append(ch, llms.TextParts(llms.ChatMessageTypeAI, "I am a bot"))
	ch = append(ch, llms.TextParts(llms.ChatMessageTypeHuman, "This is chat history"))
	ch = append(ch, llms.TextParts(llms.ChatMessageTypeHuman, "I am writing a test"))
	ch = append(ch, llms.TextParts(llms.ChatMessageTypeHuman, "Please work"))
	ch = append(ch, llms.TextParts(llms.ChatMessageTypeAI, "I am a bot"))
	ch = append(ch, llms.TextParts(llms.ChatMessageTypeHuman, "This is chat history"))
	ch = append(ch, llms.TextParts(llms.ChatMessageTypeHuman, "I am writing a test"))
	ch = append(ch, llms.TextParts(llms.ChatMessageTypeHuman, "Please work"))

	chat3 := ch[:3]
	type args struct {
		ctx       context.Context
		injection []string
		chatType  llms.ChatMessageType
	}
	tests := []struct {
		name    string
		client  *Client
		args    args
		wantLen int
	}{
		{
			name: "no chat",
			client: &Client{
				llm:    &mockLLM{},
				logger: logging.Default(),
			},
			args: args{
				ctx:       context.Background(),
				injection: []string{"Hello", "World"},
				chatType:  llms.ChatMessageTypeHuman,
			},
			wantLen: 1,
		},
		{
			name: "some chat",
			client: &Client{
				llm:         &mockLLM{},
				chatHistory: chat3,
				logger:      logging.Default(),
			},
			args: args{
				ctx:       context.Background(),
				injection: []string{"Hello", "World"},
				chatType:  llms.ChatMessageTypeHuman,
			},
			wantLen: 4,
		},
		{
			name: "full chat",
			client: &Client{
				llm:         &mockLLM{},
				chatHistory: ch,
				logger:      logging.Default(),
			},
			args: args{
				ctx:       context.Background(),
				injection: []string{"Hello", "World"},
				chatType:  llms.ChatMessageTypeHuman,
			},
			wantLen: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.client.manageChatHistory(tt.args.ctx, tt.args.injection, tt.args.chatType)
			if len(tt.client.chatHistory) != tt.wantLen {
				fmt.Println(tt.client.chatHistory)
				t.Errorf("Client.manageChatHistory() = %v, want %v", len(tt.client.chatHistory), tt.wantLen)
			}
		})
	}
}
