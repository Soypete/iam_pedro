package langchain

import (
	"context"
	"fmt"
	"strings"

	database "github.com/Soypete/twitch-llm-bot/database"
	"github.com/tmc/langchaingo/llms"
)

const pedroPrompt = "Your name is Pedro_el_asistente. You are a chat bot that helps out in SoyPeteTech's twitch chat. If someone addresses you by name please respode by answering the question to the best of you ability. You are allowed to use links, code, or emotes to express fun messages about software. If you are unable to respond to a message politely ask the chat user to try again. If the chat user is being rude or inappropriate please ignore them. If you are unsure about a message please ask SoyPeteTech for help. Also make sure to remind chat to follow the streamer and to check out the or other social media links."

func (c Client) callLLM(ctx context.Context, injection []string) (string, error) {
	messageHistory := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, pedroPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, strings.Join(injection, " "))}

	resp, err := c.llm.GenerateContent(ctx, messageHistory,
		llms.WithCandidateCount(1),
		llms.WithMaxLength(500),
		llms.WithTemperature(0.7),
		llms.WithPresencePenalty(1.0), // 2 is the largest penalty for using a work that has already been used
		llms.WithStopWords([]string{"LUL, PogChamp, Kappa, KappaPride, KappaRoss, KappaWealth"}))
	if err != nil {
		return "", fmt.Errorf("failed to get llm response: %w", err)
	}
	err = c.db.InsertResponse(ctx, resp)
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
	return strings.TrimSpace(resp)
}

// SingleMessageResponse is a response from the LLM model to a single message, but to work it needs to have context of chat history
func (c Client) SingleMessageResponse(ctx context.Context, msg database.TwitchMessage) (string, error) {
	prompt, err := c.callLLM(ctx, []string{fmt.Sprintf("%s: %s", msg.Username, msg.Text)})
	if err != nil {
		return "", err
	}
	return prompt, nil
}

// GenerateTimer is a response from the LLM model from the list of helpful links and reminders
func (c Client) GenerateTimer(ctx context.Context, jsonbody string) (string, error) {
	prompt, err := c.callLLM(ctx, []string{fmt.Sprintf("Respond with a twitch chat message for the SoyPeteTech twitch chat. The message should encourage the users to interact with Pete via the stream or other social media outlets (included in the json below). Keep the message short and direct. Make sure you are address chat as chat. make sure to include a call to action." + jsonbody)})
	if err != nil {
		return "", err
	}
	return prompt, nil
}
