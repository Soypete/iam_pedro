package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
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

func messageValidatior(s *discordgo.Session, i *discordgo.InteractionCreate) (bool, error) {
	if i.Interaction.Member == nil || i.Interaction.Member.User == nil {
		return false, fmt.Errorf("message does not contain member")
	}
	if i.Interaction.Data == nil {
		return false, fmt.Errorf("message does not contain data")
	}
	return true, nil
}
