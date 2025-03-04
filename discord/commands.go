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
		"help":        help,
		"ask_pedro":   d.askPedro,
		"stump_pedro": d.stumpPedro,
	}
}

func help(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "To ask Pedro a question, use the /ask_pedro command. To stump Pedro with a game of 20 questions, use the /stump_pedro command.",
		},
	})
	if err != nil {
		fmt.Println(fmt.Errorf("error responding to help command: %w", err))
		return
	}
	metrics.DiscordMessageSent.Add(1)
}

func (d Client) askPedro(s *discordgo.Session, i *discordgo.InteractionCreate) {
	valid, err := messageValidatior(s, i)
	if !valid {
		fmt.Println(fmt.Errorf("error responding to askPedro command, no data: %w", err))
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
	message := database.TwitchMessage{
		Username: i.Interaction.Member.User.Username,
		Text:     text,
	}
	//Insert the message into the database
	messageID, err := d.db.InsertMessage(context.Background(), message)
	if err != nil {
		fmt.Println(fmt.Errorf("error inserting message into database: %w", err))
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Failed to ask Pedro. Please try again later",
			},
		})
		return
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Processing your question...",
		},
	})
	if err != nil {
		fmt.Println(fmt.Errorf("error responding to askPedro command: %w", err))
		return
	}
	metrics.DiscordMessageSent.Add(1)
	resp, err := d.llm.SingleMessageResponse(context.Background(), message, messageID)
	if err != nil {
		fmt.Println(fmt.Errorf("error calling llm | single message response: %w", err))
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Failed to ask Pedro. Please try again later",
			},
		})
		return
	}

	if resp == "" {
		resp = "Sorry, I cannot respond to that. Please try again."
	}

	userNumber := i.Interaction.Member.User.ID
	// TODO: break this in to a function we can test
	msgText := fmt.Sprintf("<@%s>:\nQuestion: %s\nResponse: %s", userNumber, text, resp)
	_, err = d.Session.ChannelMessageSend(i.Interaction.ChannelID, msgText)
	if err != nil {
		fmt.Println(fmt.Errorf("error sending message to channel: %w", err))
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
		fmt.Println(fmt.Errorf("error responding to stumpPedro command, no data: %w", err))
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
	message := database.TwitchMessage{
		Username: i.Interaction.Member.User.Username,
		Text:     text,
	}
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "starting 20 questions game",
		},
	})
	if err != nil {
		fmt.Println(fmt.Errorf("error starting to stumpPedro command: %w", err))
		return
	}

	go d.play20Questions(i.Interaction.ChannelID, message)

	metrics.DiscordMessageSent.Add(1)
}

func (d Client) play20Questions(channelID string, message database.TwitchMessage) {
	thing := message.Text
	ctx := context.Background()
	resp, err := d.llm.Play20Questions(ctx, message, uuid.New())
	if err != nil {
		fmt.Println(fmt.Errorf("error playing 20 questions: %w", err))
		return
	}
	m, err := d.Session.ChannelMessageSend(channelID, "Playing 20 questions with Pedro. Pedro will ask yes or no questions to guess what you are thinking. Please respond with yes or no in the thread to continue the game. You have 10 seconds to respond to each question or Pedro wins!")
	if err != nil {
		fmt.Println(fmt.Errorf("error sending message to channel: %w", err))
		return
	}
	metrics.DiscordMessageSent.Add(1)
	thread, err := d.Session.MessageThreadStart(m.ChannelID, m.ID, "20 Questions with Pedro: "+message.Username, 1440)
	if err != nil {
		fmt.Println(fmt.Errorf("error starting thread: %w", err))
		return
	}

	m, err = d.Session.ChannelMessageSend(thread.ID, resp)
	if err != nil {
		fmt.Println(fmt.Errorf("error sending message to thread: %w", err))
		return
	}
	metrics.DiscordMessageSent.Add(1)

	lastMessageID := m.ID
	for questionNumber := 1; questionNumber <= 20; questionNumber++ {
		// get the user response
		time.Sleep(10 * time.Second)
		messageList, err := d.Session.ChannelMessages(thread.ID, 100, "", lastMessageID, "")
		if err != nil {
			fmt.Println(fmt.Errorf("error getting thread messages: %w", err))
			return
		}

		if len(messageList) == 0 {
			_, err = d.Session.ChannelMessageSend(thread.ID, "Game over. You did not respond in time. Pedro wins!")
			d.llm.End20Questions()
			if err != nil {
				fmt.Println(fmt.Errorf("error sending message to thread: %w", err))
				return
			}
			metrics.DiscordMessageSent.Add(1)
		}
		m = messageList[0]
		message := database.TwitchMessage{
			Username: m.Author.Username,
			Text:     m.Content,
		}

		// call the LLM model
		resp, err := d.llm.Play20Questions(ctx, message, uuid.New())
		if err != nil {
			fmt.Println(fmt.Errorf("error calling llm | playing 20 questions: %w", err))
			return
		}

		// compare the response to the message
		if resp == fmt.Sprintf("I have guessed the thing you are thinking of. It is %s", thing) {
			_, err = d.Session.ChannelMessageSend(thread.ID, resp)
			if err != nil {
				fmt.Println(fmt.Errorf("error sending success message to thread: %w", err))
				return
			}
			break
		}

		// send the response to the user
		m, err = d.Session.ChannelMessageSend(thread.ID, resp)
		if err != nil {
			fmt.Println(fmt.Errorf("error sending message to thread: %w", err))
			return
		}
		metrics.DiscordMessageSent.Add(1)

		if strings.Contains(resp, "I have guessed the thing you are thinking of") {
			_, err = d.Session.ChannelMessageSend(thread.ID, resp)
			d.llm.End20Questions()
			if err != nil {
				fmt.Println(fmt.Errorf("error sending message to thread: %w", err))
				return
			}
			_, err = d.Session.ChannelMessageSend(thread.ID, "Game over. Pedro wins!")
			d.llm.End20Questions()
			if err != nil {
				fmt.Println(fmt.Errorf("error sending message to thread: %w", err))
				return
			}
			metrics.DiscordMessageSent.Add(1)
		}

		if questionNumber == 20 {
			_, err = d.Session.ChannelMessageSend(thread.ID, "Pedro use all the questions. Game over. You win!")
			d.llm.End20Questions()
			if err != nil {
				fmt.Println(fmt.Errorf("error sending message to thread: %w", err))
				return
			}
			metrics.DiscordMessageSent.Add(1)
		}

		lastMessageID = m.ID
	}
	_, _ = d.Session.ChannelMessageSend(thread.ID, "Game over. You win this round!")
}
