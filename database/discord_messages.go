package database

import (
	"context"
	"fmt"

	"github.com/Soypete/twitch-llm-bot/types"
)

type DiscordWriter interface {
	AskPedroWriter
	TwentyQuestionsWriter
}

// AskPedroWriter is an interface for writing AskPedro DiscordMessages to the database.
type AskPedroWriter interface {
	InsertDiscordAskPedro(ctx context.Context, message types.DiscordAskMessage) error
	GetDiscordAskPedroHistory(ctx context.Context, threadID string) ([]types.DiscordAskMessage, error)
}

// InsertDiscordAskPedro inserts a DiscordAskMessage into the database.
func (p *Postgres) InsertDiscordAskPedro(ctx context.Context, message types.DiscordAskMessage) error {
	query := `INSERT INTO discord_ask_pedro (
		thread_id, 
		message, 
		username, 
		thread_timeout,
		is_from_pedro) 
	VALUES (
		:thread_id, 
		:message, 
		:username, 
		:thread_timeout,
		:is_from_pedro);`
	_, err := p.connections.NamedExecContext(ctx, query, message)
	if err != nil {
		return fmt.Errorf("error inserting discord ask pedro message: %w", err)
	}
	return nil
}

// GetDiscordAskPedroHistory retrieves the conversation history for a thread from the database.
func (p *Postgres) GetDiscordAskPedroHistory(ctx context.Context, threadID string) ([]types.DiscordAskMessage, error) {
	var messages []types.DiscordAskMessage
	query := `SELECT 
		thread_id, 
		message, 
		username, 
		thread_timeout,
		is_from_pedro
	FROM discord_ask_pedro 
	WHERE thread_id = $1 
	ORDER BY created_at ASC;`
	
	err := p.connections.SelectContext(ctx, &messages, query, threadID)
	if err != nil {
		return nil, fmt.Errorf("error getting discord ask pedro history: %w", err)
	}
	return messages, nil
}

// TwentyQuestionsWriter is an interface for writing 20Questions DiscordMessages to the database.
type TwentyQuestionsWriter interface {
	InsertDiscordPlay20Questions(ctx context.Context, message types.Discord20QuestionsMessage) error
	CreateDiscord20Questions(ctx context.Context, message types.Discord20QuestionsGame) error
	UpdateDiscord20Questions(ctx context.Context, gameID string, questionCount int) error
	AbandonDiscord20Questions(ctx context.Context, gameID string, questionCount int) error
	EndDiscord20Questions(ctx context.Context, gameID string, questionCount int) error
}

// InsertDiscordPlay20Questions inserts a Discord20QuestionsMessage into the database.
func (p *Postgres) InsertDiscordPlay20Questions(ctx context.Context, message types.Discord20QuestionsMessage) error {
	query := `INSERT INTO discord_twenty_questions (
		game_id, 
		question, 
		response) 
	VALUES (
		:game_id, 
		:question, 
		:response);`
	_, err := p.connections.NamedExecContext(ctx, query, message)
	if err != nil {
		return fmt.Errorf("error inserting discord play 20 questions message: %w", err)
	}

	return nil
}

// CreateDiscord20Questions creates a Discord20QuestionsGame in the database.
func (p *Postgres) CreateDiscord20Questions(ctx context.Context, message types.Discord20QuestionsGame) error {
	query := `INSERT INTO discord_twenty_questions_games (
		game_id, 
		thread_id, 
		answer, 
		username, 
		thread_timeout,
		status) 
	VALUES (
		:game_id, 
		:thread_id, 
		:answer, 
		:username, 
		:thread_timeout,
		'started');`
	_, err := p.connections.NamedExecContext(ctx, query, message)
	if err != nil {
		return fmt.Errorf("error creating discord 20 questions game: %w", err)
	}
	return nil
}

// UpdateDiscord20Questions updates a Discord20QuestionsGame in the database.
func (p *Postgres) UpdateDiscord20Questions(ctx context.Context, gameID string, questionCount int) error {
	query := `UPDATE discord_twenty_questions_games
	SET question_count = $2 
	WHERE game_id = $1;`
	_, err := p.connections.ExecContext(ctx, query, gameID, questionCount)
	if err != nil {
		return fmt.Errorf("error updating discord 20 questions game: %w", err)
	}

	return nil
}

// AbandonDiscord20Questions abandons a Discord20QuestionsGame in the database.
func (p *Postgres) AbandonDiscord20Questions(ctx context.Context, gameID string, questionCount int) error {
	query := `UPDATE discord_twenty_questions_games
	SET status = 'abandoned', 
	question_count = $2 
	WHERE game_id = $1;`
	_, err := p.connections.ExecContext(ctx, query, gameID, questionCount)
	if err != nil {
		return fmt.Errorf("error abandoning discord 20 questions game: %w", err)
	}

	return nil
}

// EndDiscord20Questions ends a Discord20QuestionsGame in the database.
func (p *Postgres) EndDiscord20Questions(ctx context.Context, gameID string, questionCount int) error {
	query := `UPDATE discord_twenty_questions_games
	SET status = 'ended', 
	question_count = $2 
	WHERE game_id = $1;`
	_, err := p.connections.ExecContext(ctx, query, gameID, questionCount)
	if err != nil {
		return fmt.Errorf("error ending discord 20 questions game: %w", err)
	}

	return nil
}
