package discord

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/Soypete/twitch-llm-bot/types"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"
)

// TODO: we might move these to types if it helps with db enums
const (
	StatusStart     = "started"
	StatusAbandoned = "abandoned"
	StatusEnded     = "ended"
)

func (d Client) handleDBerror(err error) {
	if err != nil {
		d.logger.Error("failed db write", "err", err.Error())
	}
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

	data := i.Data.(discordgo.ApplicationCommandInteractionData) // assert the data type
	text := data.Options[0].StringValue()
	message := types.Discord20QuestionsGame{
		Username: i.Member.User.Username,
		Answer:   text,
		GameID:   uuid.New(),
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

	d.logger.Debug("starting 20 questions game", "channelID", i.ChannelID)
	go d.play20Questions(i.ChannelID, message)

	metrics.DiscordMessageSent.Add(1)
}

func (d Client) play20Questions(channelID string, message types.Discord20QuestionsGame) {
	thing := message.Answer
	username := message.Username
	ctx := context.Background()

	// Starting thread
	d.logger.Debug("starting play20Questions", "channelID", channelID)
	m, err := d.Session.ChannelMessageSend(channelID, "Playing 20 questions with Pedro. Pedro will ask yes or no questions to guess what you are thinking. Please respond with yes or no in the thread to continue the game. You have 30 seconds to respond to each question or Pedro wins!")
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

	// Create a game instance
	gameID := uuid.New()
	game := types.Discord20QuestionsGame{
		GameID:        gameID,
		Username:      username,
		Answer:        thing,
		ThreadID:      thread.ID,
		ThreadTimeout: 30, // Hardcoded to 30
	}

	// store game in db
	d.handleDBerror(d.db.CreateDiscord20Questions(context.Background(), game))

	// Get starting message from LLM
	startMessage, err := d.llm.Start20Questions(ctx, game)
	if err != nil {
		d.handleDBerror(d.db.AbandonDiscord20Questions(ctx, game.GameID.String(), 1))
		d.logger.Error("error starting 20 questions game", "error", err.Error())
		return
	}
	question := types.Discord20QuestionsMessage{
		GameID:   game.GameID,
		ThreadID: thread.ID,
		Question: startMessage,
	}

	m, err = d.Session.ChannelMessageSend(thread.ID, startMessage)
	if err != nil {
		d.handleDBerror(d.db.AbandonDiscord20Questions(ctx, game.GameID.String(), 1))
		d.logger.Error("error sending message to thread", "error", err.Error(), "threadID", thread.ID)
		return
	}
	metrics.DiscordMessageSent.Add(1)

	gameChat := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeAI, startMessage),
	}
	d.handleDBerror(d.db.InsertDiscordPlay20Questions(context.Background(), question))
	d.handleDBerror(d.db.UpdateDiscord20Questions(context.Background(), game.GameID.String(), 1))
	lastMessageID := m.ID

	// start loop that handles the game
	for questionNumber := 1; questionNumber <= 20; questionNumber++ {
		d.logger.Debug("processing question", "number", questionNumber, "threadID", thread.ID)

		// get the user response
		time.Sleep(30 * time.Second)
		messageList, err := d.Session.ChannelMessages(thread.ID, 100, "", lastMessageID, "")
		if err != nil {
			d.handleDBerror(d.db.AbandonDiscord20Questions(ctx, game.GameID.String(), questionNumber))
			d.logger.Error("error getting thread messages", "error", err.Error(), "threadID", thread.ID)
			return
		}
		metrics.DiscordMessageRecieved.Add(1)

		if len(messageList) == 0 {
			_, err = d.Session.ChannelMessageSend(thread.ID, "Game over. You did not respond in time. Pedro wins!")
			if err != nil {
				d.handleDBerror(d.db.AbandonDiscord20Questions(ctx, game.GameID.String(), questionNumber))
				d.logger.Error("error sending timeout message to thread", "error", err.Error(), "threadID", thread.ID)
				return
			}
			metrics.DiscordMessageSent.Add(1)
			return
		}

		m = messageList[0]
		question.Response = m.Content
		d.logger.Debug("received response", "response_length", len(question.Response), "threadID", thread.ID)

		// Store response in the database
		d.handleDBerror(d.db.InsertDiscordPlay20Questions(context.Background(), question))
		d.handleDBerror(d.db.UpdateDiscord20Questions(context.Background(), game.GameID.String(), questionNumber))
		gameChat = append(gameChat, llms.TextParts(llms.ChatMessageTypeHuman, m.Content))

		// call the LLM model
		resp, err := d.llm.Play20Questions(ctx, game.Username, gameChat)
		if err != nil {
			d.handleDBerror(d.db.AbandonDiscord20Questions(ctx, game.GameID.String(), questionNumber))
			d.logger.Error("error calling llm | playing 20 questions", "error", err.Error(), "questionNumber", questionNumber)
			return
		}
		gameChat = append(gameChat, llms.TextParts(llms.ChatMessageTypeAI, resp))

		// compare the response to the message
		if resp == fmt.Sprintf("I have guessed the thing you are thinking of. It is %s", thing) {
			_, err = d.Session.ChannelMessageSend(thread.ID, resp)
			if err != nil {
				d.handleDBerror(d.db.AbandonDiscord20Questions(ctx, game.GameID.String(), questionNumber))
				d.logger.Error("error sending success message to thread", "error", err.Error(), "threadID", thread.ID)
				return
			}
			break
		}

		// send the response to the user
		m, err = d.Session.ChannelMessageSend(thread.ID, resp)
		if err != nil {
			d.handleDBerror(d.db.AbandonDiscord20Questions(ctx, game.GameID.String(), questionNumber))
			d.logger.Error("error sending message to thread", "error", err.Error(), "threadID", thread.ID)
			return
		}
		metrics.DiscordMessageSent.Add(1)

		// TODO: we should do some kind of vector comparison here in case it is some kind of synonym
		//
		// TODO: spilt out into a function for testing.
		if strings.Contains(strings.ToLower(resp), strings.ToLower(game.Answer)) {
			_, err = d.Session.ChannelMessageSend(thread.ID, resp)
			if err != nil {
				d.handleDBerror(d.db.AbandonDiscord20Questions(ctx, game.GameID.String(), questionNumber))
				d.logger.Error("error sending guess message to thread", "error", err.Error(), "threadID", thread.ID)
				return
			}
			metrics.DiscordMessageSent.Add(1)
			d.handleDBerror(d.db.EndDiscord20Questions(ctx, game.GameID.String(), questionNumber))
			return
		}

		if questionNumber == 20 {
			_, err = d.Session.ChannelMessageSend(thread.ID, "Pedro used all the questions. Game over. You win!")
			if err != nil {
				d.handleDBerror(d.db.AbandonDiscord20Questions(ctx, game.GameID.String(), questionNumber))
				d.logger.Error("error sending max questions message to thread", "error", err.Error(), "threadID", thread.ID)
				return
			}

			metrics.DiscordMessageSent.Add(1)
			d.handleDBerror(d.db.EndDiscord20Questions(ctx, game.GameID.String(), questionNumber))
			_, err = d.Session.ChannelMessageSend(thread.ID, "Game over. You win this round!")
			if err != nil {
				d.logger.Error("error sending game over message to thread", "error", err.Error(), "threadID", thread.ID)
			}
			return
		}

		lastMessageID = m.ID
	}
}
