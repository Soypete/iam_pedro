package discordchat

import (
	"context"
	"fmt"
	"strings"

	"github.com/Soypete/twitch-llm-bot/ai"
	"github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"
)

var GameChatHistory []llms.MessageContent
var thing string

func (c *Bot) manageGame(message string, chatType llms.ChatMessageType) {
	if len(GameChatHistory) == 0 {
		GameChatHistory = []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, "you are a chat bot playing 20 questions. The goal of the game is to guess what thing that the use is thinking of. You can ask yes or no questions to the user to help you narrow down that the thing is. I can be from any context like movies, history, a location etc. Make sure that your questions exclude certain criteria. If you get a positive response then the thing is inclide that criteria and you should dig in. Once you think you know what it is, make sure you take a guess. Only ask one question at a time.")}
		message = "I am thinking of a thing. Ask me a yes or no question to help you guess what it is."
	}

	GameChatHistory = append(GameChatHistory, llms.TextParts(chatType, message))
}

// End20Questions is a response from the LLM model to end the game of 20 questions
func (c *Bot) End20Questions() {
	GameChatHistory = nil
	thing = ""
}

// Play20Questions is a response from the LLM model to a game of 20 questions
func (c *Bot) Play20Questions(ctx context.Context, msg database.TwitchMessage, messageID uuid.UUID) (string, error) {
	if thing == "" {
		thing = msg.Text
	}

	c.manageGame(msg.Text, llms.ChatMessageTypeHuman)

	//start the game
	resp, err := c.llm.GenerateContent(ctx, GameChatHistory,
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
	// loop for checking if the message is the thing
	if strings.Contains(prompt, thing) {
		return fmt.Sprintf("I have guessed the thing you are thinking of. It is %s", thing), nil
	}

	c.manageGame(prompt, llms.ChatMessageTypeAI)

	metrics.SuccessfulLLMGen.Add(1)
	return prompt, nil
}
