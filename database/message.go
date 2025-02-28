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
	p.logger.Debug("generating UUID for message", "user", msg.Username)
	ID, err := uuid.NewUUID()
	if err != nil {
		p.logger.Error("error generating UUID", "error", err.Error())
		return uuid.UUID{}, fmt.Errorf("error generating UUID: %w", err)
	}
	msg.UUID = ID
	
	query := "INSERT INTO twitch_chat (username, message, isCommand, created_at, uuid) VALUES (:username, :message, :isCommand, :created_at, :uuid)"
	p.logger.Debug("inserting message into database", "messageID", ID, "user", msg.Username)
	
	_, err = p.connections.NamedExecContext(ctx, query, msg)
	if err != nil {
		p.logger.Error("error inserting message into database", "error", err.Error(), "messageID", ID)
		return uuid.UUID{}, fmt.Errorf("error inserting message: %w", err)
	}
	
	p.logger.Debug("message inserted successfully", "messageID", ID)
	return ID, nil
}

// TODO: I need to move this to a different file
// and it should have a more generic name
// like message. and then I can use the same struct
// for the discord messages
type TwitchMessage struct {
	Username  string    `db:"username"`
	Text      string    `db:"message"`
	IsCommand bool      `db:"isCommand"`
	Time      time.Time `db:"created_at"`
	UUID      uuid.UUID `db:"uuid"`
}
