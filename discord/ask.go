package discord

import (
	"context"
	"fmt"

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
	message := types.DiscordAskMessage{
		Username:      i.Interaction.Member.User.Username,
		Message:       text,
		ThreadTimeout: 0,
		IsFromPedro:   false,
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

	if resp == "" {
		d.logger.Warn("empty response from LLM", "messageID", messageID)
		resp = "Sorry, I cannot respond to that. Please try again."
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
	_, err = d.Session.ChannelMessageSend(thread.ID, resp)
	if err != nil {
		d.logger.Error("error sending response to thread", "error", err.Error(), "threadID", thread.ID)
		return
	}
	metrics.DiscordMessageSent.Add(1)

	// Update message with thread ID for database storage
	message.ThreadID = thread.ID
	message = types.DiscordAskMessage{
		Username:      i.Interaction.Member.User.Username,
		Message:       resp,
		ThreadID:      thread.ID,
		ThreadTimeout: 0,
		IsFromPedro:   true,
	}

	d.handleDBerror(d.db.InsertDiscordAskPedro(context.Background(), message))
}
