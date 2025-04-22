package discordchat

import (
	"context"
	"fmt"
	"strings"

	"github.com/Soypete/twitch-llm-bot/ai"
	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/Soypete/twitch-llm-bot/types"
	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"
)

var (
	GameChatHistory []llms.MessageContent
	thing           string
)

func (c *Bot) manageGame(message string, chatType llms.ChatMessageType) {
	if len(GameChatHistory) == 0 {
		c.logger.Debug("initializing new 20 questions game")
		GameChatHistory = []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, "you are a chat bot playing 20 questions. The goal of the game is to guess what thing that the user is thinking of by excluding different categories. You can ask yes or no questions to the user to help you narrow down that the thing is. I can be from any context like movies, history, a location etc. You can only ask one question at a time. If you get a positive response then the thing is inclide that criteria and you should dig in. Once you think you know what it is, ask the user if the thing is a certain thing. If you are wrong, the game is over. If you are right, the game is over.")}
		message = "I am thinking of a thing. Ask me a yes or no question to help you guess what it is."
	}

	c.logger.Debug("adding message to game history", "message", message, "type", chatType)
	GameChatHistory = append(GameChatHistory, llms.TextParts(chatType, message))
	c.logger.Debug("game history updated", "historyLength", len(GameChatHistory))
}

// End20Questions is a response from the LLM model to end the game of 20 questions
func (c *Bot) End20Questions() {
	c.logger.Info("ending 20 questions game")
	GameChatHistory = nil
	thing = ""
}

// Play20Questions is a response from the LLM model to a game of 20 questions
func (c *Bot) Play20Questions(ctx context.Context, msg types.TwitchMessage, messageID uuid.UUID) (string, error) {
	c.logger.Debug("processing 20 questions turn", "messageID", messageID)

	if thing == "" {
		c.logger.Debug("setting thing for 20 questions game")
		thing = msg.Text
	}

	c.manageGame(msg.Text, llms.ChatMessageTypeHuman)

	// start the game
	c.logger.Debug("calling LLM for 20 questions", "historyLength", len(GameChatHistory))
	resp, err := c.llm.GenerateContent(ctx, GameChatHistory,
		llms.WithCandidateCount(1),
		llms.WithMaxLength(500),
		llms.WithTemperature(0.7),
		llms.WithPresencePenalty(1.0), // 2 is the largest penalty for using a work that has already been used
		llms.WithStopWords([]string{"LUL, PogChamp, Kappa, KappaPride, KappaRoss, KappaWealth"}))
	if err != nil {
		c.logger.Error("failed to get 20 questions LLM response", "error", err.Error(), "messageID", messageID)
		metrics.FailedLLMGen.Add(1)
		return "", fmt.Errorf("failed to get llm response: %w", err)
	}

	prompt := ai.CleanResponse(resp.Choices[0].Content)
	c.logger.Debug("received 20 questions LLM response", "responseLength", len(prompt))

	if prompt == "" {
		c.logger.Warn("empty response from 20 questions LLM", "messageID", messageID)
		metrics.EmptyLLMResponse.Add(1)
		// We are trying to tag the user to get them to try again with a better prompt.
		return fmt.Sprintf("sorry, I cannot respond to @%s. Please try again", msg.Username), nil
	}

	// loop for checking if the message is the thing
	if strings.Contains(prompt, thing) {
		c.logger.Info("LLM guessed the thing correctly")
		return fmt.Sprintf("I have guessed the thing you are thinking of. It is %s", thing), nil
	}

	c.manageGame(prompt, llms.ChatMessageTypeAI)

	c.logger.Info("successful 20 questions response", "questionCount", (len(GameChatHistory)-1)/2)
	metrics.SuccessfulLLMGen.Add(1)
	return prompt, nil
}
