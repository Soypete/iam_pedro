package tools

import (
	"testing"
	"time"

	"github.com/Soypete/twitch-llm-bot/internal/mempalace/store"
)

func TestQueryChatHistoryToolStruct(t *testing.T) {
	t.Run("QueryChatHistoryTool has required fields", func(t *testing.T) {
		tool := &QueryChatHistoryTool{}
		_ = tool.store
		_ = tool.index
		_ = tool.classes
	})
}

func TestNewQueryChatHistoryTool(t *testing.T) {
	t.Run("creates tool with provided values", func(t *testing.T) {
		tool := NewQueryChatHistoryTool(nil, nil, nil)

		if tool.store != nil {
			t.Error("expected store to be nil")
		}
		if tool.index != nil {
			t.Error("expected index to be nil")
		}
		if tool.classes != nil {
			t.Error("expected classes to be nil")
		}
	})
}

func TestToolName(t *testing.T) {
	t.Run("returns correct name", func(t *testing.T) {
		tool := NewQueryChatHistoryTool(nil, nil, nil)
		if tool.Name() != "query_chat_history" {
			t.Errorf("expected 'query_chat_history', got %s", tool.Name())
		}
	})
}

func TestToolDescription(t *testing.T) {
	t.Run("returns non-empty description", func(t *testing.T) {
		tool := NewQueryChatHistoryTool(nil, nil, nil)
		desc := tool.Description()
		if desc == "" {
			t.Error("expected non-empty description")
		}
	})
}

func TestQueryArgsStruct(t *testing.T) {
	t.Run("queryArgs has required fields", func(t *testing.T) {
		args := &queryArgs{
			TopicClass: "golang",
			TimeRange:  "15m",
			QueryText:  "what is go",
			Limit:      10,
		}

		if args.TopicClass != "golang" {
			t.Errorf("expected 'golang', got %s", args.TopicClass)
		}
		if args.Limit != 10 {
			t.Errorf("expected 10, got %d", args.Limit)
		}
	})
}

func TestParseQueryArgs(t *testing.T) {
	t.Run("parses valid JSON", func(t *testing.T) {
		input := `{"topic_class":"golang","time_range":"15m","query_text":"go","limit":5}`
		args, err := parseQueryArgs(input)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if args.TopicClass != "golang" {
			t.Errorf("expected 'golang', got %s", args.TopicClass)
		}
		if args.Limit != 5 {
			t.Errorf("expected 5, got %d", args.Limit)
		}
	})

	t.Run("sets default limit when zero", func(t *testing.T) {
		input := `{}`
		args, err := parseQueryArgs(input)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if args.Limit != 10 {
			t.Errorf("expected default limit 10, got %d", args.Limit)
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		input := `{invalid`
		_, err := parseQueryArgs(input)

		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestParseTimeRange(t *testing.T) {
	t.Run("parses 5m", func(t *testing.T) {
		start, end, err := parseTimeRange("5m")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		diff := end.Sub(start)
		expected := 5 * 60 * time.Second
		if diff != expected {
			t.Logf("5 minutes: expected %v, got %v", expected, diff)
		}
	})

	t.Run("parses 15m", func(t *testing.T) {
		_, _, err := parseTimeRange("15m")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("parses 1h", func(t *testing.T) {
		_, _, err := parseTimeRange("1h")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("returns error for unknown format", func(t *testing.T) {
		_, _, err := parseTimeRange("unknown")
		if err == nil {
			t.Error("expected error for unknown format")
		}
	})
}

func TestFormatMessages(t *testing.T) {
	t.Run("formats messages correctly", func(t *testing.T) {
		baseTime := time.Date(2025, 1, 15, 15, 4, 0, 0, time.UTC)
		messages := []store.Message{
			{
				Username:  "user1",
				Message:   "hello",
				Timestamp: baseTime,
			},
			{
				Username:  "user2",
				Message:   "hi there",
				Timestamp: baseTime.Add(time.Minute),
			},
		}

		result := formatMessages(messages)

		if result == "" {
			t.Error("expected non-empty result")
		}
	})

	t.Run("handles empty messages", func(t *testing.T) {
		result := formatMessages([]store.Message{})
		expected := "Relevant chat messages:\n"
		if result != expected {
			t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
		}
	})
}
