package types

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"
)

func TestTwitchMessageStruct(t *testing.T) {
	t.Run("TwitchMessage has required fields", func(t *testing.T) {
		msg := TwitchMessage{
			Username:   "testuser",
			Text:       "Hello Pedro!",
			IsCommand:  false,
			StopReason: "stop",
			Time:       time.Now(),
			UUID:       uuid.New(),
		}

		if msg.Username != "testuser" {
			t.Errorf("expected testuser, got %s", msg.Username)
		}
		if msg.Text != "Hello Pedro!" {
			t.Errorf("expected Hello Pedro!, got %s", msg.Text)
		}
	})

	t.Run("TwitchMessage can have nil UUID", func(t *testing.T) {
		msg := TwitchMessage{
			Username: "testuser",
			Text:     "Hello",
		}
		_ = msg.UUID
	})

	t.Run("TwitchMessage can have WebSearch", func(t *testing.T) {
		msg := TwitchMessage{
			Username: "testuser",
			Text:     "search query",
			WebSearch: &WebSearchRequest{
				Query: "search term",
			},
		}
		if msg.WebSearch == nil {
			t.Error("expected WebSearch to be set")
		}
		if msg.WebSearch.Query != "search term" {
			t.Errorf("expected search term, got %s", msg.WebSearch.Query)
		}
	})
}

func TestWebSearchRequestStruct(t *testing.T) {
	t.Run("WebSearchRequest has required fields", func(t *testing.T) {
		req := WebSearchRequest{
			Query: "Go programming language",
			OriginalMsg: TwitchMessage{
				Username: "asker",
				Text:     "What is Go?",
			},
			ChatHistory: []llms.MessageContent{},
		}

		if req.Query != "Go programming language" {
			t.Errorf("expected Go programming language, got %s", req.Query)
		}
		if req.OriginalMsg.Username != "asker" {
			t.Errorf("expected asker, got %s", req.OriginalMsg.Username)
		}
	})
}
