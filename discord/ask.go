package discord

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/Soypete/twitch-llm-bot/types"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"
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
	message := types.DiscordAskMessage{
		Username:      i.Interaction.Member.User.Username,
		Message:       text,
		ThreadTimeout: 0,
		IsFromPedro:   false,
		Timestamp:     time.Now(),
	}

	//  Insert the message into the database
	d.handleDBerror(d.db.InsertDiscordAskPedro(context.Background(), message))
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

	d.logger.Debug("calling LLM for response", "messageID", messageID)
	resp, err := d.llm.SingleMessageResponse(context.Background(), message)
	if err != nil {
		d.logger.Error("error calling llm | single message response", "error", err.Error(), "messageID", messageID)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Failed to ask Pedro. Please try again later",
			},
		})
		return
	}

	if resp == nil || resp.Text == "" {
		d.logger.Warn("empty response from LLM", "messageID", messageID)
		resp = &types.DiscordResponse{
			Text: "Sorry, I cannot respond to that. Please try again.",
		}
	}

	userNumber := i.Interaction.Member.User.ID
	// Create initial message in channel
	initialMsg := fmt.Sprintf("<@%s> asked Pedro: %s", userNumber, text)
	m, err := d.Session.ChannelMessageSend(i.Interaction.ChannelID, initialMsg)
	if err != nil {
		d.logger.Error("error sending message to channel", "error", err.Error(), "channelID", i.Interaction.ChannelID)
		return
	}
	metrics.DiscordMessageSent.Add(1)

	// Create thread for the conversation
	threadTitle := "Ask Pedro: " + i.Interaction.Member.User.Username
	thread, err := d.Session.MessageThreadStart(m.ChannelID, m.ID, threadTitle, 1440)
	if err != nil {
		d.logger.Error("error starting thread", "error", err.Error(), "channelID", m.ChannelID)
		return
	}
	d.logger.Debug("started thread for ask pedro", "threadID", thread.ID)

	// Send Pedro's response in the thread
	_, err = d.Session.ChannelMessageSend(thread.ID, resp.Text)
	if err != nil {
		d.logger.Error("error sending response to thread", "error", err.Error(), "threadID", thread.ID)
		return
	}
	metrics.DiscordMessageSent.Add(1)

	// Check if web search is needed
	if resp.WebSearch != nil {
		d.logger.Debug("web search requested", "query", resp.WebSearch.Query, "messageID", messageID)

		// Execute web search asynchronously
		go func() {
			searchResp, err := d.llm.ExecuteWebSearch(context.Background(), resp.WebSearch)
			if err != nil {
				d.logger.Error("web search failed", "error", err.Error(), "query", resp.WebSearch.Query)
				return
			}

			// Send the search result as a follow-up message in the thread
			_, err = d.Session.ChannelMessageSend(thread.ID, searchResp)
			if err != nil {
				d.logger.Error("error sending search results to thread", "error", err.Error(), "threadID", thread.ID)
				return
			}
			metrics.DiscordMessageSent.Add(1)

			// Store the search response in the database
			searchMessage := types.DiscordAskMessage{
				Username:      "Pedro",
				Message:       searchResp,
				ThreadID:      thread.ID,
				ThreadTimeout: 0,
				IsFromPedro:   true,
				Timestamp:     time.Now(),
			}
			d.handleDBerror(d.db.InsertDiscordAskPedro(context.Background(), searchMessage))
		}()
	}

	// Store Pedro's response
	message = types.DiscordAskMessage{
		Username:      "Pedro",
		Message:       resp.Text,
		ThreadID:      thread.ID,
		ThreadTimeout: 0,
		IsFromPedro:   true,
		Timestamp:     time.Now(),
	}

	d.handleDBerror(d.db.InsertDiscordAskPedro(context.Background(), message))
}

// ThreadMessage represents a message in a thread with metadata
type ThreadMessage struct {
	ID        string
	Content   string
	Username  string
	Timestamp time.Time
	IsFromBot bool
}

// getThreadContext retrieves the full conversation history from a Discord thread
// Limits to the last 40 messages for performance and token management
func (d Client) getThreadContext(threadID string) ([]ThreadMessage, error) {
	var allMessages []ThreadMessage
	var lastMessageID string
	const maxMessages = 40 // Limit context to manage performance and token usage

	// Discord API limits to 100 messages per request, so we may need multiple calls
	for len(allMessages) < maxMessages {
		limit := 100
		if maxMessages-len(allMessages) < 100 {
			limit = maxMessages - len(allMessages)
		}

		messages, err := d.Session.ChannelMessages(threadID, limit, "", lastMessageID, "")
		if err != nil {
			d.logger.Error("failed to retrieve thread messages", "error", err.Error(), "threadID", threadID)
			return nil, fmt.Errorf("failed to retrieve thread messages: %w", err)
		}

		if len(messages) == 0 {
			break
		}

		// Convert Discord messages to our ThreadMessage format
		for _, msg := range messages {
			// Skip empty messages and system messages
			if strings.TrimSpace(msg.Content) == "" || msg.Type != discordgo.MessageTypeDefault {
				continue
			}

			timestamp, _ := discordgo.SnowflakeTimestamp(msg.ID)
			threadMsg := ThreadMessage{
				ID:        msg.ID,
				Content:   msg.Content,
				Username:  msg.Author.Username,
				Timestamp: timestamp,
				IsFromBot: msg.Author.Bot,
			}
			allMessages = append(allMessages, threadMsg)
		}

		// Set up for next iteration
		lastMessageID = messages[len(messages)-1].ID

		// If we got less than requested messages, we've reached the end
		if len(messages) < limit {
			break
		}
	}

	// Sort messages chronologically (oldest first)
	sort.Slice(allMessages, func(i, j int) bool {
		return allMessages[i].Timestamp.Before(allMessages[j].Timestamp)
	})

	d.logger.Debug("retrieved thread context", "threadID", threadID, "messageCount", len(allMessages))
	return allMessages, nil
}

// formatThreadContextForLLM converts thread messages to LLM conversation format
func (d Client) formatThreadContextForLLM(messages []ThreadMessage) []llms.MessageContent {
	var context []llms.MessageContent

	for _, msg := range messages {
		// Skip empty messages or system messages
		if strings.TrimSpace(msg.Content) == "" {
			continue
		}

		var messageType llms.ChatMessageType
		var content string

		if msg.IsFromBot {
			messageType = llms.ChatMessageTypeAI
			content = msg.Content
		} else {
			messageType = llms.ChatMessageTypeHuman
			// Include username for human messages to maintain context
			content = fmt.Sprintf("%s: %s", msg.Username, msg.Content)
		}

		context = append(context, llms.TextParts(messageType, content))
	}

	return context
}

// handleThreadReply processes replies in any thread
func (d Client) handleThreadReply(s *discordgo.Session, m *discordgo.MessageCreate) {
	d.logger.Info("handleThreadReply called", "messageID", m.ID, "author", m.Author.Username, "content", m.Content)

	// Skip if message is from a bot
	if m.Author.Bot {
		d.logger.Info("skipping bot message", "messageID", m.ID, "author", m.Author.Username)
		return
	}

	// Check if this is in a thread
	channel, err := s.Channel(m.ChannelID)
	if err != nil {
		d.logger.Error("failed to get channel info", "error", err.Error(), "channelID", m.ChannelID)
		return
	}

	d.logger.Info("channel details", "channelID", m.ChannelID, "channelType", channel.Type, "channelName", channel.Name, "publicThread", discordgo.ChannelTypeGuildPublicThread, "privateThread", discordgo.ChannelTypeGuildPrivateThread)

	// Only process thread channels
	if channel.Type != discordgo.ChannelTypeGuildPublicThread && channel.Type != discordgo.ChannelTypeGuildPrivateThread {
		d.logger.Info("skipping non-thread channel", "channelType", channel.Type, "channelID", m.ChannelID, "expectedTypes", []int{int(discordgo.ChannelTypeGuildPublicThread), int(discordgo.ChannelTypeGuildPrivateThread)})
		return
	}

	// Skip empty or very short messages
	if len(strings.TrimSpace(m.Content)) < 3 {
		d.logger.Info("skipping short message", "threadID", m.ChannelID, "content", m.Content, "length", len(strings.TrimSpace(m.Content)))
		return
	}

	d.logger.Info("proceeding to process thread reply", "threadID", m.ChannelID, "username", m.Author.Username)

	// Create context with timeout for the entire operation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get full thread context
	threadMessages, err := d.getThreadContext(m.ChannelID)
	if err != nil {
		d.logger.Error("failed to get thread context", "error", err.Error(), "threadID", m.ChannelID)
		// Send a fallback response even if we can't get context
		d.sendFallbackResponse(m.ChannelID, "I'm having trouble accessing the conversation history.")
		return
	}

	// Format context for LLM (excluding the current message to avoid duplication)
	var filteredMessages []ThreadMessage
	for _, msg := range threadMessages {
		if msg.ID != m.ID {
			filteredMessages = append(filteredMessages, msg)
		}
	}
	conversationHistory := d.formatThreadContextForLLM(filteredMessages)

	// Create message for LLM processing
	askMessage := types.DiscordAskMessage{
		ThreadID:      m.ChannelID,
		MessageID:     m.ID,
		Message:       m.Content,
		Username:      m.Author.Username,
		ThreadTimeout: 0,
		IsFromPedro:   false,
		Timestamp:     time.Now(),
	}

	// Store the user message
	if err := d.db.InsertDiscordAskPedro(ctx, askMessage); err != nil {
		d.logger.Error("failed to store user message", "error", err.Error(), "threadID", m.ChannelID)
	}

	// Get response from LLM with full context
	resp, err := d.llm.ThreadMessageResponse(ctx, askMessage, conversationHistory)
	if err != nil {
		d.logger.Error("error calling llm for thread response", "error", err.Error(), "threadID", m.ChannelID)
		d.sendFallbackResponse(m.ChannelID, "I'm having trouble generating a response right now.")
		return
	}

	if resp == "" {
		resp = "Sorry, I cannot respond to that. Please try again."
	}

	// Send Pedro's response
	sentMsg, err := d.Session.ChannelMessageSend(m.ChannelID, resp)
	if err != nil {
		d.logger.Error("error sending thread response", "error", err.Error(), "threadID", m.ChannelID)
		return
	}
	metrics.DiscordMessageSent.Add(1)

	// Store Pedro's response
	pedroMessage := types.DiscordAskMessage{
		ThreadID:        m.ChannelID,
		MessageID:       sentMsg.ID,
		Message:         resp,
		Username:        "Pedro",
		ThreadTimeout:   0,
		IsFromPedro:     true,
		ParentMessageID: m.ID,
		Timestamp:       time.Now(),
	}

	if err := d.db.InsertDiscordAskPedro(ctx, pedroMessage); err != nil {
		d.logger.Error("failed to store pedro response", "error", err.Error(), "threadID", m.ChannelID)
	}
}

// sendFallbackResponse sends a fallback message when errors occur
func (d Client) sendFallbackResponse(channelID, message string) {
	_, err := d.Session.ChannelMessageSend(channelID, message)
	if err != nil {
		d.logger.Error("error sending fallback response", "error", err.Error(), "channelID", channelID)
	} else {
		metrics.DiscordMessageSent.Add(1)
	}
}
