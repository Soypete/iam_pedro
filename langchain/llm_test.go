package langchain

import (
	"context"
	"testing"

	"github.com/tmc/langchaingo/llms"
)

func Test_cleanResponse(t *testing.T) {
	tests := []struct {
		name string
		resp string
		want string
	}{
		{
			name: "Test 1",
			resp: "Hello\nWorld",
			want: "Hello World",
		},
		{
			name: "Test 2",
			resp: "<|im_start|>user \nTtocsNeb: hi",
			want: "TtocsNeb: hi",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanResponse(tt.resp); got != tt.want {
				t.Errorf("cleanResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

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

type mockDB struct{}

func (m *mockDB) InsertResponse(ctx context.Context, resp *llms.ContentResponse) error {
	return nil
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
				llm: &mockLLM{},
				db:  &mockDB{},
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
			got, err := tt.c.callLLM(tt.args.ctx, tt.args.injection)
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
