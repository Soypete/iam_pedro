package discordchat

import (
	"context"
	"fmt"

	"github.com/Soypete/twitch-llm-bot/ai"
	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/Soypete/twitch-llm-bot/types"
	"github.com/tmc/langchaingo/llms"
)

const askPedroPrompt = "Your name is Pedro. You are a chat bot that helps out in SoyPeteTech's discord server. SoyPeteTech is a Software Streamer (Aka Miriah Peterson) who's streams consist of live coding primarily in Golang or Data/AI meetups. She also published shorts to tiktok, videos to youtube, and blogs to substack. Your code is found at https://github.com/SoyPete/IamPedro. All other links are on https://linktr.ee/soypete_tech. She is a self taught developer based in Utah, USA and is employeed a Member of Technical Staff at a startup. If someone addresses you by name please respond by answering the question to the best of you ability. You can use code to express fun messages about software. If you are unable to respond to a message politely ask the chat user to try again. If the chat user is being rude or inappropriate please ignore them. Keep your responses fun and engaging. Here are some approved emotes Do not talk about Java or Javascript! Have fun!"

// SingleMessageResponse is a response from the LLM model to a single message
func (b *Bot) SingleMessageResponse(ctx context.Context, msg types.DiscordAskMessage) (string, error) {
	b.logger.Debug("processing discord single message response", "messageID", msg.ThreadID)

	messageHistory := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, askPedroPrompt)}
	messageHistory = append(messageHistory, llms.TextParts(llms.ChatMessageTypeHuman, msg.Message))

	b.logger.Debug("calling LLM for discord message", "messageID", msg.ThreadID)
	resp, err := b.llm.GenerateContent(context.Background(), messageHistory,
		llms.WithCandidateCount(1),
		llms.WithMaxLength(500),
		llms.WithTemperature(0.7),
		llms.WithPresencePenalty(1.0), // 2 is the largest penalty for using a work that has already been used
		llms.WithStopWords([]string{"@pedro", "@Pedro", "@PedroAI", "@PedroAI_"}))
	if err != nil {
		b.logger.Error("failed to get discord LLM response", "error", err.Error(), "messageID", msg.ThreadID)
		metrics.FailedLLMGen.Add(1)
		return "", fmt.Errorf("failed to get llm response: %w", err)
	}

	prompt := ai.CleanResponse(resp.Choices[0].Content)
	b.logger.Debug("received discord LLM response", "messageID", msg.ThreadID, "responseLength", len(prompt))

	if prompt == "" {
		b.logger.Warn("empty response from discord LLM", "messageID", msg.ThreadID)
		metrics.EmptyLLMResponse.Add(1)
		// We are trying to tag the user to get them to try again with a better prompt.
		return fmt.Sprintf("sorry, I cannot respond to @%s. Please try again", msg.Username), nil
	}

	b.logger.Debug("successful discord response generation", "messageID", msg.ThreadID, "messageLength", len(prompt))
	metrics.SuccessfulLLMGen.Add(1)
	return prompt, nil
}

// ConversationResponse handles a response with conversation history
func (b *Bot) ConversationResponse(ctx context.Context, messages []types.DiscordAskMessage, newMessage string) (string, error) {
	b.logger.Debug("processing discord conversation response", "historyLength", len(messages))

	messageHistory := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, askPedroPrompt)}
	
	// Add conversation history
	for _, msg := range messages {
		if msg.IsFromPedro {
			messageHistory = append(messageHistory, llms.TextParts(llms.ChatMessageTypeAI, msg.Message))
		} else {
			messageHistory = append(messageHistory, llms.TextParts(llms.ChatMessageTypeHuman, msg.Message))
		}
	}
	
	// Add the new message
	messageHistory = append(messageHistory, llms.TextParts(llms.ChatMessageTypeHuman, newMessage))

	b.logger.Debug("calling LLM for discord conversation", "messageCount", len(messageHistory))
	resp, err := b.llm.GenerateContent(context.Background(), messageHistory,
		llms.WithCandidateCount(1),
		llms.WithMaxLength(500),
		llms.WithTemperature(0.7),
		llms.WithPresencePenalty(1.0),
		llms.WithStopWords([]string{"@pedro", "@Pedro", "@PedroAI", "@PedroAI_"}))
	if err != nil {
		b.logger.Error("failed to get discord conversation LLM response", "error", err.Error())
		metrics.FailedLLMGen.Add(1)
		return "", fmt.Errorf("failed to get llm response: %w", err)
	}

	prompt := ai.CleanResponse(resp.Choices[0].Content)
	b.logger.Debug("received discord conversation LLM response", "responseLength", len(prompt))

	if prompt == "" {
		b.logger.Warn("empty response from discord conversation LLM")
		metrics.EmptyLLMResponse.Add(1)
		return "Sorry, I cannot respond to that. Please try again", nil
	}

	b.logger.Debug("successful discord conversation response generation", "messageLength", len(prompt))
	metrics.SuccessfulLLMGen.Add(1)
	return prompt, nil
}
