// Package store provides per-stream-session SQLite storage for chat messages.
//
// Each live stream gets its own SQLite database in the active sessions directory.
// The schema is derived dynamically from the ontology - one table per topic class.
// This provides isolation between streams and keeps the database small.
//
// Message fields: id, stream_id, username, message, timestamp, topic, confidence
//
// Usage:
//   - Init() creates database and tables for a session
//   - WriteMessage() inserts classified message
//   - Query() searches by topic or text
package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Soypete/twitch-llm-bot/internal/mempalace/ontology"
	"github.com/Soypete/twitch-llm-bot/metrics"
	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db       *sql.DB
	streamID string
	mu       sync.Mutex
}

type Message struct {
	ID         string
	StreamID   string
	Username   string
	Message    string
	Timestamp  time.Time
	Topic      string
	Confidence float64
}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) Init(streamID string, classes []ontology.Class) error {
	s.streamID = streamID

	baseDir := os.Getenv("MEMPALACE_DATA_DIR")
	if baseDir == "" {
		baseDir = "/data/palaces/active"
	}

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create mempalace directory: %w", err)
	}

	dbPath := filepath.Join(baseDir, fmt.Sprintf("%s.sqlite", streamID))

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open sqlite: %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping sqlite: %w", err)
	}

	s.db = db

	return s.createTables(classes)
}

func (s *Store) createTables(classes []ontology.Class) error {
	schema := `
	CREATE TABLE IF NOT EXISTS messages_raw (
		id TEXT PRIMARY KEY,
		stream_id TEXT NOT NULL,
		username TEXT NOT NULL,
		message TEXT NOT NULL,
		timestamp TEXT NOT NULL,
		topic TEXT DEFAULT 'Unclassified',
		confidence REAL DEFAULT 0.0
	);
	CREATE INDEX IF NOT EXISTS idx_messages_raw_stream ON messages_raw(stream_id);
	CREATE INDEX IF NOT EXISTS idx_messages_raw_timestamp ON messages_raw(timestamp);
	`

	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create base tables: %w", err)
	}

	for _, class := range classes {
		tableName := sanitizeTableName(class.Label)
		createSQL := fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS messages_%s (
				id TEXT PRIMARY KEY,
				stream_id TEXT NOT NULL,
				username TEXT NOT NULL,
				message TEXT NOT NULL,
				timestamp TEXT NOT NULL,
				confidence REAL DEFAULT 0.0,
				FOREIGN KEY(id) REFERENCES messages_raw(id)
			);
			CREATE INDEX IF NOT EXISTS idx_messages_%s_stream ON messages_%s(stream_id);
			CREATE INDEX IF NOT EXISTS idx_messages_%s_timestamp ON messages_%s(timestamp);
		`, tableName, tableName, tableName, tableName, tableName)

		if _, err := s.db.Exec(createSQL); err != nil {
			return fmt.Errorf("failed to create topic table %s: %w", tableName, err)
		}
	}

	return nil
}

func sanitizeTableName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, "#", "_")
	return name
}

func (s *Store) WriteMessage(ctx context.Context, msg Message) error {
	start := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO messages_raw (id, stream_id, username, message, timestamp, topic, confidence) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		msg.ID, msg.StreamID, msg.Username, msg.Message, msg.Timestamp.Format(time.RFC3339), msg.Topic, msg.Confidence,
	)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if msg.Topic != "" && msg.Topic != "Unclassified" {
		tableName := sanitizeTableName(msg.Topic)
		_, err = s.db.ExecContext(ctx,
			fmt.Sprintf(`INSERT OR REPLACE INTO messages_%s (id, stream_id, username, message, timestamp, confidence) VALUES (?, ?, ?, ?, ?, ?)`, tableName),
			msg.ID, msg.StreamID, msg.Username, msg.Message, msg.Timestamp.Format(time.RFC3339), msg.Confidence,
		)
		if err != nil {
			return fmt.Errorf("failed to write to topic table: %w", err)
		}
	}

	metrics.MempalaceSQLiteWriteLatency.Observe(time.Since(start).Seconds())
	return nil
}

type QueryOpts struct {
	Topic     string
	TimeStart *time.Time
	TimeEnd   *time.Time
	QueryText string
	Username  string
	Limit     int
}

func (s *Store) Query(ctx context.Context, opts QueryOpts) ([]Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var tableName string
	if opts.Topic == "" || opts.Topic == "Unclassified" {
		tableName = "messages_raw"
	} else {
		tableName = fmt.Sprintf("messages_%s", sanitizeTableName(opts.Topic))
	}

	query := fmt.Sprintf("SELECT id, stream_id, username, message, timestamp, topic, confidence FROM %s WHERE 1=1", tableName)
	args := []interface{}{}

	if opts.TimeStart != nil {
		query += " AND timestamp >= ?"
		args = append(args, opts.TimeStart.Format(time.RFC3339))
	}
	if opts.TimeEnd != nil {
		query += " AND timestamp <= ?"
		args = append(args, opts.TimeEnd.Format(time.RFC3339))
	}
	if opts.Username != "" {
		query += " AND username = ?"
		args = append(args, opts.Username)
	}
	if opts.QueryText != "" {
		query += " AND message LIKE ?"
		args = append(args, "%"+opts.QueryText+"%")
	}

	query += " ORDER BY timestamp DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	} else {
		query += " LIMIT 50"
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer func() { rows.Close() }()

	var messages []Message
	for rows.Next() {
		var msg Message
		var timestampStr string
		var topic string
		err := rows.Scan(&msg.ID, &msg.StreamID, &msg.Username, &msg.Message, &timestampStr, &topic, &msg.Confidence)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		msg.Timestamp, _ = time.Parse(time.RFC3339, timestampStr)
		msg.Topic = topic
		messages = append(messages, msg)
	}

	return messages, nil
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Store) Path() string {
	baseDir := os.Getenv("MEMPALACE_DATA_DIR")
	if baseDir == "" {
		baseDir = "/data/palaces/active"
	}
	return filepath.Join(baseDir, fmt.Sprintf("%s.sqlite", s.streamID))
}
