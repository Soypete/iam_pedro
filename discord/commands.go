package discord

import (
	"context"
	"fmt"
	"time"

	"github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
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
			Description: "Ask Pedro a questio and he will answer",
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
		"ask_pedro":   askPedro,
		"stump_pedro": d.stumpPedro,
	}
}

func help(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "help command is not implemented yet.",
		},
	})
	if err != nil {
		spew.Dump(err)
		return
	}
	metrics.DiscordMessageSent.Add(1)
}

func askPedro(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "askPedro command is not implemented yet.",
		},
	})
	if err != nil {
		spew.Dump(err)
		return
	}
	metrics.DiscordMessageSent.Add(1)
}

func (d Client) stumpPedro(s *discordgo.Session, i *discordgo.InteractionCreate) {
	response := "Failed to play 20 questions. Please try again later"
	if i.Interaction.Member == nil || i.Interaction.Member.User == nil {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: response,
			},
		})
		if err != nil {
			spew.Dump(err)
		}
		return
	}
	if i.Interaction.Data == nil {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: response,
			},
		})
		if err != nil {
			spew.Dump(err)
		}
		return
	}
	data := i.Interaction.Data.(discordgo.ApplicationCommandInteractionData) // assert the data type
	text := data.Options[0].StringValue()
	message := database.TwitchMessage{
		Username: i.Interaction.Member.User.Username,
		Text:     text,
	}
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponsePong,
	})
	if err != nil {
		spew.Dump(err)
	}

	go d.play20Questions(i.Interaction.ChannelID, message)

	metrics.DiscordMessageSent.Add(1)
}

// TODO: we should duration tracking for the game
func (d Client) play20Questions(channelID string, message database.TwitchMessage) {
	ctx := context.Background()
	resp, err := d.llm.Play20Questions(ctx, message, uuid.New())
	if err != nil {
		fmt.Println(err)
		return
	}
	m, err := d.Session.ChannelMessageSend(channelID, "Playing 20 questions with Pedro. Pedro will ask yes or no questions to guess what you are thinking. Please respond with yes or no in the thread to continue the game. You have 10 seconds to respond to each question or Pedro wins!")
	if err != nil {
		spew.Dump(err)
		return
	}
	metrics.DiscordMessageSent.Add(1)
	fmt.Println(resp)
	thread, err := d.Session.MessageThreadStart(m.ChannelID, m.ID, "20 Questions with Pedro: "+message.Username, 1440)
	if err != nil {
		spew.Dump(err)
		return
	}

	m, err = d.Session.ChannelMessageSend(thread.ID, resp)
	if err != nil {
		spew.Dump(err)
		return
	}
	metrics.DiscordMessageSent.Add(1)
	for questionNumber := 1; questionNumber <= 20; questionNumber++ {
		// get the user response
		time.Sleep(10 * time.Second)
		messageList, err := d.Session.ChannelMessages(thread.ID, 1, "", m.ID, "")
		if err != nil {
			spew.Dump(err)
			return
		}
		fmt.Println(len(messageList))

		if len(messageList) == 0 {
			_, err = d.Session.ChannelMessageSend(thread.ID, "Game over. You did not respond in time. Pedro wins!")
			d.llm.End20Questions()
			if err != nil {
				spew.Dump(err)
				return
			}
			metrics.DiscordMessageSent.Add(1)
		}
		if len(messageList) < 1 {
			return
		}
		m = messageList[0]
		message := database.TwitchMessage{
			Username: m.Author.Username,
			Text:     m.Content,
		}

		// call the LLM model
		resp, err := d.llm.Play20Questions(ctx, message, uuid.New())
		if err != nil {
			fmt.Println(err)
			return
		}

		// compare the response to the message
		if resp == fmt.Sprintf("I have guessed the thing you are thinking of. It is %s", message.Text) {
			_, err = d.Session.ChannelMessageSend(thread.ID, resp)
			if err != nil {
				spew.Dump(err)
				return
			}
			break
		}

		// send the response to the user
		m, err = d.Session.ChannelMessageSend(thread.ID, resp)
		if err != nil {
			spew.Dump(err)
			return
		}
		metrics.DiscordMessageSent.Add(1)

	}
}
