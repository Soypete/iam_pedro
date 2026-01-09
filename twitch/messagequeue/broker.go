// Package messagequeue provides a message broker for distributing Twitch chat messages to multiple consumers
package messagequeue

import (
	"context"
	"sync"

	"github.com/Soypete/twitch-llm-bot/logging"
	v2 "github.com/gempir/go-twitch-irc/v2"
)

// Consumer is an interface for consuming messages from the queue
type Consumer interface {
	ProcessMessage(ctx context.Context, msg v2.PrivateMessage)
	Name() string
}

// Broker distributes messages to multiple consumers
type Broker struct {
	consumers []Consumer
	msgQueue  chan v2.PrivateMessage
	logger    *logging.Logger
	mu        sync.RWMutex
}

// NewBroker creates a new message broker
func NewBroker(queueSize int, logger *logging.Logger) *Broker {
	if logger == nil {
		logger = logging.Default()
	}
	if queueSize <= 0 {
		queueSize = 1000
	}

	return &Broker{
		consumers: make([]Consumer, 0),
		msgQueue:  make(chan v2.PrivateMessage, queueSize),
		logger:    logger,
	}
}

// Subscribe adds a consumer to receive messages
func (b *Broker) Subscribe(consumer Consumer) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.consumers = append(b.consumers, consumer)
	b.logger.Info("consumer subscribed to message broker", "consumer", consumer.Name())
}

// Publish sends a message to the queue (non-blocking)
func (b *Broker) Publish(msg v2.PrivateMessage) bool {
	select {
	case b.msgQueue <- msg:
		return true
	default:
		b.logger.Warn("message queue full, dropping message")
		return false
	}
}

// Start begins processing messages and distributing to consumers
func (b *Broker) Start(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		b.logger.Info("message broker started", "consumers", len(b.consumers))

		for {
			select {
			case <-ctx.Done():
				b.logger.Info("message broker shutting down")
				return
			case msg := <-b.msgQueue:
				b.fanout(ctx, msg)
			}
		}
	}()
}

// fanout distributes a message to all consumers in parallel
func (b *Broker) fanout(ctx context.Context, msg v2.PrivateMessage) {
	b.mu.RLock()
	consumers := b.consumers
	b.mu.RUnlock()

	var wg sync.WaitGroup
	for _, consumer := range consumers {
		wg.Add(1)
		go func(c Consumer) {
			defer wg.Done()
			c.ProcessMessage(ctx, msg)
		}(consumer)
	}
	wg.Wait()
}

// GetQueueLength returns the current queue depth
func (b *Broker) GetQueueLength() int {
	return len(b.msgQueue)
}
