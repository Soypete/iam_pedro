package langchain

import (
	"context"
	"fmt"
	"strings"

	database "github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"
)

const pedroPrompt = "Your name is Pedro. You are a chat bot that helps out in SoyPeteTech's twitch chat. SoyPeteTech is a Software Streamer (Aka Miriah Peterson) who's streams consist of live coding primarily in Golang or Data/AI meetups. She is a self taught developer based in Utah, USA and is employeed a Member of Technical Staff at a startup. If someone addresses you by name please respond by answering the question to the best of you ability. Do not use links, but you can use code, or emotes to express fun messages about software. If you are unable to respond to a message politely ask the chat user to try again. If the chat user is being rude or inappropriate please ignore them. Keep your responses fun and engaging. Here are some approved emotes soypet2Thinking soypet2Dance soypet2ConfusedPedro soypet2SneakyDevil soypet2Hug soypet2Winning soypet2Love soypet2Peace soypet2Brokepedro soypet2Profpedro soypet2HappyPedro soypet2Max soypet2Loulou soypet2Thinking soypet2Pray soypet2Lol. Do not exceed 500 characters. Do not use new lines. Do not talk about Java or Javascript! Have fun!"

func (c *Client) manageChatHistory(ctx context.Context, injection []string, chatType llms.ChatMessageType) {

	if len(c.chatHistory) >= 10 {
		c.chatHistory = c.chatHistory[1:]
	}
	c.chatHistory = append(c.chatHistory, llms.TextParts(chatType, strings.Join(injection, " ")))
}

func (c *Client) callLLM(ctx context.Context, injection []string, messageID uuid.UUID) (string, error) {
	c.manageChatHistory(ctx, injection, llms.ChatMessageTypeHuman)
	messageHistory := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, pedroPrompt)}
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
	c.manageChatHistory(ctx, []string{cleanResponse(resp.Choices[0].Content)}, llms.ChatMessageTypeAI)

	err = c.db.InsertResponse(ctx, resp, messageID, c.modelName)
	if err != nil {
		return cleanResponse(resp.Choices[0].Content), fmt.Errorf("failed to write to db: %w", (err))
	}
	return cleanResponse(resp.Choices[0].Content), nil
}

// cleanResponse removes any newlines from the response
func cleanResponse(resp string) string {
	// remove any newlines
	resp = strings.ReplaceAll(resp, "\n", " ")
	resp = strings.ReplaceAll(resp, "<|im_start|>", "")
	resp = strings.ReplaceAll(resp, "<|im_end|>", "")
	resp = strings.TrimPrefix(resp, "!") // remove any leading ! so that we dont trigger commands
	resp = strings.TrimPrefix(resp, "/") // remove any leading / so that we dont trigger commands
	return strings.TrimSpace(resp)
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

var GameChatHistory []llms.MessageContent
var thing string

func (c *Client) manageGame(message string, chatType llms.ChatMessageType) {
	if len(GameChatHistory) == 0 {
		GameChatHistory = []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, "you are a chat bot playing 20 questions. The goal of the game is to guess what thing that the use is thinking of. You can ask yes or no questions to the user to help you narrow down that the thing is. I can be from any context like movies, history, a location etc. Make sure that your questions exclude certain criteria. Only ask one question at a time.")}
		message = "I am thinking of a thing. Ask me a yes or no question to help you guess what it is."
	}

	GameChatHistory = append(GameChatHistory, llms.TextParts(chatType, message))
}

// End20Questions is a response from the LLM model to end the game of 20 questions
func (c *Client) End20Questions() {
	GameChatHistory = nil
	thing = ""
}

// Play20Questions is a response from the LLM model to a game of 20 questions
func (c *Client) Play20Questions(ctx context.Context, msg database.TwitchMessage, messageID uuid.UUID) (string, error) {
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
	prompt := cleanResponse(resp.Choices[0].Content)
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
