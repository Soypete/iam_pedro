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
	c.logger.Debug("managing chat history", "type", chatType, "message", strings.Join(injection, " "))
	
	if len(c.chatHistory) >= 10 {
		c.logger.Debug("pruning chat history", "old_size", len(c.chatHistory))
		c.chatHistory = c.chatHistory[1:]
	}
	c.chatHistory = append(c.chatHistory, llms.TextParts(chatType, strings.Join(injection, " ")))
	c.logger.Debug("updated chat history", "new_size", len(c.chatHistory))
}

func (c *Client) callLLM(ctx context.Context, injection []string, messageID uuid.UUID) (string, error) {
	c.logger.Debug("calling LLM", "message", strings.Join(injection, " "), "messageID", messageID)
	
	c.manageChatHistory(ctx, injection, llms.ChatMessageTypeHuman)
	messageHistory := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, ai.PedroPrompt)}
	messageHistory = append(messageHistory, c.chatHistory...)

	c.logger.Debug("generating content", "historyLength", len(messageHistory))
	resp, err := c.llm.GenerateContent(ctx, messageHistory,
		llms.WithCandidateCount(1),
		llms.WithMaxLength(500),
		llms.WithTemperature(0.7),
		llms.WithPresencePenalty(1.0), // 2 is the largest penalty for using a work that has already been used
		llms.WithStopWords([]string{"LUL, PogChamp, Kappa, KappaPride, KappaRoss, KappaWealth"}))
	if err != nil {
		c.logger.Error("failed to get LLM response", "error", err.Error())
		return "", fmt.Errorf("failed to get llm response: %w", err)
	}

	// add pedro's prompt to the chat history
	cleanedResponse := ai.CleanResponse(resp.Choices[0].Content)
	c.logger.Debug("received LLM response", "response", cleanedResponse)
	c.manageChatHistory(ctx, []string{cleanedResponse}, llms.ChatMessageTypeAI)

	err = c.db.InsertResponse(ctx, resp, messageID, c.modelName)
	if err != nil {
		c.logger.Error("failed to write response to database", "error", err.Error(), "messageID", messageID)
		return cleanedResponse, fmt.Errorf("failed to write to db: %w", (err))
	}
	
	c.logger.Debug("successfully generated and stored response", "messageID", messageID)
	return cleanedResponse, nil
}

// SingleMessageResponse is a response from the LLM model to a single message, but to work it needs to have context of chat history
func (c *Client) SingleMessageResponse(ctx context.Context, msg database.TwitchMessage, messageID uuid.UUID) (string, error) {
	c.logger.Debug("processing single message response", "messageID", messageID)
	
	// TODO: i don't like passing the []string here. it should be cast in the callLLM function
	prompt, err := c.callLLM(ctx, []string{fmt.Sprintf("%s: %s", msg.Username, msg.Text)}, messageID)
	if err != nil {
		c.logger.Error("failed to generate response", "error", err.Error(), "messageID", messageID)
		metrics.FailedLLMGen.Add(1)
		return "", err
	}
	
	if prompt == "" {
		c.logger.Warn("empty response from LLM", "messageID", messageID)
		metrics.EmptyLLMResponse.Add(1)
		// We are trying to tag the user to get them to try again with a better prompt.
		return fmt.Sprintf("sorry, I cannot respond to @%s. Please try again", msg.Username), nil
	}
	
	c.logger.Debug("successful response generation", "messageID", messageID, "messageLength", len(prompt))
	metrics.SuccessfulLLMGen.Add(1)
	return prompt, nil
}

// End20Questions is a response from the LLM model to end the game of 20 questions
func (c *Client) End20Questions() {
	c.logger.Debug("ending 20 questions game")
}

// Play20Questions is a response from the LLM model to a game of 20 questions
func (c *Client) Play20Questions(ctx context.Context, msg database.TwitchMessage, messageID uuid.UUID) (string, error) {
	c.logger.Debug("play 20 questions called but not implemented for twitch", "messageID", messageID)
	return "", nil
}
