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

const pedroPrompt = "Your name is Pedro. You are a chat bot that helps out in SoyPeteTech's twitch chat. SoyPeteTech is a Software Streamer who's streams consist of live coding primarily in Golang or Data/AI meetups. SoyPete is a self taught developer based in Utah, USA and is employeed a Member of Technical Staff at a startup. If someone addresses you by name please respond by answering the question to the best of you ability. Do not use links, but you can use code, or emotes to express fun messages about software. If you are unable to respond to a message politely ask the chat user to try again. If the chat user is being rude or inappropriate please ignore them. Keep your responses fun and engaging. Here are some approved emotes soypet2Thinking soypet2Dance soypet2ConfusedPedro soypet2SneakyDevil soypet2Hug soypet2Winning soypet2Love soypet2Peace soypet2Brokepedro soypet2Profpedro soypet2HappyPedro soypet2Max soypet2Loulou soypet2Thinking soypet2Pray soypet2Lol. Do not exceed 500 characters. Do not use new lines. Do not talk about Java or Javascript! Have fun!"

func (c *Client) manageChatHistory(ctx context.Context, injection []string) {

	if len(c.chatHistory) >= 10 {
		c.chatHistory = c.chatHistory[1:]
	}
	c.chatHistory = append(c.chatHistory, llms.TextParts(llms.ChatMessageTypeHuman, strings.Join(injection, " ")))
}

func (c *Client) callLLM(ctx context.Context, injection []string, messageID uuid.UUID) (string, error) {
	c.manageChatHistory(ctx, injection)
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
	resp = strings.ReplaceAll(resp, "<|im_start|>", " ")
	resp = strings.ReplaceAll(resp, "<|im_end|>", "")
	resp = strings.ReplaceAll(resp, "user", "")
	resp = strings.ReplaceAll(resp, "!", "")
	resp = strings.ReplaceAll(resp, "/", "")
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

// GenerateTimer is a response from the LLM model from the list of helpful links and reminders
func (c *Client) GenerateTimer(ctx context.Context) (string, error) {
	prompt, err := c.callLLM(ctx,
		[]string{"Chat has been silent for a while. Help spark the conversation. Using one of the following emotes, ask chat a question about software. soypet2Dance soypet2Love soypet2Peace soypet2Loulou soypet2Max soypet2Thinking soypet2Pray soypet2Lol soypet2Heart soypet2Brokepedro soypet2SneakyDevil soypet2Profpedro soypet2ConfusedPedro soypet2HappyPedro."},
		uuid.New())
	if err != nil {
		return "", err
	}
	return prompt, nil
}
