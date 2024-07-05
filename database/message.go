package database

import (
	"context"
	"fmt"
	"log"
	"time"
)

type MessageWriter interface {
	InsertMessage(ctx context.Context, msg TwitchMessage) error
}

func (p *Postgres) InsertMessage(ctx context.Context, msg TwitchMessage) error {
	query := "INSERT INTO twitch_chat (username, message, isCommand, created_at) VALUES ($1, $2, $3, $4)"
	_, err := p.connections.ExecContext(ctx, query, msg.Username, msg.Text, msg.IsCommand, msg.Time)
	if err != nil {
		log.Println("error inserting message: ", err)
		return fmt.Errorf("error inserting message: %w", err)
	}
	return nil
}

func (p *Postgres) InsertChatHistory(ctx context.Context, messages []string) error {
	query := "INSERT INTO twitch_chat (chats, created_at) VALUES ($1, $2)"
	_, err := p.connections.ExecContext(ctx, query, messages, time.Now())
	if err != nil {
		return fmt.Errorf("error inserting chat history: %w", err)
	}
	return nil
}

// TODO: I need to move this
type TwitchMessage struct {
	Username  string
	Text      string
	IsCommand bool
	Time      time.Time
}

func (p *Postgres) QueryMessageHistory(interval time.Duration) ([]TwitchMessage, error) {
	var messages []TwitchMessage
	rows, err := p.connections.Query(`select
  username,
  message,
  ts
from (
  select
    username,
    message,
    created_at as ts
  from
    twitch_chat
  where
    isCommand = false
    and created_at > current_date
  order by 
    created_at desc
  limit 10
) subquery
order by 
  ts asc;`)
	if err != nil {
		return nil, fmt.Errorf("error querying message history: %w", err)
	}
	for rows.Next() {
		var message TwitchMessage
		err := rows.Scan(&message.Username, &message.Text, &message.Time)
		if err != nil {
			return nil, fmt.Errorf("error scanning message: %w", err)
		}
		messages = append(messages, message)
	}
	return messages, nil
}
