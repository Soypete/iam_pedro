package discord

import (
	"fmt"
	"os"

	"github.com/Soypete/twitch-llm-bot/langchain"
	"github.com/bwmarrin/discordgo"
)

type Client struct {
	Session *discordgo.Session
	llm     langchain.Inferencer // should this be a pointer? // we want a pointer to be able to update a leverage chat history, but we probably want a different history than the twitch chat history
}

// Setup function is responsible for setting up the discord bot and connecting it to pedroGPT.
func Setup(llm langchain.Inferencer) (Client, error) {
	authToken := os.Getenv("DISCORD_SECRET")
	session, err := discordgo.New("Bot " + authToken)
	if err != nil {
		return Client{}, fmt.Errorf("error creating discord session: %w", err)
	}
	c := Client{
		Session: session,
		llm:     llm,
	}
	// opens websocket connection
	err = session.Open()
	if err != nil {
		return Client{}, fmt.Errorf("error opening connection to discord: %w", err)
	}
	// TODO: this needs to be handled properly
	// since it is not running in the context of this function we need
	// to handle the error in the main function

	for _, v := range AddCommands() {
		_, err := session.ApplicationCommandCreate(session.State.User.ID, "", v)
		if err != nil {
			return Client{}, fmt.Errorf("error creating command: %w", err)
		}
	}

	commandHandlers := c.MakeCommandHandlers()
	// after the commands are registered we can add the handlers
	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	fmt.Printf("%p\n", llm)

	return c, nil
}
