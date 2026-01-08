package faq

import (
	"context"
	"fmt"
	"time"

	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// SyncResult contains statistics about a sync operation
type SyncResult struct {
	EntriesProcessed int
	EntriesCreated   int
	EntriesUpdated   int
	EntriesDeleted   int
	Errors           []error
	Duration         time.Duration
}

// Syncer handles syncing FAQ config to the database
type Syncer struct {
	db               *sqlx.DB
	embeddingService *EmbeddingService
	logger           *logging.Logger
}

// NewSyncer creates a new FAQ syncer
func NewSyncer(db *sqlx.DB, embeddingService *EmbeddingService, logger *logging.Logger) *Syncer {
	if logger == nil {
		logger = logging.Default()
	}
	return &Syncer{
		db:               db,
		embeddingService: embeddingService,
		logger:           logger,
	}
}

// SyncFromConfig synchronizes FAQ entries from a config file to the database
// This performs a full sync: deletes entries not in config, updates existing, creates new
func (s *Syncer) SyncFromConfig(ctx context.Context, config *Config) (*SyncResult, error) {
	start := time.Now()
	result := &SyncResult{}

	s.logger.Info("starting FAQ sync",
		"entryCount", len(config.Entries),
		"embeddingModel", config.EmbeddingModel,
	)

	// Start a transaction for atomic sync
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get existing entries (by question text for matching)
	existingEntries, err := s.getExistingEntries(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing entries: %w", err)
	}

	// Track which entries from config we've processed
	processedQuestions := make(map[string]bool)

	// Process each config entry
	for _, entryConfig := range config.Entries {
		result.EntriesProcessed++
		processedQuestions[entryConfig.Question] = true

		// Generate embedding for this entry
		embedding, err := s.embeddingService.Generate(ctx, entryConfig.Question)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to generate embedding for '%s': %w", entryConfig.Question, err))
			s.logger.Error("failed to generate embedding", "question", entryConfig.Question, "error", err.Error())
			continue
		}

		// Check if entry already exists
		existingID, exists := existingEntries[entryConfig.Question]
		if exists {
			// Update existing entry
			if err := s.updateEntry(ctx, tx, existingID, entryConfig, embedding); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to update '%s': %w", entryConfig.Question, err))
				s.logger.Error("failed to update entry", "question", entryConfig.Question, "error", err.Error())
				continue
			}
			result.EntriesUpdated++
			s.logger.Debug("updated FAQ entry", "question", entryConfig.Question)
		} else {
			// Create new entry
			if err := s.createEntry(ctx, tx, entryConfig, embedding); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to create '%s': %w", entryConfig.Question, err))
				s.logger.Error("failed to create entry", "question", entryConfig.Question, "error", err.Error())
				continue
			}
			result.EntriesCreated++
			s.logger.Debug("created FAQ entry", "question", entryConfig.Question)
		}
	}

	// Delete entries that are no longer in the config
	for question, id := range existingEntries {
		if !processedQuestions[question] {
			if err := s.deleteEntry(ctx, tx, id); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to delete '%s': %w", question, err))
				s.logger.Error("failed to delete entry", "question", question, "error", err.Error())
				continue
			}
			result.EntriesDeleted++
			s.logger.Debug("deleted FAQ entry", "question", question)
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	result.Duration = time.Since(start)

	s.logger.Info("FAQ sync completed",
		"processed", result.EntriesProcessed,
		"created", result.EntriesCreated,
		"updated", result.EntriesUpdated,
		"deleted", result.EntriesDeleted,
		"errors", len(result.Errors),
		"duration", result.Duration,
	)

	return result, nil
}

// getExistingEntries returns a map of question -> id for all existing FAQ entries
func (s *Syncer) getExistingEntries(ctx context.Context, tx *sqlx.Tx) (map[string]uuid.UUID, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id, question FROM faq_entries`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	entries := make(map[string]uuid.UUID)
	for rows.Next() {
		var id uuid.UUID
		var question string
		if err := rows.Scan(&id, &question); err != nil {
			return nil, err
		}
		entries[question] = id
	}

	return entries, rows.Err()
}

// createEntry creates a new FAQ entry
func (s *Syncer) createEntry(ctx context.Context, tx *sqlx.Tx, config EntryConfig, embedding []float32) error {
	query := `
		INSERT INTO faq_entries (question, response, category, embedding, is_active, cooldown_seconds)
		VALUES ($1, $2, $3, $4::vector, $5, $6)
	`

	var category *string
	if config.Category != "" {
		category = &config.Category
	}

	_, err := tx.ExecContext(ctx, query,
		config.Question,
		config.Response,
		category,
		VectorToString(embedding),
		config.IsEntryActive(),
		config.GetActiveCooldown(300),
	)
	return err
}

// updateEntry updates an existing FAQ entry
func (s *Syncer) updateEntry(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, config EntryConfig, embedding []float32) error {
	query := `
		UPDATE faq_entries
		SET response = $2,
		    category = $3,
		    embedding = $4::vector,
		    is_active = $5,
		    cooldown_seconds = $6,
		    updated_at = NOW()
		WHERE id = $1
	`

	var category *string
	if config.Category != "" {
		category = &config.Category
	}

	_, err := tx.ExecContext(ctx, query,
		id,
		config.Response,
		category,
		VectorToString(embedding),
		config.IsEntryActive(),
		config.GetActiveCooldown(300),
	)
	return err
}

// deleteEntry deletes an FAQ entry (also cascades to cooldowns and responses)
func (s *Syncer) deleteEntry(ctx context.Context, tx *sqlx.Tx, id uuid.UUID) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM faq_entries WHERE id = $1`, id)
	return err
}

// RegenerateAllEmbeddings regenerates embeddings for all FAQ entries
// Useful when changing embedding models
func (s *Syncer) RegenerateAllEmbeddings(ctx context.Context) (*SyncResult, error) {
	start := time.Now()
	result := &SyncResult{}

	s.logger.Info("regenerating all FAQ embeddings")

	// Get all entries
	rows, err := s.db.QueryContext(ctx, `SELECT id, question FROM faq_entries`)
	if err != nil {
		return nil, fmt.Errorf("failed to query entries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	type entry struct {
		ID       uuid.UUID
		Question string
	}

	var entries []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.ID, &e.Question); err != nil {
			return nil, fmt.Errorf("failed to scan entry: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entries: %w", err)
	}

	// Regenerate embedding for each entry
	for _, e := range entries {
		result.EntriesProcessed++

		embedding, err := s.embeddingService.Generate(ctx, e.Question)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to generate embedding for '%s': %w", e.Question, err))
			s.logger.Error("failed to generate embedding", "question", e.Question, "error", err.Error())
			continue
		}

		_, err = s.db.ExecContext(ctx, `
			UPDATE faq_entries SET embedding = $2::vector, updated_at = NOW() WHERE id = $1
		`, e.ID, VectorToString(embedding))
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to update embedding for '%s': %w", e.Question, err))
			s.logger.Error("failed to update embedding", "question", e.Question, "error", err.Error())
			continue
		}

		result.EntriesUpdated++
		s.logger.Debug("regenerated embedding", "question", e.Question)
	}

	result.Duration = time.Since(start)

	s.logger.Info("embedding regeneration completed",
		"processed", result.EntriesProcessed,
		"updated", result.EntriesUpdated,
		"errors", len(result.Errors),
		"duration", result.Duration,
	)

	return result, nil
}

// ListEntries returns all FAQ entries from the database
func (s *Syncer) ListEntries(ctx context.Context) ([]FAQEntry, error) {
	query := `
		SELECT id, question, response, category, is_active, cooldown_seconds,
		       last_triggered_at, created_at, updated_at
		FROM faq_entries
		ORDER BY category NULLS LAST, question
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query FAQ entries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var entries []FAQEntry
	for rows.Next() {
		var e FAQEntry
		if err := rows.Scan(&e.ID, &e.Question, &e.Response, &e.Category, &e.IsActive, &e.CooldownSeconds,
			&e.LastTriggeredAt, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan FAQ entry: %w", err)
		}
		entries = append(entries, e)
	}

	return entries, rows.Err()
}

// FAQEntry represents an FAQ entry from the database
type FAQEntry struct {
	ID              uuid.UUID
	Question        string
	Response        string
	Category        *string
	IsActive        bool
	CooldownSeconds int
	LastTriggeredAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// TestMatch tests semantic matching for a message without triggering cooldowns
func (s *Syncer) TestMatch(ctx context.Context, message string, threshold float64) (*Match, error) {
	embedding, err := s.embeddingService.Generate(ctx, message)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	matcher := NewMatcher(s.db)

	// Use a modified query that ignores cooldowns for testing
	vectorStr := VectorToString(embedding)
	query := `
		SELECT
			id,
			question,
			response,
			category,
			cooldown_seconds,
			1 - (embedding <=> $1::vector) AS similarity
		FROM faq_entries
		WHERE is_active = true
		  AND embedding IS NOT NULL
		  AND 1 - (embedding <=> $1::vector) >= $2
		ORDER BY embedding <=> $1::vector
		LIMIT 1
	`

	var match Match
	err = matcher.db.QueryRowContext(ctx, query, vectorStr, threshold).Scan(
		&match.ID,
		&match.Question,
		&match.Response,
		&match.Category,
		&match.CooldownSeconds,
		&match.SimilarityScore,
	)

	if err != nil {
		return nil, nil // No match
	}

	return &match, nil
}
