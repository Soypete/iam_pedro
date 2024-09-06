package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type MessageWriter interface {
	InsertMessage(ctx context.Context, msg TwitchMessage) (uuid.UUID, error)
}

// InsertMessage inserts a message into the database and returns the message ID if successful.
func (p *Postgres) InsertMessage(ctx context.Context, msg TwitchMessage) (uuid.UUID, error) {
	ID, err := uuid.NewUUID()
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("error generating UUID: %w", err)
	}
	msg.UUID = ID
	query := "INSERT INTO twitch_chat (username, message, isCommand, created_at, uuid) VALUES (:username, :message, :isCommand, :created_at, :uuid)"
	_, err = p.connections.NamedExecContext(ctx, query, msg)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("error inserting message: %w", err)
	}
	return ID, nil
}

// TODO: I need to move this
type TwitchMessage struct {
	Username  string    `db:"username"`
	Text      string    `db:"message"`
	IsCommand bool      `db:"isCommand"`
	Time      time.Time `db:"created_at"`
	UUID      uuid.UUID `db:"uuid"`
}
