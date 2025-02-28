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
	b.logger.Info("processing discord single message response", "user", msg.Username, "messageID", messageID)

	messageHistory := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, ai.PedroPrompt)}
	messageHistory = append(messageHistory, llms.TextParts(llms.ChatMessageTypeHuman, msg.Text))
	
	b.logger.Debug("calling LLM for discord message", "text", msg.Text)
	resp, err := b.llm.GenerateContent(context.Background(), messageHistory,
		llms.WithCandidateCount(1),
		llms.WithMaxLength(500),
		llms.WithTemperature(0.7),
		llms.WithPresencePenalty(1.0), // 2 is the largest penalty for using a work that has already been used
		llms.WithStopWords([]string{"@pedro", "@Pedro", "@PedroAI", "@PedroAI_"}))
	if err != nil {
		b.logger.Error("failed to get discord LLM response", "error", err.Error())
		metrics.FailedLLMGen.Add(1)
		return "", fmt.Errorf("failed to get llm response: %w", err)
	}
	
	prompt := ai.CleanResponse(resp.Choices[0].Content)
	b.logger.Debug("received discord LLM response", "response", prompt)
	
	if prompt == "" {
		b.logger.Warn("empty response from discord LLM", "user", msg.Username)
		metrics.EmptyLLMResponse.Add(1)
		// We are trying to tag the user to get them to try again with a better prompt.
		return fmt.Sprintf("sorry, I cannot respond to @%s. Please try again", msg.Username), nil
	}

	// Insert the response into the database
	if err = b.db.InsertResponse(ctx, resp, messageID, b.modelName); err != nil {
		b.logger.Error("failed to insert discord response into database", "error", err.Error(), "messageID", messageID)
		return prompt, fmt.Errorf("failed to insert response into database: %w", err)
	}
	
	b.logger.Info("successful discord response generation", "user", msg.Username, "messageLength", len(prompt))
	metrics.SuccessfulLLMGen.Add(1)
	return prompt, nil
}
