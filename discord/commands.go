package discord

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// SlashCommands that print a response to the user
func AddCommands() []*discordgo.ApplicationCommand {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "help",
			Description: "Get help with the bot",
		},
		{
			Name:        "ask_pedro",
			Description: "Ask Pedro a question and he will answer",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "question",
					Description: "What you want to ask Pedro?",
					Required:    true,
				},
			},
		},
		{
			Name:        "stump_pedro",
			Description: "Stump Pedro by playing a game of 20 questions",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "guess",
					Description: "The thing you want to stump Pedro with while playing 20 questions",
					Required:    true,
				},
			},
		},
	}
	return commands
}

// MakeCommandHandlers returns a map of command names to their respective functions
func (d Client) MakeCommandHandlers() map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"help":        d.help,
		"ask_pedro":   d.askPedro,
		"stump_pedro": d.stumpPedro,
	}
}

func (d Client) help(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "To ask Pedro a question, use the /ask_pedro command. To stump Pedro with a game of 20 questions, use the /stump_pedro command.",
		},
	})
	if err != nil {
		d.logger.Error("error responding to help command", "error", err.Error())
		return
	}
	d.logger.Debug("help command handled successfully")
	metrics.DiscordMessageSent.Add(1)
}

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
	username := i.Interaction.Member.User.Username

	message := database.TwitchMessage{
		Username: username,
		Text:     text,
	}

	// Insert the message into the database
	messageID, err := d.db.InsertMessage(context.Background(), message)
	if err != nil {
		d.logger.Error("failed to insert message into database", "error", err.Error())
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Failed to ask Pedro. Please try again later",
			},
		})
		return
	}

	d.logger.Debug("message inserted into database", "messageID", messageID)

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
	resp, err := d.llm.SingleMessageResponse(context.Background(), message, messageID)
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

	if resp.Text == "" {
		d.logger.Warn("empty response from LLM", "messageID", messageID)
		resp.Text = "Sorry, I cannot respond to that. Please try again."
	}

	userNumber := i.Interaction.Member.User.ID
	// TODO: break this in to a function we can test
	msgText := fmt.Sprintf("<@%s>:\nQuestion: %s\nResponse: %s", userNumber, text, resp.Text)
	_, err = d.Session.ChannelMessageSend(i.Interaction.ChannelID, msgText)
	if err != nil {
		d.logger.Error("error sending message to channel", "error", err.Error(), "channelID", i.Interaction.ChannelID)
		return
	}
	metrics.DiscordMessageSent.Add(1)

	// TODO: listen for the user to respond to the message with a follow up question to Pedro in a thread
}

func messageValidatior(s *discordgo.Session, i *discordgo.InteractionCreate) (bool, error) {
	if i.Interaction.Member == nil || i.Interaction.Member.User == nil {
		return false, fmt.Errorf("message does not contain member")
	}
	if i.Interaction.Data == nil {
		return false, fmt.Errorf("message does not contain data")
	}
	return true, nil
}

func (d Client) stumpPedro(s *discordgo.Session, i *discordgo.InteractionCreate) {
	response := "Failed to play 20 questions. Please try again later"
	valid, err := messageValidatior(s, i)
	if !valid {
		d.logger.Error("error responding to stumpPedro command, no data", "error", err.Error())
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: response,
			},
		})
		return
	}

	data := i.Interaction.Data.(discordgo.ApplicationCommandInteractionData) // assert the data type
	text := data.Options[0].StringValue()
	username := i.Interaction.Member.User.Username

	message := database.TwitchMessage{
		Username: username,
		Text:     text,
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "starting 20 questions game",
		},
	})
	if err != nil {
		d.logger.Error("error starting stumpPedro command", "error", err.Error())
		return
	}

	d.logger.Debug("starting 20 questions game", "channelID", i.Interaction.ChannelID)
	go d.play20Questions(i.Interaction.ChannelID, message)

	metrics.DiscordMessageSent.Add(1)
}

func (d Client) play20Questions(channelID string, message database.TwitchMessage) {
	thing := message.Text
	username := message.Username
	ctx := context.Background()

	d.logger.Debug("starting play20Questions", "channelID", channelID)

	gameID := uuid.New()
	resp, err := d.llm.Play20Questions(ctx, message, gameID)
	if err != nil {
		d.logger.Error("error playing 20 questions", "error", err.Error())
		return
	}

	m, err := d.Session.ChannelMessageSend(channelID, "Playing 20 questions with Pedro. Pedro will ask yes or no questions to guess what you are thinking. Please respond with yes or no in the thread to continue the game. You have 10 seconds to respond to each question or Pedro wins!")
	if err != nil {
		d.logger.Error("error sending message to channel", "error", err.Error(), "channelID", channelID)
		return
	}
	metrics.DiscordMessageSent.Add(1)

	threadTitle := "20 Questions with Pedro: " + username
	thread, err := d.Session.MessageThreadStart(m.ChannelID, m.ID, threadTitle, 1440)
	if err != nil {
		d.logger.Error("error starting thread", "error", err.Error(), "channelID", m.ChannelID)
		return
	}
	d.logger.Debug("started thread for 20 questions", "threadID", thread.ID)

	m, err = d.Session.ChannelMessageSend(thread.ID, resp)
	if err != nil {
		d.logger.Error("error sending message to thread", "error", err.Error(), "threadID", thread.ID)
		return
	}
	metrics.DiscordMessageSent.Add(1)

	lastMessageID := m.ID
	for questionNumber := 1; questionNumber <= 20; questionNumber++ {
		d.logger.Debug("processing question", "number", questionNumber, "threadID", thread.ID)

		// get the user response
		time.Sleep(10 * time.Second)
		messageList, err := d.Session.ChannelMessages(thread.ID, 100, "", lastMessageID, "")
		if err != nil {
			d.logger.Error("error getting thread messages", "error", err.Error(), "threadID", thread.ID)
			return
		}

		if len(messageList) == 0 {
			_, err = d.Session.ChannelMessageSend(thread.ID, "Game over. You did not respond in time. Pedro wins!")
			d.llm.End20Questions()
			if err != nil {
				d.logger.Error("error sending timeout message to thread", "error", err.Error(), "threadID", thread.ID)
				return
			}
			metrics.DiscordMessageSent.Add(1)
			return
		}

		m = messageList[0]
		message := database.TwitchMessage{
			Username: m.Author.Username,
			Text:     m.Content,
		}
		d.logger.Debug("received response", "response_length", len(message.Text))

		// call the LLM model
		resp, err := d.llm.Play20Questions(ctx, message, gameID)
		if err != nil {
			d.logger.Error("error calling llm | playing 20 questions", "error", err.Error(), "questionNumber", questionNumber)
			return
		}

		// compare the response to the message
		if resp == fmt.Sprintf("I have guessed the thing you are thinking of. It is %s", thing) {
			_, err = d.Session.ChannelMessageSend(thread.ID, resp)
			if err != nil {
				d.logger.Error("error sending success message to thread", "error", err.Error(), "threadID", thread.ID)
				return
			}
			break
		}

		// send the response to the user
		m, err = d.Session.ChannelMessageSend(thread.ID, resp)
		if err != nil {
			d.logger.Error("error sending message to thread", "error", err.Error(), "threadID", thread.ID)
			return
		}
		metrics.DiscordMessageSent.Add(1)

		if strings.Contains(resp, "I have guessed the thing you are thinking of") {
			_, err = d.Session.ChannelMessageSend(thread.ID, resp)
			d.llm.End20Questions()
			if err != nil {
				d.logger.Error("error sending guess message to thread", "error", err.Error(), "threadID", thread.ID)
				return
			}
			_, err = d.Session.ChannelMessageSend(thread.ID, "Game over. Pedro wins!")
			d.llm.End20Questions()
			if err != nil {
				d.logger.Error("error sending game over message to thread", "error", err.Error(), "threadID", thread.ID)
				return
			}
			metrics.DiscordMessageSent.Add(1)
			return
		}

		if questionNumber == 20 {
			_, err = d.Session.ChannelMessageSend(thread.ID, "Pedro used all the questions. Game over. You win!")
			d.llm.End20Questions()
			if err != nil {
				d.logger.Error("error sending max questions message to thread", "error", err.Error(), "threadID", thread.ID)
				return
			}
			metrics.DiscordMessageSent.Add(1)
			return
		}

		lastMessageID = m.ID
	}

	_, _ = d.Session.ChannelMessageSend(thread.ID, "Game over. You win this round!")
}
