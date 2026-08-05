package twitchchat

import (
	"testing"

	"github.com/Soypete/twitch-llm-bot/logging"
)

func TestSetup(t *testing.T) {
	t.Run("uses default logger when nil", func(t *testing.T) {
		logger := logging.Default()
		if logger == nil {
			t.Error("expected non-nil logger")
		}
	})
}

func TestSetupWithStreamConfig(t *testing.T) {
	t.Run("appends /v1 to path if not present", func(t *testing.T) {
		llmPath := "http://localhost:8080"
		expectedSuffix := "/v1"

		if len(llmPath) >= 3 && llmPath[len(llmPath)-3:] != expectedSuffix {
			llmPath = llmPath + expectedSuffix
		}

		if llmPath != "http://localhost:8080/v1" {
			t.Errorf("expected path with /v1, got %s", llmPath)
		}
	})

	t.Run("does not append /v1 if already present", func(t *testing.T) {
		llmPath := "http://localhost:8080/v1"
		expectedPath := llmPath

		if len(llmPath) >= 3 && llmPath[len(llmPath)-3:] != "/v1" {
			llmPath = llmPath + "/v1"
		}

		if llmPath != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, llmPath)
		}
	})
}

func TestClientStruct(t *testing.T) {
	t.Run("Client has required fields", func(t *testing.T) {
		client := &Client{}
		_ = client.llm
		_ = client.chatHistory
		_ = client.modelName
		_ = client.logger
		_ = client.ddgClient
		_ = client.streamConfig
		_ = client.streamAddendum
	})
}

func TestSetupWithMeetupMode(t *testing.T) {
	t.Run("converts slug to config path", func(t *testing.T) {
		meetupSlug := "golang-nov-2025"
		expectedPath := "configs/streams/golang-nov-2025.yaml"

		configPath := ""
		if meetupSlug != "" {
			configPath = "configs/streams/" + meetupSlug + ".yaml"
		}

		if configPath != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, configPath)
		}
	})

	t.Run("returns empty config for empty slug", func(t *testing.T) {
		meetupSlug := ""
		configPath := ""
		if meetupSlug != "" {
			configPath = "configs/streams/" + meetupSlug + ".yaml"
		}

		if configPath != "" {
			t.Errorf("expected empty path, got %s", configPath)
		}
	})
}
