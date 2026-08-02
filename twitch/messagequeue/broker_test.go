package messagequeue

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Soypete/twitch-llm-bot/logging"
	v2 "github.com/gempir/go-twitch-irc/v2"
)

type mockConsumer struct {
	name      string
	messages  []v2.PrivateMessage
	processed int
}

func (m *mockConsumer) ProcessMessage(ctx context.Context, msg v2.PrivateMessage) {
	m.messages = append(m.messages, msg)
	m.processed++
}

func (m *mockConsumer) Name() string {
	return m.name
}

func TestNewBroker(t *testing.T) {
	t.Run("creates broker with default queue size", func(t *testing.T) {
		broker := NewBroker(0, nil)
		if broker == nil {
			t.Error("expected non-nil broker")
		}
		if len(broker.msgQueue) != 0 {
			t.Errorf("expected empty queue, got capacity %d", cap(broker.msgQueue))
		}
	})

	t.Run("creates broker with custom queue size", func(t *testing.T) {
		broker := NewBroker(500, nil)
		if cap(broker.msgQueue) != 500 {
			t.Errorf("expected queue size 500, got %d", cap(broker.msgQueue))
		}
	})

	t.Run("uses default logger if nil", func(t *testing.T) {
		broker := NewBroker(100, nil)
		if broker.logger == nil {
			t.Error("expected non-nil logger")
		}
	})

	t.Run("uses custom logger if provided", func(t *testing.T) {
		logger := logging.Default()
		broker := NewBroker(100, logger)
		if broker.logger != logger {
			t.Error("expected custom logger to be used")
		}
	})
}

func TestBrokerSubscribe(t *testing.T) {
	t.Run("subscribes a consumer", func(t *testing.T) {
		broker := NewBroker(100, nil)
		consumer := &mockConsumer{name: "testconsumer"}

		broker.Subscribe(consumer)

		broker.mu.RLock()
		consumerCount := len(broker.consumers)
		broker.mu.RUnlock()

		if consumerCount != 1 {
			t.Errorf("expected 1 consumer, got %d", consumerCount)
		}
	})

	t.Run("subscribes multiple consumers", func(t *testing.T) {
		broker := NewBroker(100, nil)
		consumer1 := &mockConsumer{name: "consumer1"}
		consumer2 := &mockConsumer{name: "consumer2"}

		broker.Subscribe(consumer1)
		broker.Subscribe(consumer2)

		broker.mu.RLock()
		consumerCount := len(broker.consumers)
		broker.mu.RUnlock()

		if consumerCount != 2 {
			t.Errorf("expected 2 consumers, got %d", consumerCount)
		}
	})
}

func TestBrokerPublish(t *testing.T) {
	t.Run("publishes message to queue", func(t *testing.T) {
		broker := NewBroker(100, nil)
		msg := v2.PrivateMessage{
			Message: "test message",
			User:    v2.User{DisplayName: "testuser"},
		}

		result := broker.Publish(msg)
		if !result {
			t.Error("expected publish to succeed")
		}
	})

	t.Run("drops message when queue is full", func(t *testing.T) {
		broker := NewBroker(1, nil)
		msg := v2.PrivateMessage{Message: "test"}

		broker.Publish(msg)

		result := broker.Publish(msg)
		if result {
			t.Error("expected publish to fail when queue is full")
		}
	})
}

func TestBrokerStart(t *testing.T) {
	t.Run("starts and distributes messages", func(t *testing.T) {
		broker := NewBroker(100, nil)
		consumer := &mockConsumer{name: "testconsumer"}
		broker.Subscribe(consumer)

		ctx, cancel := context.WithCancel(context.Background())
		var wg sync.WaitGroup
		broker.Start(ctx, &wg)

		msg := v2.PrivateMessage{
			Message: "test message",
			User:    v2.User{DisplayName: "testuser"},
		}
		broker.Publish(msg)

		time.Sleep(50 * time.Millisecond)

		if consumer.processed != 1 {
			t.Errorf("expected 1 processed message, got %d", consumer.processed)
		}

		cancel()
		wg.Wait()
	})

	t.Run("handles multiple consumers", func(t *testing.T) {
		broker := NewBroker(100, nil)
		consumer1 := &mockConsumer{name: "consumer1"}
		consumer2 := &mockConsumer{name: "consumer2"}
		broker.Subscribe(consumer1)
		broker.Subscribe(consumer2)

		ctx, cancel := context.WithCancel(context.Background())
		var wg sync.WaitGroup
		broker.Start(ctx, &wg)

		msg := v2.PrivateMessage{Message: "test"}
		broker.Publish(msg)

		time.Sleep(50 * time.Millisecond)

		if consumer1.processed != 1 {
			t.Errorf("expected consumer1 to process 1 message, got %d", consumer1.processed)
		}
		if consumer2.processed != 1 {
			t.Errorf("expected consumer2 to process 1 message, got %d", consumer2.processed)
		}

		cancel()
		wg.Wait()
	})
}

func TestBrokerGetQueueLength(t *testing.T) {
	t.Run("returns queue length", func(t *testing.T) {
		broker := NewBroker(100, nil)
		broker.Publish(v2.PrivateMessage{Message: "test1"})
		broker.Publish(v2.PrivateMessage{Message: "test2"})

		length := broker.GetQueueLength()
		if length != 2 {
			t.Errorf("expected queue length 2, got %d", length)
		}
	})
}

func TestConsumerInterface(t *testing.T) {
	t.Run("mockConsumer implements Consumer", func(t *testing.T) {
		var _ Consumer = &mockConsumer{}
	})
}
