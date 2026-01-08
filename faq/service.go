package faq

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// Service orchestrates FAQ semantic matching and response generation
type Service struct {
	embeddingService   *EmbeddingService
	matcher            *Matcher
	llm                llms.Model
	threshold          float64
	logger             *logging.Logger
	db                 *sqlx.DB
	usePerUserCooldown bool
}

// ServiceConfig configures the FAQ service
type ServiceConfig struct {
	// LLMPath is the base URL for the LLM/embedding API
	LLMPath string

	// EmbeddingModel is the model name for generating embeddings
	EmbeddingModel string

	// ChatModel is the model name for generating responses
	ChatModel string

	// SimilarityThreshold is the minimum similarity score to trigger a response
	SimilarityThreshold float64

	// UsePerUserCooldown enables per-user cooldown tracking (in addition to global)
	UsePerUserCooldown bool

	// Logger for logging operations
	Logger *logging.Logger
}

// NewService creates a new FAQ service
func NewService(db *sqlx.DB, config ServiceConfig) (*Service, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}
	if config.LLMPath == "" {
		return nil, fmt.Errorf("LLMPath cannot be empty")
	}

	logger := config.Logger
	if logger == nil {
		logger = logging.Default()
	}

	// Create embedding service
	embeddingService, err := NewEmbeddingService(config.LLMPath, config.EmbeddingModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding service: %w", err)
	}

	// Create LLM for response generation
	llmPath := config.LLMPath
	if !strings.HasSuffix(llmPath, "/v1") {
		llmPath = llmPath + "/v1"
	}

	llm, err := openai.New(
		openai.WithBaseURL(llmPath),
		openai.WithModel(config.ChatModel),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Set default threshold if not specified
	threshold := config.SimilarityThreshold
	if threshold <= 0 {
		threshold = 0.75
	}

	return &Service{
		embeddingService:   embeddingService,
		matcher:            NewMatcher(db),
		llm:                llm,
		threshold:          threshold,
		logger:             logger,
		db:                 db,
		usePerUserCooldown: config.UsePerUserCooldown,
	}, nil
}

// MatchResult contains the result of checking a message against FAQ entries
type MatchResult struct {
	// Matched indicates if a match was found
	Matched bool

	// FAQID is the ID of the matched FAQ entry
	FAQID uuid.UUID

	// Question is the matched FAQ question
	Question string

	// CachedResponse is the raw cached response from the FAQ entry
	CachedResponse string

	// GeneratedResponse is the LLM-generated natural response
	GeneratedResponse string

	// SimilarityScore is the cosine similarity score
	SimilarityScore float64

	// Category is the FAQ category
	Category string
}

// CheckMessage checks if a message matches any FAQ entries and generates a response
// Returns nil if no match is found
func (s *Service) CheckMessage(ctx context.Context, userMessage string, userID string) (*MatchResult, error) {
	if userMessage == "" {
		return nil, nil
	}

	s.logger.Debug("checking message against FAQ entries",
		"messageLength", len(userMessage),
		"userID", userID,
	)

	// Generate embedding for the user's message
	embedding, err := s.embeddingService.Generate(ctx, userMessage)
	if err != nil {
		s.logger.Error("failed to generate embedding for message", "error", err.Error())
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Find matching FAQ entry
	var match *Match
	if s.usePerUserCooldown {
		match, err = s.matcher.FindMatchForUser(ctx, embedding, s.threshold, userID)
	} else {
		match, err = s.matcher.FindMatch(ctx, embedding, s.threshold)
	}

	if err != nil {
		s.logger.Error("failed to find FAQ match", "error", err.Error())
		return nil, fmt.Errorf("failed to find FAQ match: %w", err)
	}

	if match == nil {
		s.logger.Debug("no FAQ match found", "threshold", s.threshold)
		return nil, nil
	}

	s.logger.Info("FAQ match found",
		"faqID", match.ID,
		"question", match.Question,
		"similarity", match.SimilarityScore,
	)

	// Generate natural response using LLM
	generatedResponse, err := s.generateResponse(ctx, userMessage, match)
	if err != nil {
		s.logger.Error("failed to generate FAQ response", "error", err.Error())
		// Fall back to cached response on LLM failure
		generatedResponse = match.Response
	}

	// Record the trigger for cooldown tracking
	if s.usePerUserCooldown {
		if err := s.matcher.RecordTriggerWithUser(ctx, match.ID, userID); err != nil {
			s.logger.Error("failed to record FAQ trigger", "error", err.Error())
			// Continue anyway - the response was already generated
		}
	} else {
		if err := s.matcher.RecordTrigger(ctx, match.ID); err != nil {
			s.logger.Error("failed to record FAQ trigger", "error", err.Error())
		}
	}

	// Record the response for analytics
	if err := s.recordResponse(ctx, match.ID, userID, userMessage, match.SimilarityScore, generatedResponse); err != nil {
		s.logger.Error("failed to record FAQ response", "error", err.Error())
		// Continue anyway
	}

	result := &MatchResult{
		Matched:           true,
		FAQID:             match.ID,
		Question:          match.Question,
		CachedResponse:    match.Response,
		GeneratedResponse: generatedResponse,
		SimilarityScore:   match.SimilarityScore,
	}

	if match.Category.Valid {
		result.Category = match.Category.String
	}

	return result, nil
}

// generateResponse uses the LLM to generate a natural response based on the FAQ match
func (s *Service) generateResponse(ctx context.Context, userMessage string, match *Match) (string, error) {
	// Check for special tokens that require dynamic handling
	if strings.HasPrefix(match.Response, "FETCH_") {
		// Handle special tokens like FETCH_LATEST_VIDEO
		// For now, just use the response as-is
		// TODO: Implement dynamic fetching based on token
		s.logger.Debug("special token detected in FAQ response", "token", match.Response)
	}

	prompt := fmt.Sprintf(`A viewer asked: "%s"

This matches our FAQ about: "%s"
The information to share is: %s

Generate a brief, friendly chat response (under 400 characters) that naturally answers their question with this information. Be conversational and on-brand for a tech streamer. Do not use newlines.`,
		userMessage, match.Question, match.Response)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, "You are Pedro, a friendly chatbot assistant for SoyPeteTech's Twitch stream. You help viewers with quick, helpful responses. Keep responses under 400 characters with no newlines."),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	response, err := s.llm.GenerateContent(ctx, messages,
		llms.WithTemperature(0.7),
		llms.WithMaxTokens(100),
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate LLM response: %w", err)
	}

	if len(response.Choices) == 0 || response.Choices[0].Content == "" {
		return "", fmt.Errorf("empty response from LLM")
	}

	// Clean the response (remove newlines for Twitch compatibility)
	cleanedResponse := strings.ReplaceAll(response.Choices[0].Content, "\n", " ")
	cleanedResponse = strings.TrimSpace(cleanedResponse)

	// Truncate if too long
	if len(cleanedResponse) > 450 {
		cleanedResponse = cleanedResponse[:447] + "..."
	}

	return cleanedResponse, nil
}

// recordResponse saves the FAQ response for analytics
func (s *Service) recordResponse(ctx context.Context, faqID uuid.UUID, userID, userMessage string, similarity float64, response string) error {
	query := `
		INSERT INTO faq_responses (faq_id, user_id, user_message, similarity_score, response_sent)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := s.db.ExecContext(ctx, query, faqID, userID, userMessage, similarity, response)
	return err
}

// ProcessMessageAsync checks a message for FAQ matches in a non-blocking way
// Returns a channel that will receive the result when available
func (s *Service) ProcessMessageAsync(userMessage string, userID string) <-chan *MatchResult {
	resultCh := make(chan *MatchResult, 1)

	go func() {
		defer close(resultCh)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		result, err := s.CheckMessage(ctx, userMessage, userID)
		if err != nil {
			s.logger.Error("async FAQ check failed", "error", err.Error())
			return
		}

		if result != nil {
			resultCh <- result
		}
	}()

	return resultCh
}

// UpdateThreshold updates the similarity threshold
func (s *Service) UpdateThreshold(threshold float64) error {
	if threshold <= 0 || threshold > 1 {
		return fmt.Errorf("threshold must be between 0 and 1")
	}
	s.threshold = threshold
	return nil
}

// GetThreshold returns the current similarity threshold
func (s *Service) GetThreshold() float64 {
	return s.threshold
}

// CleanupCooldowns removes old cooldown entries to prevent table bloat
func (s *Service) CleanupCooldowns(ctx context.Context, olderThan time.Duration) (int64, error) {
	return s.matcher.CleanupOldCooldowns(ctx, olderThan)
}

// FUTURE SCALABILITY NOTES:
//
// Currently single-instance, but designed for horizontal scaling:
//
// 1. Database-based vector search (not in-memory) allows multiple instances
//    to query without synchronization issues
//
// 2. For distributed message deduplication, consider:
//    - Redis-based message ID tracking with TTL
//    - Database-level advisory locks: SELECT pg_try_advisory_lock(message_hash)
//    - Claim-based processing: UPDATE faq_entries SET processing_instance = $1
//      WHERE id = $2 AND processing_instance IS NULL
//
// 3. For high-volume scenarios:
//    - Add message queue (NATS, Redis Streams) between chat and FAQ processor
//    - Multiple FAQ processor workers can consume from queue
//    - Use database transactions for atomic cooldown updates
//
// 4. Embedding generation could be batched if volume requires
