package llms

import "context"

type ChatMessageType string

const (
	ChatMessageTypeSystem ChatMessageType = "system"
	ChatMessageTypeHuman  ChatMessageType = "user"
	ChatMessageTypeAI     ChatMessageType = "assistant"
)

type MessageContent struct {
	Type ChatMessageType `json:"role"`
	Text string          `json:"content"`
}

type GenerateOptions struct {
	CandidateCount   int      `json:"n,omitempty"`
	MaxLength        int      `json:"max_tokens,omitempty"`
	Temperature      float64  `json:"temperature,omitempty"`
	PresencePenalty  float64  `json:"presence_penalty,omitempty"`
	StopWords        []string `json:"stop,omitempty"`
}

type Choice struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type GenerateResponse struct {
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Model defines the interface for LLM providers
type Model interface {
	GenerateContent(ctx context.Context, messages []MessageContent, opts ...GenerateOption) (*GenerateResponse, error)
}

type GenerateOption func(*GenerateOptions)

func WithCandidateCount(count int) GenerateOption {
	return func(opts *GenerateOptions) {
		opts.CandidateCount = count
	}
}

func WithMaxLength(length int) GenerateOption {
	return func(opts *GenerateOptions) {
		opts.MaxLength = length
	}
}

func WithTemperature(temp float64) GenerateOption {
	return func(opts *GenerateOptions) {
		opts.Temperature = temp
	}
}

func WithPresencePenalty(penalty float64) GenerateOption {
	return func(opts *GenerateOptions) {
		opts.PresencePenalty = penalty
	}
}

func WithStopWords(words []string) GenerateOption {
	return func(opts *GenerateOptions) {
		opts.StopWords = words
	}
}

func TextParts(msgType ChatMessageType, text string) MessageContent {
	return MessageContent{
		Type: msgType,
		Text: text,
	}
}