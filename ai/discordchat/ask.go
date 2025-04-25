package discordchat

import (
	"context"
	"fmt"

	"github.com/Soypete/twitch-llm-bot/ai"
	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/Soypete/twitch-llm-bot/types"
	"github.com/tmc/langchaingo/llms"
)

// SingleMessageResponse is a response from the LLM model to a single message
func (b *Bot) SingleMessageResponse(ctx context.Context, msg types.DiscordAskMessage) (string, error) {
	b.logger.Debug("processing discord single message response", "messageID", msg.ThreadID)

	messageHistory := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, ai.PedroPrompt)}
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
