package faq

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Match represents a matched FAQ entry with its similarity score
type Match struct {
	ID              uuid.UUID
	Question        string
	Response        string
	Category        sql.NullString
	SimilarityScore float64
	CooldownSeconds int
}

// Matcher handles semantic matching of messages against FAQ entries
type Matcher struct {
	db *sqlx.DB
}

// NewMatcher creates a new FAQ matcher
func NewMatcher(db *sqlx.DB) *Matcher {
	return &Matcher{db: db}
}

// FindMatch searches for a matching FAQ entry using cosine similarity
// Returns the best match if similarity >= threshold and cooldown has passed
// Returns nil if no match is found
func (m *Matcher) FindMatch(ctx context.Context, embedding []float32, threshold float64) (*Match, error) {
	if len(embedding) == 0 {
		return nil, fmt.Errorf("embedding cannot be empty")
	}

	// Convert embedding to PostgreSQL vector string format
	vectorStr := VectorToString(embedding)

	// Query using pgvector cosine similarity
	// The <=> operator returns cosine distance, so we convert to similarity with 1 - distance
	// We also check the cooldown in the query for efficiency
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
		  AND (last_triggered_at IS NULL OR last_triggered_at < NOW() - INTERVAL '1 second' * cooldown_seconds)
		ORDER BY embedding <=> $1::vector
		LIMIT 1
	`

	var match Match
	err := m.db.QueryRowContext(ctx, query, vectorStr, threshold).Scan(
		&match.ID,
		&match.Question,
		&match.Response,
		&match.Category,
		&match.CooldownSeconds,
		&match.SimilarityScore,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No match found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query FAQ match: %w", err)
	}

	return &match, nil
}

// FindMatchForUser searches for a matching FAQ entry, also checking per-user cooldowns
func (m *Matcher) FindMatchForUser(ctx context.Context, embedding []float32, threshold float64, userID string) (*Match, error) {
	if len(embedding) == 0 {
		return nil, fmt.Errorf("embedding cannot be empty")
	}
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}

	vectorStr := VectorToString(embedding)

	// Query that also checks per-user cooldowns via LEFT JOIN
	query := `
		SELECT
			f.id,
			f.question,
			f.response,
			f.category,
			f.cooldown_seconds,
			1 - (f.embedding <=> $1::vector) AS similarity
		FROM faq_entries f
		LEFT JOIN faq_user_cooldowns uc ON f.id = uc.faq_id AND uc.user_id = $3
		WHERE f.is_active = true
		  AND f.embedding IS NOT NULL
		  AND 1 - (f.embedding <=> $1::vector) >= $2
		  AND (f.last_triggered_at IS NULL OR f.last_triggered_at < NOW() - INTERVAL '1 second' * f.cooldown_seconds)
		  AND (uc.triggered_at IS NULL OR uc.triggered_at < NOW() - INTERVAL '1 second' * f.cooldown_seconds)
		ORDER BY f.embedding <=> $1::vector
		LIMIT 1
	`

	var match Match
	err := m.db.QueryRowContext(ctx, query, vectorStr, threshold, userID).Scan(
		&match.ID,
		&match.Question,
		&match.Response,
		&match.Category,
		&match.CooldownSeconds,
		&match.SimilarityScore,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query FAQ match for user: %w", err)
	}

	return &match, nil
}

// RecordTrigger updates the last_triggered_at timestamp for global cooldown
func (m *Matcher) RecordTrigger(ctx context.Context, faqID uuid.UUID) error {
	query := `UPDATE faq_entries SET last_triggered_at = NOW() WHERE id = $1`
	_, err := m.db.ExecContext(ctx, query, faqID)
	if err != nil {
		return fmt.Errorf("failed to record FAQ trigger: %w", err)
	}
	return nil
}

// RecordUserTrigger records a per-user cooldown using upsert
func (m *Matcher) RecordUserTrigger(ctx context.Context, faqID uuid.UUID, userID string) error {
	query := `
		INSERT INTO faq_user_cooldowns (faq_id, user_id, triggered_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (faq_id, user_id)
		DO UPDATE SET triggered_at = NOW()
	`
	_, err := m.db.ExecContext(ctx, query, faqID, userID)
	if err != nil {
		return fmt.Errorf("failed to record user FAQ trigger: %w", err)
	}
	return nil
}

// RecordTriggerWithUser records both global and per-user triggers
func (m *Matcher) RecordTriggerWithUser(ctx context.Context, faqID uuid.UUID, userID string) error {
	// Use a transaction to ensure both updates happen atomically
	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Update global trigger time
	_, err = tx.ExecContext(ctx, `UPDATE faq_entries SET last_triggered_at = NOW() WHERE id = $1`, faqID)
	if err != nil {
		return fmt.Errorf("failed to update global trigger: %w", err)
	}

	// Upsert user trigger
	_, err = tx.ExecContext(ctx, `
		INSERT INTO faq_user_cooldowns (faq_id, user_id, triggered_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (faq_id, user_id)
		DO UPDATE SET triggered_at = NOW()
	`, faqID, userID)
	if err != nil {
		return fmt.Errorf("failed to update user trigger: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit trigger transaction: %w", err)
	}

	return nil
}

// CleanupOldCooldowns removes user cooldown entries older than the specified duration
// This should be called periodically to prevent the table from growing indefinitely
func (m *Matcher) CleanupOldCooldowns(ctx context.Context, olderThan time.Duration) (int64, error) {
	result, err := m.db.ExecContext(ctx, `
		DELETE FROM faq_user_cooldowns
		WHERE triggered_at < NOW() - $1::interval
	`, olderThan.String())
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old cooldowns: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}
