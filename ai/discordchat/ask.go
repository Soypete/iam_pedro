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
func (b *Bot) SingleMessageResponse(ctx context.Context, msg database.TwitchMessage, messageID uuid.UUID) (string, error) {
	messageHistory := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, ai.PedroPrompt)}
	messageHistory = append(messageHistory, llms.TextParts(llms.ChatMessageTypeHuman, msg.Text))
	resp, err := b.llm.GenerateContent(ctx, messageHistory,
		llms.WithCandidateCount(1),
		llms.WithMaxLength(500),
		llms.WithTemperature(0.7),
		llms.WithPresencePenalty(1.0), // 2 is the largest penalty for using a work that has already been used
		llms.WithStopWords([]string{"LUL, PogChamp, Kappa, KappaPride, KappaRoss, KappaWealth"}))
	if err != nil {
		return "", fmt.Errorf("failed to get llm response: %w", err)
	}
	prompt := ai.CleanResponse(resp.Choices[0].Content)
	if prompt == "" {
		metrics.EmptyLLMResponse.Add(1)
		// We are trying to tag the user to get them to try again with a better prompt.
		return fmt.Sprintf("sorry, I cannot respont to @%s. Please try again", msg.Username), nil
	}
	return "", nil
}
