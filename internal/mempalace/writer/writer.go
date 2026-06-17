// Package writer provides a parallel IRC consumer that classifies and stores
// chat messages in Mem Palace.
//
// It runs as a third consumer alongside the existing responder and moderator.
// Every incoming chat message is:
//  1. Classified against the ontology using the classifier
//  2. Written to the session's SQLite store
//
// This is designed to be non-blocking - classification and storage happen
// asynchronously and don't affect the main chat response latency.
package writer

import (
	"context"
	"sync"
	"time"

	"github.com/Soypete/twitch-llm-bot/internal/mempalace/classifier"
	"github.com/Soypete/twitch-llm-bot/internal/mempalace/lifecycle"
	"github.com/Soypete/twitch-llm-bot/internal/mempalace/store"
	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/Soypete/twitch-llm-bot/types"
	v2 "github.com/gempir/go-twitch-irc/v2"
	"github.com/google/uuid"
)

type Writer struct {
	classifier *classifier.Classifier
	lifecycle  *lifecycle.Controller
	store      *store.Store
	logger     *logging.Logger
	messageCh  chan v2.PrivateMessage
	eventsCh   <-chan lifecycle.SessionEvent
	startedAt  time.Time
}

func NewWriter(
	classifier *classifier.Classifier,
	lifecycle *lifecycle.Controller,
	logger *logging.Logger,
) *Writer {
	if logger == nil {
		logger = logging.Default()
	}

	return &Writer{
		classifier: classifier,
		lifecycle:  lifecycle,
		logger:     logger,
		messageCh:  make(chan v2.PrivateMessage, 100),
		eventsCh:   lifecycle.Events(),
	}
}

func (w *Writer) SetStore(s *store.Store) {
	w.store = s
}

func (w *Writer) MessageChannel() chan<- v2.PrivateMessage {
	return w.messageCh
}

func (w *Writer) Start(ctx context.Context, wg *sync.WaitGroup) error {
	wg.Add(1)
	go func() {
		defer wg.Done()
		w.logger.Info("mempalace writer started")

		for {
			select {
			case <-ctx.Done():
				w.logger.Info("mempalace writer shutting down")
				return
			case event := <-w.eventsCh:
				w.handleSessionEvent(ctx, event)
			case msg := <-w.messageCh:
				w.processMessage(ctx, msg)
			}
		}
	}()

	return nil
}

func (w *Writer) handleSessionEvent(ctx context.Context, event lifecycle.SessionEvent) {
	switch event.Type {
	case lifecycle.EventSessionStart:
		w.startedAt = event.StartedAt
		w.logger.Info("session started", "streamID", event.StreamID)
	case lifecycle.EventSessionEnd:
		w.logger.Info("session ended", "streamID", event.StreamID)
	}
}

func (w *Writer) processMessage(ctx context.Context, msg v2.PrivateMessage) {
	if !w.lifecycle.IsActive() {
		return
	}

	if w.store == nil {
		return
	}

	topic, err := w.classifier.Classify(ctx, msg.Message)
	if err != nil {
		w.logger.Debug("failed to classify message", "error", err)
		topic = "Unclassified"
	}

	twitchMsg := types.TwitchMessage{
		Username: msg.User.DisplayName,
		Text:     msg.Message,
		Time:     time.Now(),
		UUID:     uuid.New(),
	}

	message := store.Message{
		ID:         twitchMsg.UUID.String(),
		StreamID:   w.lifecycle.StreamID(),
		Username:   twitchMsg.Username,
		Message:    twitchMsg.Text,
		Timestamp:  twitchMsg.Time,
		Topic:      topic,
		Confidence: 0.8,
	}

	if err := w.store.WriteMessage(ctx, message); err != nil {
		w.logger.Debug("failed to write message", "error", err)
	}
}

func (w *Writer) Stop() error {
	if w.store != nil {
		return w.store.Close()
	}
	return nil
}
