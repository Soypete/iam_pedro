package discordchat

import (
	"context"
	"fmt"

	"github.com/Soypete/twitch-llm-bot/ai"
	"github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"
)

// SingleMessageResponse is a response from the LLM model to a single message
func (b *Bot) SingleMessageResponse(ctx context.Context, msg database.TwitchMessage, messageID uuid.UUID) (database.TwitchMessage, error) {
	b.logger.Debug("processing discord single message response", "messageID", messageID)

	messageHistory := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, ai.PedroPrompt)}
	messageHistory = append(messageHistory, llms.TextParts(llms.ChatMessageTypeHuman, msg.Text))

	b.logger.Debug("calling LLM for discord message", "messageID", messageID)
	resp, err := b.llm.GenerateContent(context.Background(), messageHistory,
		llms.WithCandidateCount(1),
		llms.WithMaxLength(500),
		llms.WithTemperature(0.7),
		llms.WithPresencePenalty(1.0), // 2 is the largest penalty for using a work that has already been used
		llms.WithStopWords([]string{"@pedro", "@Pedro", "@PedroAI", "@PedroAI_"}))
	if err != nil {
		b.logger.Error("failed to get discord LLM response", "error", err.Error(), "messageID", messageID)
		metrics.FailedLLMGen.Add(1)
		return database.TwitchMessage{}, fmt.Errorf("failed to get llm response: %w", err)
	}

	prompt := ai.CleanResponse(resp.Choices[0].Content)
	b.logger.Debug("received discord LLM response", "messageID", messageID, "responseLength", len(prompt))

	if prompt == "" {
		b.logger.Warn("empty response from discord LLM", "messageID", messageID)
		metrics.EmptyLLMResponse.Add(1)
		// We are trying to tag the user to get them to try again with a better prompt.
		return database.TwitchMessage{
			Text: fmt.Sprintf("sorry, I cannot respond to @%s. Please try again", msg.Username),
			UUID: messageID,
		}, nil
	}

	b.logger.Debug("successful discord response generation", "messageID", messageID, "messageLength", len(prompt))
	metrics.SuccessfulLLMGen.Add(1)
	return database.TwitchMessage{
		Text: prompt,
		UUID: messageID,
	}, nil
}
