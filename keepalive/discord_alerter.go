package keepalive

import (
	"context"
	"fmt"

	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/bwmarrin/discordgo"
)

const (
	// Hardcoded channel for soypetetech Discord server
	alertChannelName = "pedrogpt"
)

// DiscordAlerter sends alerts to a Discord channel
type DiscordAlerter struct {
	session   *discordgo.Session
	channelID string
	userID    string // Discord user ID for mentions (e.g., "<@123456789>")
	logger    *logging.Logger
}

// NewDiscordAlerter creates a new Discord alerter using existing Discord bot token
func NewDiscordAlerter(token string, userID string, logger *logging.Logger) (*DiscordAlerter, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	// Open the session to access guilds
	err = session.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open Discord session: %w", err)
	}

	// Find the channel ID by name
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

// findChannelByName searches all guilds for a channel with the given name
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

// SendAlert sends an alert message to the configured Discord channel
func (da *DiscordAlerter) SendAlert(ctx context.Context, serviceName string, message string) error {
	// Format the message with user mention using Discord mention format
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

// Close closes the Discord session
func (da *DiscordAlerter) Close() error {
	return da.session.Close()
}
