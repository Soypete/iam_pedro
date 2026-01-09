package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/Soypete/twitch-llm-bot/types"
)

// FindSimilarMessages finds similar messages from chat history using vector similarity
func (p *Postgres) FindSimilarMessages(ctx context.Context, embedding []float32, limit int, minSimilarity float64) ([]types.TwitchMessage, error) {
	if limit <= 0 {
		limit = 10
	}
	if minSimilarity <= 0 {
		minSimilarity = 0.7 // Default threshold for similarity
	}

	vec := arrayToString(embedding)

	query := `
		SELECT
			username,
			message as text,
			isCommand,
			stop_reason,
			created_at as time,
			uuid,
			1 - (embedding <=> $1) as similarity
		FROM twitch_chat
		WHERE embedding IS NOT NULL
			AND 1 - (embedding <=> $1) >= $2
		ORDER BY embedding <=> $1
		LIMIT $3
	`

	type messageWithSimilarity struct {
		types.TwitchMessage
		Similarity float64 `db:"similarity"`
	}

	var results []messageWithSimilarity
	err := p.connections.SelectContext(ctx, &results, query, vec, minSimilarity, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query similar messages: %w", err)
	}

	// Convert to regular TwitchMessage slice
	messages := make([]types.TwitchMessage, len(results))
	for i, r := range results {
		messages[i] = r.TwitchMessage
	}

	return messages, nil
}

// StoreMessageEmbedding stores an embedding for a chat message
func (p *Postgres) StoreMessageEmbedding(ctx context.Context, messageUUID string, embedding []float32) error {
	vec := arrayToString(embedding)

	query := `
		UPDATE twitch_chat
		SET embedding = $1
		WHERE uuid = $2
	`

	_, err := p.connections.ExecContext(ctx, query, vec, messageUUID)
	if err != nil {
		return fmt.Errorf("failed to store embedding: %w", err)
	}

	return nil
}

// arrayToString converts a float32 slice to PostgreSQL vector format
func arrayToString(arr []float32) string {
	if len(arr) == 0 {
		return "[]"
	}

	var builder strings.Builder
	builder.WriteString("[")
	for i, v := range arr {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(fmt.Sprintf("%f", v))
	}
	builder.WriteString("]")
	return builder.String()
}
