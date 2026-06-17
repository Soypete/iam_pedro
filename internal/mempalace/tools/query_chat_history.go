// Package tools provides LLM tool definitions for Mem Palace interactions.
//
// Currently provides query_chat_history tool that allows Pedro to search
// past chat messages by topic or keywords. The tool returns relevant
// messages with metadata (username, timestamp, topic).
//
// Tool schema follows OpenAI function calling format for compatibility
// with langchaingo.
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"time"

	"github.com/Soypete/twitch-llm-bot/internal/mempalace/ontology"
	"github.com/Soypete/twitch-llm-bot/internal/mempalace/store"
	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/tmc/langchaingo/llms"
)

type QueryChatHistoryTool struct {
	store   *store.Store
	index   *ontology.Index
	classes []ontology.Class
}

func NewQueryChatHistoryTool(store *store.Store, index *ontology.Index, classes []ontology.Class) *QueryChatHistoryTool {
	return &QueryChatHistoryTool{
		store:   store,
		index:   index,
		classes: classes,
	}
}

func (t *QueryChatHistoryTool) Name() string {
	return "query_chat_history"
}

func (t *QueryChatHistoryTool) Description() string {
	return `Query chat history from the Mem Palace session. Use this when users ask about topics discussed during the stream.
Arguments:
- topic_class: The topic category to search (e.g., "Go", "LLM Engineering", "DevOps"). If not sure, leave empty.
- time_range: Time range for search, format as "start,end" in RFC3339 or relative like "15m", "1h". Leave empty for all time.
- query_text: Specific text to search for in messages.
- limit: Maximum number of messages to return (default 10).`
}

func (t *QueryChatHistoryTool) Call(ctx context.Context, input string) (string, error) {
	args, err := parseQueryArgs(input)
	if err != nil {
		metrics.MempalaceToolCallsTotal.WithLabelValues("query_chat_history", "parse_error").Add(1)
		return "", err
	}

	topic := args.TopicClass
	if topic == "" && args.QueryText != "" && t.index != nil {
		results, err := t.index.Search(ctx, args.QueryText, 1)
		if err == nil && len(results) > 0 {
			topic = results[0].Term
		}
	}

	opts := store.QueryOpts{
		Topic:     topic,
		QueryText: args.QueryText,
		Limit:     args.Limit,
	}

	if args.TimeRange != "" {
		start, end, err := parseTimeRange(args.TimeRange)
		if err == nil {
			opts.TimeStart = &start
			opts.TimeEnd = &end
		}
	}

	messages, err := t.store.Query(ctx, opts)
	if err != nil {
		metrics.MempalaceToolCallsTotal.WithLabelValues("query_chat_history", "error").Add(1)
		return "", fmt.Errorf("failed to query chat history: %w", err)
	}

	if len(messages) == 0 {
		metrics.MempalaceToolCallsTotal.WithLabelValues("query_chat_history", "no_results").Add(1)
		return "No messages found matching the query.", nil
	}

	metrics.MempalaceToolCallsTotal.WithLabelValues("query_chat_history", "success").Add(1)

	result := formatMessages(messages)
	return result, nil
}

func GetQueryChatHistoryToolDefinition() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "query_chat_history",
			Description: "Query chat history from the Mem Palace session for relevant past discussions",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"topic_class": map[string]any{
						"type":        "string",
						"description": "The topic category to search (e.g., Go, LLM Engineering, DevOps)",
					},
					"time_range": map[string]any{
						"type":        "string",
						"description": "Time range for search, format as '15m', '1h', or 'start,end' in RFC3339",
					},
					"query_text": map[string]any{
						"type":        "string",
						"description": "Specific text to search for in messages",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of messages to return",
						"default":     10,
					},
				},
			},
		},
	}
}

type queryArgs struct {
	TopicClass string `json:"topic_class"`
	TimeRange  string `json:"time_range"`
	QueryText  string `json:"query_text"`
	Limit      int    `json:"limit"`
}

func parseQueryArgs(input string) (*queryArgs, error) {
	var args queryArgs
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		return nil, fmt.Errorf("failed to parse query args: %w", err)
	}
	if args.Limit == 0 {
		args.Limit = 10
	}
	return &args, nil
}

func parseTimeRange(tr string) (time.Time, time.Time, error) {
	now := time.Now()

	switch tr {
	case "5m":
		return now.Add(-5 * time.Minute), now, nil
	case "15m":
		return now.Add(-15 * time.Minute), now, nil
	case "30m":
		return now.Add(-30 * time.Minute), now, nil
	case "1h":
		return now.Add(-1 * time.Hour), now, nil
	case "2h":
		return now.Add(-2 * time.Hour), now, nil
	default:
		return now, now, fmt.Errorf("unsupported time range format")
	}
}

func formatMessages(messages []store.Message) string {
	result := "Relevant chat messages:\n"
	for _, msg := range messages {
		result += fmt.Sprintf("[%s] %s: %s\n",
			msg.Timestamp.Format("15:04"),
			msg.Username,
			msg.Message,
		)
	}
	return result
}
