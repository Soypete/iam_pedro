package llamacpp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Soypete/twitch-llm-bot/llms"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	model      string
}

type GenerateRequest struct {
	Messages []llms.MessageContent `json:"messages"`
	Model    string                `json:"model,omitempty"`
	llms.GenerateOptions
}

func New(baseURL string, opts ...Option) (*Client, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("baseURL cannot be empty")
	}

	client := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		model: "default",
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

type Option func(*Client)

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

func WithModel(model string) Option {
	return func(c *Client) {
		c.model = model
	}
}

func (c *Client) GenerateContent(ctx context.Context, messages []llms.MessageContent, opts ...llms.GenerateOption) (*llms.GenerateResponse, error) {
	options := &llms.GenerateOptions{}
	for _, opt := range opts {
		opt(options)
	}

	request := GenerateRequest{
		Messages:        messages,
		Model:           c.model,
		GenerateOptions: *options,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Construct the URL for chat completions endpoint
	url := fmt.Sprintf("%s/v1/chat/completions", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var response llms.GenerateResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Ensure we have at least one choice
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from API")
	}

	return &response, nil
}