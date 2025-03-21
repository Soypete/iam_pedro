package discord

import (
	"fmt"
	"os"

	"github.com/Soypete/twitch-llm-bot/ai"
	"github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/bwmarrin/discordgo"
)

type Client struct {
	Session *discordgo.Session
	llm     ai.Chatter
	db      database.DiscordWriter
	logger  *logging.Logger
}

// Setup function is responsible for setting up the discord bot and connecting it to pedroGPT.
func Setup(llm ai.Chatter, db database.DiscordWriter, logger *logging.Logger) (Client, error) {
	authToken := os.Getenv("DISCORD_SECRET")
	session, err := discordgo.New("Bot " + authToken)
	if err != nil {
		logger.Error("error creating discord session", "error", err.Error())
		return Client{}, fmt.Errorf("error creating discord session: %w", err)
	}

	c := Client{
		Session: session,
		llm:     llm,
		db:      db,
		logger:  logger,
	}
	// opens websocket connection
	err = session.Open()
	if err != nil {
		logger.Error("error opening connection to discord", "error", err.Error())
		return Client{}, fmt.Errorf("error opening connection to discord: %w", err)
	}

	logger.Info("registering discord commands")
	for _, v := range AddCommands() {
		_, err := session.ApplicationCommandCreate(session.State.User.ID, "", v)
		if err != nil {
			logger.Error("error creating command", "command", v.Name, "error", err.Error())
			return Client{}, fmt.Errorf("error creating command: %w", err)
		}
		logger.Debug("registered command", "command", v.Name)
	}

	commandHandlers := c.MakeCommandHandlers()
	// after the commands are registered we can add the handlers
	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	return c, nil
}
