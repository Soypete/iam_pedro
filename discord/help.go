package discord

import (
	"time"

	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/bwmarrin/discordgo"
)

func (d Client) help(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Track command metrics
	start := time.Now()
	metrics.DiscordCommandTotal.WithLabelValues("help_pedro").Inc()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.DiscordCommandDuration.WithLabelValues("help_pedro").Observe(duration)
	}()

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "To ask Pedro a question, use the /ask_pedro command. To stump Pedro with a game of 20 questions, use the /stump_pedro command.",
		},
	})
	if err != nil {
		d.logger.Error("error responding to help command", "error", err.Error())
		metrics.DiscordCommandErrors.WithLabelValues("help_pedro").Inc()
		return
	}
	d.logger.Debug("help command handled successfully")
	metrics.DiscordMessageSent.Add(1)
}
