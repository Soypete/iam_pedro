package twitchirc

import (
	"context"
	"strings"
	"time"

	database "github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/metrics"
	v2 "github.com/gempir/go-twitch-irc/v2"
)

func cleanMessage(msg v2.PrivateMessage) database.TwitchMessage {
	chat := database.TwitchMessage{
		Username: msg.User.DisplayName,
		Text:     msg.Message,
		// TODO: add an embedding for the message
		Time: time.Now(),
	}

	if strings.HasPrefix(msg.Message, "!") {
		chat.IsCommand = true
	}

	if strings.Contains(msg.User.DisplayName, "RestreamBot") {
		text := strings.ReplaceAll(msg.Message, "]", ":")
		words := strings.Split(text, ":")

		chat.Username = strings.TrimSpace(words[1]) // sets username to the first word after the video source.
		chat.Text = strings.TrimSpace(words[2])     // create a clean message without the video source.
	}
	return chat
}

func needsResponseChat(msg database.TwitchMessage) bool {
	switch {
	case strings.Contains(msg.Text, "pedro"):
		return true
	case strings.Contains(msg.Text, "Pedro"):
		return true
	case strings.Contains(msg.Text, "llm"):
		return true
	case strings.Contains(msg.Text, "LLM"):
		return true
	case strings.Contains(msg.Text, "bot"):
		return true
	default:
		return false
	}
}

func (irc *IRC) HandleChat(ctx context.Context, msg v2.PrivateMessage) {
	chat := cleanMessage(msg)

	// do not persist messages from the Nightbot
	if msg.User.DisplayName == "Nightbot" {
		irc.logger.Debug("ignoring Nightbot message")
		return
	}

	// TODO: replace nitbot commands with a classifier model that prompts the LLM
	if strings.Contains(chat.Text, "Pedro") || strings.Contains(chat.Text, "pedro") || strings.Contains(chat.Text, "soy_llm_bot") {
		irc.logger.Debug("processing message that mentions bot")

		messageID, err := irc.db.InsertMessage(ctx, chat)
		if err != nil {
			irc.logger.Error("failed to insert message into database", "error", err.Error())
			return
		}

		irc.logger.Debug("message inserted into database", "messageID", messageID)
		resp, err := irc.llm.SingleMessageResponse(ctx, chat, messageID)
		if err != nil {
			irc.logger.Error("failed to get response from LLM", "error", err.Error(), "messageID", messageID)
			return
		}

		// Don't log the actual response content to protect privacy
		irc.logger.Debug("sending response to Twitch", "messageID", messageID, "responseLength", len(resp))
		irc.Client.Say("soypetetech", resp)
		metrics.TwitchMessageSentCount.Add(1)
	}
}
