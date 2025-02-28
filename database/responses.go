package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"
)

type ResponseWriter interface {
	InsertResponse(ctx context.Context, resp *llms.ContentResponse, messageID uuid.UUID, modelName string) error
}

func (p *Postgres) InsertResponse(ctx context.Context, resp *llms.ContentResponse, messageID uuid.UUID, modelName string) error {
	p.logger.Debug("inserting LLM response into database", "messageID", messageID, "model", modelName, "choices", len(resp.Choices))

	var isUsed bool
	for i, choice := range resp.Choices {
		if i == 0 {
			isUsed = true
		}

		query := "INSERT INTO bot_response (model_name, response, stop_reason, was_successful, chat_id) VALUES ($1, $2, $3, $4, $5)"
		_, err := p.connections.ExecContext(ctx, query, modelName, choice.Content, choice.StopReason, isUsed, messageID)
		if err != nil {
			p.logger.Error("error inserting response into database", "error", err.Error(), "messageID", messageID, "choice", i)
			return fmt.Errorf("error upserting response: %w", err)
		}

		p.logger.Debug("response choice inserted", "messageID", messageID, "choice", i, "stopReason", choice.StopReason)
	}

	p.logger.Debug("all response choices inserted successfully", "messageID", messageID)
	return nil
}
