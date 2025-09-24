package discord

import (
	"context"
	"fmt"
	"time"

	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/Soypete/twitch-llm-bot/types"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func (d Client) askPedro(s *discordgo.Session, i *discordgo.InteractionCreate) {
	valid, err := messageValidatior(s, i)
	if !valid {
		d.logger.Error("error responding to askPedro command, no data", "error", err.Error())
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Failed to ask Pedro. Please try again later",
			},
		})
		return
	}

	data := i.Interaction.Data.(discordgo.ApplicationCommandInteractionData) // assert the data type
	text := data.Options[0].StringValue()
	messageID := uuid.New()

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Processing your question...",
		},
	})
	if err != nil {
		d.logger.Error("error responding to askPedro command", "error", err.Error())
		return
	}
	metrics.DiscordMessageSent.Add(1)

	// Get initial response
	message := types.DiscordAskMessage{
		Username:      i.Interaction.Member.User.Username,
		Message:       text,
		ThreadTimeout: 30, // 30 seconds timeout for thread responses
		IsFromPedro:   false,
	}

	d.logger.Debug("calling LLM for response", "messageID", messageID)
	resp, err := d.llm.SingleMessageResponse(context.Background(), message)
	if err != nil {
		d.logger.Error("error calling llm | single message response", "error", err.Error(), "messageID", messageID)
		_, _ = d.Session.ChannelMessageSend(i.Interaction.ChannelID, "Failed to ask Pedro. Please try again later")
		return
	}

	if resp == "" {
		d.logger.Warn("empty response from LLM", "messageID", messageID)
		resp = "Sorry, I cannot respond to that. Please try again."
	}

	userNumber := i.Interaction.Member.User.ID
	msgText := fmt.Sprintf("<@%s>:\nQuestion: %s\nResponse: %s", userNumber, text, resp)
	m, err := d.Session.ChannelMessageSend(i.Interaction.ChannelID, msgText)
	if err != nil {
		d.logger.Error("error sending message to channel", "error", err.Error(), "channelID", i.Interaction.ChannelID)
		return
	}
	metrics.DiscordMessageSent.Add(1)

	// Start a thread for follow-up questions
	threadTitle := "Ask Pedro: " + i.Interaction.Member.User.Username
	thread, err := d.Session.MessageThreadStart(m.ChannelID, m.ID, threadTitle, 1440) // 24 hours auto-archive
	if err != nil {
		d.logger.Error("error starting thread", "error", err.Error(), "channelID", m.ChannelID)
		return
	}
	d.logger.Debug("started thread for ask pedro", "threadID", thread.ID)

	// Store initial messages in database with thread ID
	message.ThreadID = thread.ID
	d.handleDBerror(d.db.InsertDiscordAskPedro(context.Background(), message))

	pedroMessage := types.DiscordAskMessage{
		ThreadID:      thread.ID,
		Username:      "Pedro",
		Message:       resp,
		ThreadTimeout: 30,
		IsFromPedro:   true,
	}
	d.handleDBerror(d.db.InsertDiscordAskPedro(context.Background(), pedroMessage))

	// Send initial message to thread
	_, err = d.Session.ChannelMessageSend(thread.ID, "Feel free to ask follow-up questions in this thread. I'll respond within 30 seconds!")
	if err != nil {
		d.logger.Error("error sending initial thread message", "error", err.Error(), "threadID", thread.ID)
		return
	}
	metrics.DiscordMessageSent.Add(1)

	// Start goroutine to handle thread conversation
	go d.handleAskPedroThread(thread.ID, i.Interaction.Member.User.Username)
}

func (d Client) handleAskPedroThread(threadID string, username string) {
	ctx := context.Background()
	d.logger.Debug("starting ask pedro thread handler", "threadID", threadID)

	lastMessageID := ""
	maxIterations := 20 // Maximum number of exchanges in the thread

	for i := 0; i < maxIterations; i++ {
		// Wait for user response
		time.Sleep(30 * time.Second)

		// Get new messages in the thread
		messageList, err := d.Session.ChannelMessages(threadID, 100, "", lastMessageID, "")
		if err != nil {
			d.logger.Error("error getting thread messages", "error", err.Error(), "threadID", threadID)
			return
		}
		metrics.DiscordMessageRecieved.Add(1)

		// Check if there are new messages from users (not Pedro)
		var userMessage *discordgo.Message
		for _, msg := range messageList {
			if !msg.Author.Bot {
				userMessage = msg
				break
			}
		}

		if userMessage == nil {
			// No new user messages, check if we should continue waiting
			if i > 0 {
				// After first iteration, if no response, end the thread monitoring
				d.logger.Debug("no new messages in thread, ending monitoring", "threadID", threadID)
				return
			}
			continue
		}

		d.logger.Debug("received user message in thread", "threadID", threadID, "message", userMessage.Content)

		// Store user message in database
		userMsg := types.DiscordAskMessage{
			ThreadID:      threadID,
			Username:      userMessage.Author.Username,
			Message:       userMessage.Content,
			ThreadTimeout: 30,
			IsFromPedro:   false,
		}
		d.handleDBerror(d.db.InsertDiscordAskPedro(ctx, userMsg))

		// Get conversation history from database
		history, err := d.db.GetDiscordAskPedroHistory(ctx, threadID)
		if err != nil {
			d.logger.Error("error getting conversation history", "error", err.Error(), "threadID", threadID)
			// Fall back to single message response if history retrieval fails
			history = []types.DiscordAskMessage{}
		}

		// Generate response with conversation context
		var resp string
		if len(history) > 0 {
			resp, err = d.llm.ConversationResponse(ctx, history, userMessage.Content)
		} else {
			// Fallback to single message if no history
			resp, err = d.llm.SingleMessageResponse(ctx, userMsg)
		}

		if err != nil {
			d.logger.Error("error generating response", "error", err.Error(), "threadID", threadID)
			resp = "Sorry, I encountered an error. Please try again."
		}

		if resp == "" {
			d.logger.Warn("empty response from LLM", "threadID", threadID)
			resp = "Sorry, I cannot respond to that. Please try again."
		}

		// Send Pedro's response
		m, err := d.Session.ChannelMessageSend(threadID, resp)
		if err != nil {
			d.logger.Error("error sending response to thread", "error", err.Error(), "threadID", threadID)
			return
		}
		metrics.DiscordMessageSent.Add(1)

		// Store Pedro's response in database
		pedroMsg := types.DiscordAskMessage{
			ThreadID:      threadID,
			Username:      "Pedro",
			Message:       resp,
			ThreadTimeout: 30,
			IsFromPedro:   true,
		}
		d.handleDBerror(d.db.InsertDiscordAskPedro(ctx, pedroMsg))

		lastMessageID = m.ID
	}

	d.logger.Debug("reached maximum iterations for thread", "threadID", threadID)
}