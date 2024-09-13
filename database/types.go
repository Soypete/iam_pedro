package database

import (
	"time"

	"github.com/google/uuid"
)

// TwitchMessage is a struct that represents a message from the Twitch chat.
type TwitchMessage struct {
	Username  string      `db:"username"`
	Text      string      `db:"message"`
	IsCommand bool        `db:"isCommand"`
	Time      time.Time   `db:"created_at"`
	UUID      uuid.UUID   `db:"uuid"`
	Embedding [][]float32 `db:"embedding"`
}
