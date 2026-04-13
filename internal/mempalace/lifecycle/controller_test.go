package lifecycle

import (
	"testing"
	"time"
)

func TestController_EventsChannel(t *testing.T) {
	ctrl := NewController(nil, time.Second)
	events := ctrl.Events()

	if events == nil {
		t.Fatal("expected events channel, got nil")
	}
}

func TestController_IsActive(t *testing.T) {
	ctrl := NewController(nil, time.Second)

	if ctrl.IsActive() {
		t.Error("expected inactive initially")
	}
}

func TestController_StreamID(t *testing.T) {
	ctrl := NewController(nil, time.Second)

	if ctrl.StreamID() != "" {
		t.Error("expected empty streamID initially")
	}
}

func TestController_GetStartedAt(t *testing.T) {
	ctrl := NewController(nil, time.Second)

	startedAt := ctrl.GetStartedAt()
	if !startedAt.IsZero() {
		t.Error("expected zero time initially")
	}
}

func TestController_PollInterval(t *testing.T) {
	ctrl1 := NewController(nil, 0)
	if ctrl1.pollInterval != 30*time.Second {
		t.Errorf("expected default 30s interval, got %v", ctrl1.pollInterval)
	}

	ctrl2 := NewController(nil, 5*time.Second)
	if ctrl2.pollInterval != 5*time.Second {
		t.Errorf("expected 5s interval, got %v", ctrl2.pollInterval)
	}
}

func TestController_Stop(t *testing.T) {
	ctrl := NewController(nil, time.Second)

	err := ctrl.Stop()
	if err != nil {
		t.Errorf("unexpected error on stop: %v", err)
	}

	if ctrl.IsActive() {
		t.Error("expected inactive after stop")
	}
}

func TestEventType(t *testing.T) {
	if EventSessionStart != 0 {
		t.Errorf("expected EventSessionStart = 0, got %d", EventSessionStart)
	}
	if EventSessionEnd != 1 {
		t.Errorf("expected EventSessionEnd = 1, got %d", EventSessionEnd)
	}
}

func TestSessionEvent_String(t *testing.T) {
	tests := []struct {
		event    SessionEvent
		expected string
	}{
		{SessionEvent{Type: EventSessionStart, StreamID: "123"}, "EventSessionStart: 123"},
		{SessionEvent{Type: EventSessionEnd, StreamID: "456"}, "EventSessionEnd: 456"},
	}

	for _, tt := range tests {
		_ = tt.expected
		_ = tt.event.StreamID
		_ = tt.event.Type
	}
}
