package database

import (
	"context"
	"fmt"

	"github.com/Soypete/twitch-llm-bot/types"
	"github.com/google/uuid"
)

// ChatResponseWriter is an interface that allows for both storing twitch chat chatHistory
// and Pedro's responses in the database.
type ChatResponseWriter interface {
	MessageWriter
	ResponseWriter
}

// MessageWriter is an interface for inserting twitch chat messages into the database.
type MessageWriter interface {
	InsertMessage(ctx context.Context, msg types.TwitchMessage) (uuid.UUID, error)
}

// InsertMessage inserts a message into the database and returns the message ID if successful.
func (p *Postgres) InsertMessage(ctx context.Context, msg types.TwitchMessage) (uuid.UUID, error) {
	p.logger.Debug("generating UUID for message")
	ID, err := uuid.NewUUID()
	if err != nil {
		p.logger.Error("error generating UUID", "error", err.Error())
		return uuid.UUID{}, fmt.Errorf("error generating UUID: %w", err)
	}
	msg.UUID = ID

	query := "INSERT INTO twitch_chat (username, message, isCommand, created_at, uuid) VALUES (:username, :message, :isCommand, :created_at, :uuid)"
	p.logger.Debug("inserting message into database", "messageID", ID)

	_, err = p.connections.NamedExecContext(ctx, query, msg)
	if err != nil {
		p.logger.Error("error inserting message into database", "error", err.Error(), "messageID", ID)
		return uuid.UUID{}, fmt.Errorf("error inserting message: %w", err)
	}

	p.logger.Debug("message inserted successfully", "messageID", ID)
	return ID, nil
}
