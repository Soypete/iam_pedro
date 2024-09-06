package database

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"
)

type ResponseWriter interface {
	InsertResponse(ctx context.Context, resp *llms.ContentResponse, messageID uuid.UUID, modelName string) error
}

func (p *Postgres) InsertResponse(ctx context.Context, resp *llms.ContentResponse, messageID uuid.UUID, modelName string) error {
	var isUsed bool
	for i, choice := range resp.Choices {
		if i == 0 {
			isUsed = true
		}
		query := "INSERT INTO bot_response (model_name, response, stop_reason, was_successful, chat_id) VALUES ($1, $2, $3, $4, $5)"
		_, err := p.connections.ExecContext(ctx, query, modelName, choice.Content, choice.StopReason, isUsed, messageID)
		if err != nil {
			log.Println("error upserting response: ", err)
			return fmt.Errorf("error upserting response: %w", err)
		}
	}
	return nil
}
