package discordchat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Soypete/twitch-llm-bot/ai"
	"github.com/Soypete/twitch-llm-bot/ai/agent"
	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/Soypete/twitch-llm-bot/types"
	"github.com/tmc/langchaingo/llms"
)

const askPedroPrompt = "Your name is Pedro. You are a chat bot that helps out in SoyPeteTech's discord server. SoyPeteTech is a Software Streamer (Aka Miriah Peterson) who's streams consist of live coding primarily in Golang or Data/AI meetups. She also published shorts to tiktok, videos to youtube, and blogs to substack. Your code is found at https://github.com/SoyPete/IamPedro. All other links are on https://linktr.ee/soypete_tech. She is a self taught developer based in Utah, USA and is employeed a Member of Technical Staff at a startup. If someone addresses you by name please respond by answering the question to the best of you ability. You can use code to express fun messages about software. If you are unsure about current events, news, or need to look up recent information, you can use the web_search tool to find up-to-date information. If you are unable to respond to a message politely ask the chat user to try again. If the chat user is being rude or inappropriate please ignore them. Keep your responses fun and engaging. Here are some approved emotes Do not talk about Java or Javascript! Have fun!"

// SingleMessageResponse is a response from the LLM model to a single message
func (b *Bot) SingleMessageResponse(ctx context.Context, msg types.DiscordAskMessage) (*types.DiscordResponse, error) {
	b.logger.Debug("processing discord single message response", "messageID", msg.ThreadID)

	// Manage chat history
	b.manageChatHistory(ctx, []string{fmt.Sprintf("%s: %s", msg.Username, msg.Message)}, llms.ChatMessageTypeHuman)

	now := time.Now().Format(time.DateOnly)
	messageHistory := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, fmt.Sprintf(ai.PedroPrompt, now))}
	messageHistory = append(messageHistory, b.chatHistory...)

	b.logger.Debug("calling LLM for discord message", "messageID", msg.ThreadID, "model", b.modelName)

	// Get web search tool definition from shared agent package
	toolDefinition := agent.GetWebSearchToolDefinition()

	resp, err := b.llm.GenerateContent(ctx, messageHistory,
		llms.WithModel(b.modelName),
		llms.WithCandidateCount(1),
		llms.WithMaxLength(500),
		llms.WithTemperature(0.7),
		llms.WithPresencePenalty(1.0),
		llms.WithStopWords([]string{"@pedro", "@Pedro", "@PedroAI", "@PedroAI_"}),
		llms.WithTools([]llms.Tool{toolDefinition}))
	if err != nil {
		b.logger.Error("failed to get discord LLM response", "error", err.Error(), "messageID", msg.ThreadID)
		metrics.FailedLLMGenCount.Add(1)
		return nil, fmt.Errorf("failed to get llm response: %w", err)
	}

	// Check if the LLM wants to call a tool
	if len(resp.Choices) > 0 && len(resp.Choices[0].ToolCalls) > 0 {
		toolCall := resp.Choices[0].ToolCalls[0]
		b.logger.Debug("tool call requested", "function", toolCall.FunctionCall.Name, "messageID", msg.ThreadID)

		if toolCall.FunctionCall.Name == "web_search" {
			// Parse the tool call using shared agent package
			query, err := agent.ParseWebSearchToolCall(toolCall)
			if err != nil {
				b.logger.Error("failed to parse tool call arguments", "error", err.Error())
				return &types.DiscordResponse{
					Text: "Sorry, I had trouble understanding the search request :confused:",
				}, nil
			}

			b.logger.Debug("web search requested via tool call", "query", query, "messageID", msg.ThreadID)
			return &types.DiscordResponse{
				Text: "one second and I will look that up for you :thinking:",
				WebSearch: &types.WebSearchRequest{
					Query:       query,
					ChatHistory: b.chatHistory,
				},
			}, nil
		}
	}

	// No tool call, just return the text response
	prompt := ai.CleanResponse(resp.Choices[0].Content)
	b.logger.Debug("received discord LLM response", "messageID", msg.ThreadID, "responseLength", len(prompt))

	if prompt == "" {
		b.logger.Warn("empty response from discord LLM", "messageID", msg.ThreadID)
		metrics.EmptyLLMResponseCount.Add(1)
		// We are trying to tag the user to get them to try again with a better prompt.
		return &types.DiscordResponse{
			Text: fmt.Sprintf("sorry, I cannot respond to @%s. Please try again", msg.Username),
		}, nil
	}

	// Add Pedro's response to chat history
	b.manageChatHistory(ctx, []string{prompt}, llms.ChatMessageTypeAI)

	b.logger.Debug("successful discord response generation", "messageID", msg.ThreadID, "messageLength", len(prompt))
	metrics.SuccessfulLLMGenCount.Add(1)
	return &types.DiscordResponse{
		Text: prompt,
	}, nil
}

func (b *Bot) manageChatHistory(ctx context.Context, injection []string, chatType llms.ChatMessageType) {
	b.logger.Debug("managing chat history", "type", chatType, "message", strings.Join(injection, " "))

	if len(b.chatHistory) >= 10 {
		b.logger.Debug("pruning chat history", "old_size", len(b.chatHistory))
		b.chatHistory = b.chatHistory[1:]
	}
	b.chatHistory = append(b.chatHistory, llms.TextParts(chatType, strings.Join(injection, " ")))
	b.logger.Debug("updated chat history", "new_size", len(b.chatHistory))
}

// ExecuteWebSearch performs a web search and generates a response based on the results
func (b *Bot) ExecuteWebSearch(ctx context.Context, request *types.WebSearchRequest) (string, error) {
	b.logger.Debug("executing web search", "query", request.Query)

	// Use the agent to perform the web search
	agentResult, err := b.agent.Call(ctx, request.Query)
	if err != nil {
		b.logger.Error("web search failed", "error", err.Error(), "query", request.Query)
		metrics.WebSearchFailCount.Add(1)
		return "Sorry, I couldn't search for that information right now :confused:", nil
	}
	
	metrics.WebSearchSuccessCount.Add(1)
	b.logger.Debug("web search successful", "query", request.Query, "result", agentResult)

	now := time.Now().Format(time.DateOnly)
	messageHistory := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, fmt.Sprintf(ai.PedroPrompt, now))}
	messageHistory = append(messageHistory, llms.TextParts(llms.ChatMessageTypeSystem,
		fmt.Sprintf("Pedro, the web search returned the following result: %s. Please provide a helpful summary to the user's question. If you still cannot answer, apologize and ask them to try again.", agentResult)))
	messageHistory = append(messageHistory, llms.TextParts(llms.ChatMessageTypeHuman, request.Query))

	resp, err := b.llm.GenerateContent(ctx, messageHistory,
		llms.WithModel(b.modelName),
		llms.WithCandidateCount(1),
		llms.WithMaxLength(500),
		llms.WithTemperature(0.7),
		llms.WithPresencePenalty(1.0))
	if err != nil {
		b.logger.Error("failed to generate response with search results", "error", err.Error())
		return "Sorry, I found the information but couldn't process it :confused:", nil
	}

	cleanedResponse := ai.CleanResponse(resp.Choices[0].Content)
	if cleanedResponse == "" {
		b.logger.Error("llm returned empty response")
		return "Sorry, I couldn't process the search results :confused:", nil
	}

	// Update chat history with the search-informed response
	b.manageChatHistory(ctx, []string{cleanedResponse}, llms.ChatMessageTypeAI)

	b.logger.Debug("sending web search response", "responseLength", len(cleanedResponse))
	return cleanedResponse, nil
}

// ThreadMessageResponse is a response from the LLM model with full conversation context
func (b *Bot) ThreadMessageResponse(ctx context.Context, msg types.DiscordAskMessage, conversationHistory []llms.MessageContent) (string, error) {
	b.logger.Debug("processing discord thread message response", "messageID", msg.MessageID, "threadID", msg.ThreadID)

	// Build message history starting with system prompt
	messageHistory := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, askPedroPrompt)}

	// Add conversation history (excluding the current message since it's already in conversationHistory)
	messageHistory = append(messageHistory, conversationHistory...)

	// Add current message
	messageHistory = append(messageHistory, llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("%s: %s", msg.Username, msg.Message)))

	b.logger.Debug("calling LLM for discord thread message", "messageID", msg.MessageID, "historyLength", len(messageHistory))
	resp, err := b.llm.GenerateContent(ctx, messageHistory,
		llms.WithModel(b.modelName),
		llms.WithCandidateCount(1),
		llms.WithMaxLength(500),
		llms.WithTemperature(0.7),
		llms.WithPresencePenalty(1.0),
		llms.WithStopWords([]string{"@pedro", "@Pedro", "@PedroAI", "@PedroAI_"}))
	if err != nil {
		b.logger.Error("failed to get discord thread LLM response", "error", err.Error(), "messageID", msg.MessageID)
		metrics.FailedLLMGenCount.Add(1)
		return "", fmt.Errorf("failed to get llm response: %w", err)
	}

	prompt := ai.CleanResponse(resp.Choices[0].Content)
	b.logger.Debug("received discord thread LLM response", "messageID", msg.MessageID, "responseLength", len(prompt))

	if prompt == "" {
		b.logger.Warn("empty response from discord thread LLM", "messageID", msg.MessageID)
		metrics.EmptyLLMResponseCount.Add(1)
		return fmt.Sprintf("sorry, I cannot respond to @%s. Please try again", msg.Username), nil
	}

	b.logger.Debug("successful discord thread response generation", "messageID", msg.MessageID, "messageLength", len(prompt))
	metrics.SuccessfulLLMGenCount.Add(1)
	return prompt, nil
}
