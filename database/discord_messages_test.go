package database

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Soypete/twitch-llm-bot/types"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDiscordAskPedroHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	postgres := &Postgres{connections: sqlxDB}

	threadID := "test-thread-123"
	
	// Expected messages
	expectedMessages := []types.DiscordAskMessage{
		{
			ThreadID:      threadID,
			Message:       "Hello Pedro",
			Username:      "testuser",
			ThreadTimeout: 30,
			IsFromPedro:   false,
		},
		{
			ThreadID:      threadID,
			Message:       "Hello! How can I help you?",
			Username:      "Pedro",
			ThreadTimeout: 30,
			IsFromPedro:   true,
		},
	}

	rows := sqlmock.NewRows([]string{"thread_id", "message", "username", "thread_timeout", "is_from_pedro"})
	for _, msg := range expectedMessages {
		rows.AddRow(msg.ThreadID, msg.Message, msg.Username, msg.ThreadTimeout, msg.IsFromPedro)
	}

	mock.ExpectQuery("SELECT thread_id, message, username, thread_timeout, is_from_pedro FROM discord_ask_pedro WHERE thread_id = \\$1 ORDER BY created_at ASC").
		WithArgs(threadID).
		WillReturnRows(rows)

	// Execute
	messages, err := postgres.GetDiscordAskPedroHistory(context.Background(), threadID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedMessages, messages)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertDiscordAskPedro(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	postgres := &Postgres{connections: sqlxDB}

	message := types.DiscordAskMessage{
		ThreadID:      "test-thread-456",
		Message:       "Test message",
		Username:      "testuser",
		ThreadTimeout: 30,
		IsFromPedro:   false,
	}

	mock.ExpectExec("INSERT INTO discord_ask_pedro").
		WithArgs(message.ThreadID, message.Message, message.Username, message.ThreadTimeout, message.IsFromPedro).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Execute
	err = postgres.InsertDiscordAskPedro(context.Background(), message)

	// Assert
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateDiscord20Questions(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	postgres := &Postgres{connections: sqlxDB}

	game := types.Discord20QuestionsGame{
		GameID:        uuid.New(),
		ThreadID:      "test-thread-789",
		Answer:        "cat",
		Username:      "testuser",
		ThreadTimeout: 30,
		Status:        "started",
	}

	mock.ExpectExec("INSERT INTO discord_twenty_questions_games").
		WithArgs(game.GameID, game.ThreadID, game.Answer, game.Username, game.ThreadTimeout).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Execute
	err = postgres.CreateDiscord20Questions(context.Background(), game)

	// Assert
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}