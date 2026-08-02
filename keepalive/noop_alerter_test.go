package keepalive

import (
	"context"
	"testing"
)

func TestNoOpAlerter(t *testing.T) {
	t.Run("SendAlert returns nil", func(t *testing.T) {
		alerter := NewNoOpAlerter()
		err := alerter.SendAlert(context.Background(), "testService", "test message")
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})

	t.Run("Close returns nil", func(t *testing.T) {
		alerter := NewNoOpAlerter()
		err := alerter.Close()
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})
}

func TestMockAlerter(t *testing.T) {
	t.Run("SendAlert tracks messages", func(t *testing.T) {
		alerter := NewMockAlerter()
		err := alerter.SendAlert(context.Background(), "testService", "test message")
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}

		if len(alerter.SentMessages) != 1 {
			t.Errorf("expected 1 message, got %d", len(alerter.SentMessages))
		}

		expected := "[testService] test message"
		if alerter.SentMessages[0] != expected {
			t.Errorf("expected %s, got %s", expected, alerter.SentMessages[0])
		}
	})

	t.Run("SendAlert tracks multiple messages", func(t *testing.T) {
		alerter := NewMockAlerter()
		alerter.SendAlert(context.Background(), "service1", "message1")
		alerter.SendAlert(context.Background(), "service2", "message2")

		if len(alerter.SentMessages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(alerter.SentMessages))
		}
	})

	t.Run("Close returns nil", func(t *testing.T) {
		alerter := NewMockAlerter()
		err := alerter.Close()
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})
}
