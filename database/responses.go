package database

import (
	"context"
	"fmt"

	"github.com/Soypete/twitch-llm-bot/types"
)

type ResponseWriter interface {
	InsertResponse(ctx context.Context, resp types.TwitchMessage, modelName string) error
}

func (p *Postgres) InsertResponse(ctx context.Context, resp types.TwitchMessage, modelName string) error {
	p.logger.Debug("inserting LLM response into database", "messageID", resp.UUID, "model", modelName)

	query := "INSERT INTO bot_response (model_name, response, stop_reason, was_successful, chat_id) VALUES ($1, $2, $3, $4, $5)"
	_, err := p.connections.ExecContext(ctx, query, modelName, resp.Text, resp.StopReason, true, resp.UUID)
	if err != nil {
		p.logger.Error("error inserting response into database", "error", err.Error(), "messageID", resp.UUID)
		return fmt.Errorf("error upserting response: %w", err)
	}

	p.logger.Debug("all response choices inserted successfully", "messageID", resp.UUID)
	return nil
}
