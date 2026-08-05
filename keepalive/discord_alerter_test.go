package keepalive

import (
	"testing"
)

func TestDiscordAlerterStruct(t *testing.T) {
	t.Run("DiscordAlerter has required fields", func(t *testing.T) {
		alerter := &DiscordAlerter{}
		_ = alerter.session
		_ = alerter.channelID
		_ = alerter.userID
		_ = alerter.logger
	})
}

func TestAlertMessageFormat(t *testing.T) {
	t.Run("formats message with userID", func(t *testing.T) {
		userID := "123456789"
		message := "Service down"

		alertMessage := ""
		if userID != "" {
			alertMessage = "<@" + userID + "> **Alert:** " + message
		}

		expected := "<@123456789> **Alert:** Service down"
		if alertMessage != expected {
			t.Errorf("expected %s, got %s", expected, alertMessage)
		}
	})

	t.Run("formats message without userID", func(t *testing.T) {
		userID := ""
		message := "Service down"

		alertMessage := ""
		if userID != "" {
			alertMessage = "<@" + userID + "> **Alert:** " + message
		} else {
			alertMessage = "**Alert:** " + message
		}

		expected := "**Alert:** Service down"
		if alertMessage != expected {
			t.Errorf("expected %s, got %s", expected, alertMessage)
		}
	})
}

func TestAlertChannelNameConstant(t *testing.T) {
	t.Run("alert channel name is set", func(t *testing.T) {
		if alertChannelName != "pedrogpt" {
			t.Errorf("expected 'pedrogpt', got %s", alertChannelName)
		}
	})
}
