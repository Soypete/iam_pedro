package twitchchat

import (
	"context"
	"fmt"
	"strings"

	"github.com/Soypete/twitch-llm-bot/ai"
	database "github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"
)

func (c *Client) manageChatHistory(ctx context.Context, injection []string, chatType llms.ChatMessageType) {

	if len(c.chatHistory) >= 10 {
		c.chatHistory = c.chatHistory[1:]
	}
	c.chatHistory = append(c.chatHistory, llms.TextParts(chatType, strings.Join(injection, " ")))
}

func (c *Client) callLLM(ctx context.Context, injection []string, messageID uuid.UUID) (string, error) {
	c.manageChatHistory(ctx, injection, llms.ChatMessageTypeHuman)
	messageHistory := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, ai.PedroPrompt)}
	messageHistory = append(messageHistory, c.chatHistory...)

	resp, err := c.llm.GenerateContent(ctx, messageHistory,
		llms.WithCandidateCount(1),
		llms.WithMaxLength(500),
		llms.WithTemperature(0.7),
		llms.WithPresencePenalty(1.0), // 2 is the largest penalty for using a work that has already been used
		llms.WithStopWords([]string{"LUL, PogChamp, Kappa, KappaPride, KappaRoss, KappaWealth"}))
	if err != nil {
		return "", fmt.Errorf("failed to get llm response: %w", err)
	}

	// add pedro's prompt to the chat history
	c.manageChatHistory(ctx, []string{ai.CleanResponse(resp.Choices[0].Content)}, llms.ChatMessageTypeAI)

	err = c.db.InsertResponse(ctx, resp, messageID, c.modelName)
	if err != nil {
		return ai.CleanResponse(resp.Choices[0].Content), fmt.Errorf("failed to write to db: %w", (err))
	}
	return ai.CleanResponse(resp.Choices[0].Content), nil
}

// SingleMessageResponse is a response from the LLM model to a single message, but to work it needs to have context of chat history
func (c *Client) SingleMessageResponse(ctx context.Context, msg database.TwitchMessage, messageID uuid.UUID) (string, error) {
	// TODO: i don't like passing the []string here. it should be cast in the callLLM function
	prompt, err := c.callLLM(ctx, []string{fmt.Sprintf("%s: %s", msg.Username, msg.Text)}, messageID)
	if err != nil {
		metrics.FailedLLMGen.Add(1)
		return "", err
	}
	if prompt == "" {
		metrics.EmptyLLMResponse.Add(1)
		// We are trying to tag the user to get them to try again with a better prompt.
		return fmt.Sprintf("sorry, I cannot respont to @%s. Please try again", msg.Username), nil
	}
	metrics.SuccessfulLLMGen.Add(1)
	return prompt, nil
}

// End20Questions is a response from the LLM model to end the game of 20 questions
func (c *Client) End20Questions() {
}

// Play20Questions is a response from the LLM model to a game of 20 questions
func (c *Client) Play20Questions(ctx context.Context, msg database.TwitchMessage, messageID uuid.UUID) (string, error) {
	return "", nil
}
