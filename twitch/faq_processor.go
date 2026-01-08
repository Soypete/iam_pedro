package twitchirc

import (
	"context"
	"time"

	"github.com/Soypete/twitch-llm-bot/faq"
	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/Soypete/twitch-llm-bot/types"
	v2 "github.com/gempir/go-twitch-irc/v2"
)

// FAQProcessor handles semantic FAQ matching for Twitch chat messages
type FAQProcessor struct {
	service    *faq.Service
	responseCh chan<- types.TwitchMessage
	logger     *logging.Logger
}

// NewFAQProcessor creates a new FAQ processor
// responseCh is the channel where FAQ responses will be sent
func NewFAQProcessor(service *faq.Service, responseCh chan<- types.TwitchMessage, logger *logging.Logger) *FAQProcessor {
	if logger == nil {
		logger = logging.Default()
	}
	return &FAQProcessor{
		service:    service,
		responseCh: responseCh,
		logger:     logger,
	}
}

// ProcessMessage checks a message against FAQ entries and sends a response if matched
// This method is designed to be called as a goroutine to avoid blocking the main chat flow
func (p *FAQProcessor) ProcessMessage(ctx context.Context, msg types.TwitchMessage) {
	if p.service == nil {
		return
	}

	p.logger.Debug("FAQ processor checking message",
		"username", msg.Username,
		"messageLength", len(msg.Text),
	)

	// Check against FAQ entries
	result, err := p.service.CheckMessage(ctx, msg.Text, msg.Username)
	if err != nil {
		p.logger.Error("FAQ processing error",
			"error", err.Error(),
			"username", msg.Username,
		)
		metrics.FAQCheckFailCount.Add(1)
		return
	}

	if result == nil {
		p.logger.Debug("no FAQ match found", "username", msg.Username)
		return
	}

	p.logger.Info("FAQ match found",
		"username", msg.Username,
		"faqID", result.FAQID,
		"question", result.Question,
		"similarity", result.SimilarityScore,
	)

	// Create response message
	response := types.TwitchMessage{
		Username: "Pedro_FAQ",
		Text:     result.GeneratedResponse,
		Time:     time.Now(),
	}

	// Send to response channel (non-blocking with select to prevent deadlock)
	select {
	case p.responseCh <- response:
		metrics.FAQResponseSentCount.Add(1)
		p.logger.Debug("FAQ response sent to channel", "faqID", result.FAQID)
	case <-ctx.Done():
		p.logger.Debug("context cancelled, FAQ response not sent")
	default:
		p.logger.Warn("FAQ response channel full, dropping response", "faqID", result.FAQID)
	}
}

// ProcessMessageFromPrivate converts a v2.PrivateMessage and processes it
// This is a convenience method for the main chat handler
func (p *FAQProcessor) ProcessMessageFromPrivate(ctx context.Context, msg v2.PrivateMessage) {
	twitchMsg := types.TwitchMessage{
		Username: msg.User.DisplayName,
		Text:     msg.Message,
		Time:     time.Now(),
	}
	p.ProcessMessage(ctx, twitchMsg)
}

// ShouldProcessMessage determines if a message should be checked against FAQs
// Returns false for bot messages, commands, and other messages that shouldn't trigger FAQs
func ShouldProcessMessage(msg v2.PrivateMessage) bool {
	// Skip bot messages
	switch msg.User.DisplayName {
	case "Nightbot", "StreamElements", "Moobot", "Pedro_el_asistente":
		return false
	}

	// Skip commands (they start with !)
	if len(msg.Message) > 0 && msg.Message[0] == '!' {
		return false
	}

	// Skip very short messages (likely not real questions)
	if len(msg.Message) < 10 {
		return false
	}

	return true
}
