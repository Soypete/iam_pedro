package discordchat

import (
	"context"
	"fmt"

	"github.com/Soypete/twitch-llm-bot/ai"
	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/Soypete/twitch-llm-bot/types"
	"github.com/tmc/langchaingo/llms"
)

const (
	QuestionSystemPrompt = `You are Pedro, the discord chat bot playing 20 questions in the SoyPeteTech discord community. 
The goal of the game is to guess what thing that the user is thinking of by excluding different categories. 
You can ask yes or no questions to the user to help you narrow down that the thing is. It can be from any context 
like movies, history, a location etc. You can only ask one question at a time. The User will respond to you question with a positive 
	negative sentiment or yes or no. If you get a positive response then the 
answer is included that criteria and you should dig in. Once you think you know what it is, ask the user if the you 
have guessed correctly by specifying what your guess is directly. Continue guessing and narrowing down the cirteria until
	you are out of questions. If you guess correctly before you have asked all 20 qustions and you are corret, the game is over and 
	you win. If you cannot guess the thing with	20 questions, the user wins.`
	IntroMessage = "%s: I am thinking of a thing. Ask me a yes or no question to help you guess what it is."
)

func (c *Bot) formatPrompt(user string) []llms.MessageContent {
	c.logger.Debug("initializing new 20 questions game")
	return []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, QuestionSystemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf(IntroMessage, user)),
	}
}

// Start20Questions is a response from the LLM model to start a game of 20 questions.
// It returns the first question.
func (c *Bot) Start20Questions(ctx context.Context, msg types.Discord20QuestionsGame) (string, error) {
	c.logger.Debug("starting 20 questions game")
	chat := c.formatPrompt(msg.Username)
	resp, err := c.llm.GenerateContent(ctx, chat,
		llms.WithCandidateCount(1),
		llms.WithTemperature(0.7),
		llms.WithPresencePenalty(1.0)) // 2 is the largest penalty for using a work that has already been used
	if err != nil {
		c.logger.Error("failed to get 20 questions LLM response", "error", err.Error(), "messageID", msg.GameID)
		metrics.FailedLLMGen.Add(1)
		return "", fmt.Errorf("failed to get llm response: %w", err)
	}
	return resp.Choices[0].Content, nil
}

// Play20Questions is a response from the LLM model to a game of 20 questions
func (c *Bot) Play20Questions(ctx context.Context, user string, gameChat []llms.MessageContent) (string, error) {
	chat := append(c.formatPrompt(user), gameChat...)
	c.logger.Debug("calling LLM for 20 questions", "historyLength", len(chat))
	resp, err := c.llm.GenerateContent(ctx, chat,
		llms.WithCandidateCount(1),
		llms.WithTemperature(0.7),
		llms.WithPresencePenalty(1.0)) // 2 is the largest penalty for using a work that has already been used
	if err != nil {
		c.logger.Error("failed to get 20 questions LLM response", "error", err.Error())
		metrics.FailedLLMGen.Add(1)
		return "", fmt.Errorf("failed to get llm response: %w", err)
	}

	prompt := ai.CleanResponse(resp.Choices[0].Content)
	c.logger.Debug("received 20 questions LLM response", "responseLength", len(prompt))

	if prompt == "" {
		c.logger.Warn("empty response from 20 questions LLM")
		metrics.EmptyLLMResponse.Add(1)
		// We are trying to tag the user to get them to try again with a better prompt.
		return fmt.Sprintf("sorry, I cannot respond to @%s. Please try again", user), nil
	}

	metrics.SuccessfulLLMGen.Add(1)
	return prompt, nil
}
