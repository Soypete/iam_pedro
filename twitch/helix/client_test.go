package helix

import (
	"testing"

	"github.com/Soypete/twitch-llm-bot/logging"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with default logger", func(t *testing.T) {
		client := NewClient("clientID", "token", "broadcaster", "moderator", nil)
		if client.clientID != "clientID" {
			t.Errorf("expected clientID 'clientID', got %s", client.clientID)
		}
		if client.accessToken != "token" {
			t.Errorf("expected accessToken 'token', got %s", client.accessToken)
		}
		if client.broadcasterID != "broadcaster" {
			t.Errorf("expected broadcasterID 'broadcaster', got %s", client.broadcasterID)
		}
		if client.moderatorID != "moderator" {
			t.Errorf("expected moderatorID 'moderator', got %s", client.moderatorID)
		}
	})

	t.Run("creates client with custom logger", func(t *testing.T) {
		logger := logging.Default()
		client := NewClient("clientID", "token", "broadcaster", "moderator", logger)
		if client.logger != logger {
			t.Error("expected logger to be set")
		}
	})

	t.Run("sets default HTTP timeout", func(t *testing.T) {
		client := NewClient("clientID", "token", "broadcaster", "moderator", nil)
		if client.httpClient == nil {
			t.Error("expected non-nil httpClient")
		}
	})
}

func TestClientUpdateToken(t *testing.T) {
	t.Run("updates access token", func(t *testing.T) {
		client := NewClient("clientID", "oldToken", "broadcaster", "moderator", nil)
		client.UpdateToken("newToken")
		if client.accessToken != "newToken" {
			t.Errorf("expected accessToken 'newToken', got %s", client.accessToken)
		}
	})
}

func TestUserDataStruct(t *testing.T) {
	t.Run("user data has required fields", func(t *testing.T) {
		user := UserData{
			ID:          "123",
			Login:       "testuser",
			DisplayName: "TestUser",
			Type:        "admin",
		}

		if user.ID != "123" {
			t.Errorf("expected ID '123', got %s", user.ID)
		}
		if user.Login != "testuser" {
			t.Errorf("expected Login 'testuser', got %s", user.Login)
		}
	})
}

func TestStreamStatusStruct(t *testing.T) {
	t.Run("stream status has required fields", func(t *testing.T) {
		status := StreamStatus{
			ID:        "stream123",
			UserID:    "user123",
			UserLogin: "testuser",
			UserName:  "TestUser",
			GameID:    "456",
			GameName:  "Go Programming",
			Type:      "live",
			Title:     "Learning Go",
			IsMature:  false,
		}

		if status.Type != "live" {
			t.Errorf("expected Type 'live', got %s", status.Type)
		}
		if status.IsMature {
			t.Error("expected IsMature to be false")
		}
	})

	t.Run("stream status handles offline", func(t *testing.T) {
		status := StreamStatus{
			Type: "",
		}

		if status.Type != "" {
			t.Errorf("expected Type '', got %s", status.Type)
		}
	})
}

func TestPollChoiceStruct(t *testing.T) {
	t.Run("poll choice has title", func(t *testing.T) {
		choice := PollChoice{Title: "Yes"}
		if choice.Title != "Yes" {
			t.Errorf("expected Title 'Yes', got %s", choice.Title)
		}
	})
}

func TestPredictionOutcomeStruct(t *testing.T) {
	t.Run("prediction outcome has title", func(t *testing.T) {
		outcome := PredictionOutcome{Title: "Team A wins"}
		if outcome.Title != "Team A wins" {
			t.Errorf("expected Title 'Team A wins', got %s", outcome.Title)
		}
	})
}

func TestChatSettingsStruct(t *testing.T) {
	t.Run("chat settings can be created", func(t *testing.T) {
		emoteMode := true
		slowMode := false

		settings := ChatSettings{
			EmoteMode:      &emoteMode,
			SlowMode:       &slowMode,
			SubscriberMode: func(b bool) *bool { return &b }(true),
		}

		if settings.EmoteMode == nil || !*settings.EmoteMode {
			t.Error("expected EmoteMode to be true")
		}
		if settings.SlowMode == nil || *settings.SlowMode {
			t.Error("expected SlowMode to be false")
		}
	})
}
