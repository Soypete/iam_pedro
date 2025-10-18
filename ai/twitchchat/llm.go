package twitchchat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Soypete/twitch-llm-bot/ai"
	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/Soypete/twitch-llm-bot/types"
	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"
)

func (c *Client) manageChatHistory(ctx context.Context, injection []string, chatType llms.ChatMessageType) {
	c.logger.Debug("managing chat history", "type", chatType, "message", strings.Join(injection, " "))

	if len(c.chatHistory) >= 10 {
		c.logger.Debug("pruning chat history", "old_size", len(c.chatHistory))
		c.chatHistory = c.chatHistory[1:]
	}
	c.chatHistory = append(c.chatHistory, llms.TextParts(chatType, strings.Join(injection, " ")))
	c.logger.Debug("updated chat history", "new_size", len(c.chatHistory))
}

func (c *Client) callLLM(ctx context.Context, injection []string, messageID uuid.UUID) (*llms.ContentResponse, error) {
	c.logger.Debug("calling LLM", "message", strings.Join(injection, " "), "messageID", messageID)

	now := time.Now().Format(time.DateOnly)
	c.manageChatHistory(ctx, injection, llms.ChatMessageTypeHuman)
	messageHistory := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, fmt.Sprintf(ai.PedroPrompt, now))}
	messageHistory = append(messageHistory, c.chatHistory...)

	// Define the web search tool for function calling
	toolDefinition := llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "web_search",
			Description: "Search the web for current information, news, or recent events",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "The search query to look up",
					},
				},
				"required": []string{"query"},
			},
		},
	}

	c.logger.Debug("generating content", "historyLength", len(messageHistory))
	resp, err := c.llm.GenerateContent(ctx, messageHistory,
		llms.WithCandidateCount(1),
		llms.WithMaxLength(500),
		llms.WithTemperature(0.7),
		llms.WithPresencePenalty(1.0),
		llms.WithStopWords([]string{"LUL, PogChamp, Kappa, KappaPride, KappaRoss, KappaWealth"}),
		llms.WithTools([]llms.Tool{toolDefinition}))
	if err != nil {
		c.logger.Error("failed to get LLM response", "error", err.Error())
		return nil, fmt.Errorf("failed to get llm response: %w", err)
	}

	c.logger.Debug("successfully generated response", "messageID", messageID)
	return resp, nil
}

// SingleMessageResponse is a response from the LLM model to a single message, but to work it needs to have context of chat history
func (c *Client) SingleMessageResponse(ctx context.Context, msg types.TwitchMessage, messageID uuid.UUID) (types.TwitchMessage, error) {
	c.logger.Debug("processing single message response", "messageID", messageID)

	resp, err := c.callLLM(ctx, []string{fmt.Sprintf("%s: %s", msg.Username, msg.Text)}, messageID)
	if err != nil {
		c.logger.Error("failed to generate response", "error", err.Error(), "messageID", messageID)
		metrics.FailedLLMGen.Add(1)
		return types.TwitchMessage{}, err
	}

	// Check if the LLM wants to call a tool
	if len(resp.Choices) > 0 && len(resp.Choices[0].ToolCalls) > 0 {
		toolCall := resp.Choices[0].ToolCalls[0]
		c.logger.Debug("tool call requested", "function", toolCall.FunctionCall.Name, "messageID", messageID)

		if toolCall.FunctionCall.Name == "web_search" {
			// Extract the query from the tool call arguments
			var args struct {
				Query string `json:"query"`
			}
			if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args); err != nil {
				c.logger.Error("failed to parse tool call arguments", "error", err.Error())
				return types.TwitchMessage{
					Text: "Sorry, I had trouble understanding the search request soypet2ConfusedPedro",
					UUID: messageID,
				}, nil
			}

			c.logger.Debug("web search requested via tool call", "query", args.Query, "messageID", messageID)
			msg.UUID = messageID
			return types.TwitchMessage{
				Text: "one second and I will look that up for you soypet2Thinking",
				UUID: messageID,
				WebSearch: &types.WebSearchRequest{
					Query:       args.Query,
					OriginalMsg: msg,
					ChatHistory: c.chatHistory,
				},
			}, nil
		}
	}

	// No tool call, process the text response
	prompt := ai.CleanResponse(resp.Choices[0].Content)
	if prompt == "" {
		c.logger.Warn("empty response from LLM", "messageID", messageID)
		metrics.EmptyLLMResponse.Add(1)
		return types.TwitchMessage{
			Text: fmt.Sprintf("sorry, I cannot respond to @%s. Please try again", msg.Username),
			UUID: messageID,
		}, nil
	}

	// Add Pedro's response to chat history
	c.manageChatHistory(ctx, []string{prompt}, llms.ChatMessageTypeAI)

	c.logger.Debug("successful response generation", "messageID", messageID, "messageLength", len(prompt))
	metrics.SuccessfulLLMGen.Add(1)
	return types.TwitchMessage{
		Text: prompt,
		UUID: messageID,
	}, nil
}

// End20Questions is a response from the LLM model to end the game of 20 questions
func (c *Client) End20Questions() {
	c.logger.Debug("ending 20 questions game")
}

// Play20Questions is a response from the LLM model to a game of 20 questions
func (c *Client) Play20Questions(ctx context.Context, msg types.TwitchMessage, messageID uuid.UUID) (string, error) {
	c.logger.Debug("play 20 questions called but not implemented for twitch", "messageID", messageID)
	return "", nil
}

// ExecuteWebSearch performs a web search and generates a response based on the results
func (c *Client) ExecuteWebSearch(ctx context.Context, request *types.WebSearchRequest, responseChan chan<- types.TwitchMessage) {
	c.logger.Debug("executing web search", "query", request.Query, "originalMessageID", request.OriginalMsg.UUID)

	// Perform the search
	searchResult, err := c.ddgClient.Search(request.Query)
	if err != nil {
		c.logger.Error("web search failed", "error", err.Error(), "query", request.Query, "messageID", request.OriginalMsg.UUID)
		metrics.WebSearchFailCount.Add(1)
		responseChan <- types.TwitchMessage{
			Text: "Sorry, I couldn't search for that information right now soypet2ConfusedPedro",
			UUID: request.OriginalMsg.UUID,
		}
		return
	}
	
	metrics.WebSearchSuccessCount.Add(1)
	c.logger.Debug("web search successful", "query", request.Query, "messageID", request.OriginalMsg.UUID)

	now := time.Now().Format(time.DateOnly)
	messageHistory := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, fmt.Sprintf(ai.PedroPrompt, now))}
	// messageHistory = append(messageHistory, request.ChatHistory...)
	messageHistory = append(messageHistory, llms.TextParts(llms.ChatMessageTypeSystem,
		fmt.Sprintf("Pedro, we have called the duckduckgo search api and the following is the json formatted response: %s. Please provide a helpful summary to the user's question. If you still cannot answer, apologize and ask them to try again.", searchResult)))
	messageHistory = append(messageHistory, llms.TextParts(llms.ChatMessageTypeHuman, request.Query))

	resp, err := c.llm.GenerateContent(ctx, messageHistory,
		llms.WithCandidateCount(1),
		llms.WithMaxLength(500),
		llms.WithTemperature(0.7),
		llms.WithPresencePenalty(1.0))
	if err != nil {
		c.logger.Error("failed to generate response with search results", "error", err.Error())
		responseChan <- types.TwitchMessage{
			Text: "Sorry, I found the information but couldn't process it soypet2ConfusedPedro",
			UUID: request.OriginalMsg.UUID,
		}
		return
	}

	cleanedResponse := ai.CleanResponse(resp.Choices[0].Content)
	if cleanedResponse == "" {
		c.logger.Error("llm returned empty response", "responselen", len(responseChan))
	}

	// Update chat history with the search-informed response
	c.manageChatHistory(ctx, []string{cleanedResponse}, llms.ChatMessageTypeAI)

	c.logger.Debug("sending web search response", "messageID", request.OriginalMsg.UUID, "responseLength", len(cleanedResponse))
	responseChan <- types.TwitchMessage{
		Text: cleanedResponse,
		UUID: request.OriginalMsg.UUID,
	}
}
