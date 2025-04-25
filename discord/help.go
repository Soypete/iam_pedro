package discord

import (
	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/bwmarrin/discordgo"
)

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
