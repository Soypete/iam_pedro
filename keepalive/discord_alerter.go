package keepalive

import (
	"context"
	"fmt"

	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/bwmarrin/discordgo"
)

const (
	alertChannelName = "pedrogpt"
)

type DiscordAlerter struct {
	session   *discordgo.Session
	channelID string
	userID    string
	logger    *logging.Logger
}

func NewDiscordAlerter(token string, userID string, logger *logging.Logger) (*DiscordAlerter, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	err = session.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open Discord session: %w", err)
	}

	channelID, err := findChannelByName(session, alertChannelName)
	if err != nil {
		if closeErr := session.Close(); closeErr != nil {
			logger.Error("failed to close Discord session", "error", closeErr.Error())
		}
		return nil, fmt.Errorf("failed to find channel %s: %w", alertChannelName, err)
	}

	logger.Info("Discord alerter initialized", "channel", alertChannelName, "channelID", channelID, "userID", userID)

	return &DiscordAlerter{
		session:   session,
		channelID: channelID,
		userID:    userID,
		logger:    logger,
	}, nil
}

func findChannelByName(session *discordgo.Session, channelName string) (string, error) {
	for _, guild := range session.State.Guilds {
		channels, err := session.GuildChannels(guild.ID)
		if err != nil {
			continue
		}

		for _, channel := range channels {
			if channel.Name == channelName {
				return channel.ID, nil
			}
		}
	}

	return "", fmt.Errorf("channel %s not found in any guild", channelName)
}

func (da *DiscordAlerter) SendAlert(ctx context.Context, serviceName string, message string) error {
	var alertMessage string
	if da.userID != "" {
		alertMessage = fmt.Sprintf("<@%s> **Alert:** %s", da.userID, message)
	} else {
		alertMessage = fmt.Sprintf("**Alert:** %s", message)
	}

	_, err := da.session.ChannelMessageSend(da.channelID, alertMessage)
	if err != nil {
		da.logger.Error("failed to send Discord alert",
			"error", err.Error(),
			"service", serviceName,
			"channel_id", da.channelID)
		return fmt.Errorf("failed to send Discord message: %w", err)
	}

	da.logger.Info("Discord alert sent",
		"service", serviceName,
		"channel_id", da.channelID)

	return nil
}

func (da *DiscordAlerter) Close() error {
	return da.session.Close()
}
