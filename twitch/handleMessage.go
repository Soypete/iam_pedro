package twitchirc

import (
	"context"
	"strings"
	"time"

	"github.com/Soypete/twitch-llm-bot/ai/twitchchat"
	"github.com/Soypete/twitch-llm-bot/metrics"
	types "github.com/Soypete/twitch-llm-bot/types"
	v2 "github.com/gempir/go-twitch-irc/v2"
)

func cleanMessage(msg v2.PrivateMessage) types.TwitchMessage {
	chat := types.TwitchMessage{
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

func needsResponseChat(msg types.TwitchMessage) bool {
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

	// Fork message to FAQ processor (non-blocking, runs in parallel)
	// This checks if the message matches any FAQ entries and responds automatically
	if irc.faqProcessor != nil && ShouldProcessMessage(msg) {
		metrics.FAQCheckCount.Add(1)
		go irc.faqProcessor.ProcessMessageFromPrivate(ctx, msg)
	}

	// TODO: replace nitbot commands with a classifier model that prompts the LLM
	if strings.Contains(chat.Text, "Pedro") || strings.Contains(chat.Text, "pedro") || strings.Contains(chat.Text, "soy_llm_bot") {
		irc.logger.Debug("processing message that mentions bot")

		messageID, err := irc.db.InsertMessage(ctx, chat)
		if err != nil {
			irc.logger.Error("failed to insert message into types", "error", err.Error())
			return
		}

		irc.logger.Debug("message inserted into types", "messageID", messageID)
		resp, err := irc.llm.SingleMessageResponse(ctx, chat, messageID)
		if err != nil {
			irc.logger.Error("failed to get response from LLM", "error", err.Error(), "messageID", messageID)
			return
		}

		// Check if this is a web search request
		if resp.WebSearch != nil {
			irc.logger.Debug("web search requested", "query", resp.WebSearch.Query, "messageID", messageID)

			// Send immediate response
			err = irc.db.InsertResponse(ctx, resp, irc.modelName)
			if err != nil {
				irc.logger.Error("failed to insert immediate response into database", "error", err.Error(), "messageID", resp.UUID)
			}
			irc.Client.Say(peteTwitchChannel, resp.Text)
			metrics.TwitchMessageSentCount.Add(1)

			// Start async web search
			if twitchLLM, ok := irc.llm.(*twitchchat.Client); ok {
				go twitchLLM.ExecuteWebSearch(ctx, resp.WebSearch, irc.asyncResponseCh)
			} else {
				irc.logger.Error("LLM client does not support web search")
			}
			return
		}

		err = irc.db.InsertResponse(ctx, resp, irc.modelName)
		if err != nil {
			irc.logger.Error("failed to insert response into types", "error", err.Error(), "messageID", resp.UUID)
			// continue to send the response even if it fails to insert into the types
		}
		// Don't log the actual response content to protect privacy
		irc.logger.Debug("sending response to Twitch", "messageID", resp.UUID, "responseLength", len(resp.Text))
		irc.Client.Say(peteTwitchChannel, resp.Text)
		metrics.TwitchMessageSentCount.Add(1)
	}
}
