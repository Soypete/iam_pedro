package keepalive

import (
	"context"
	"fmt"
)

type NoOpAlerter struct{}

func NewNoOpAlerter() *NoOpAlerter {
	return &NoOpAlerter{}
}

func (n *NoOpAlerter) SendAlert(ctx context.Context, serviceName string, message string) error {
	return nil
}

func (n *NoOpAlerter) Close() error {
	return nil
}

type MockAlerter struct {
	SentMessages []string
}

func NewMockAlerter() *MockAlerter {
	return &MockAlerter{}
}

func (m *MockAlerter) SendAlert(ctx context.Context, serviceName string, message string) error {
	m.SentMessages = append(m.SentMessages, fmt.Sprintf("[%s] %s", serviceName, message))
	return nil
}

func (m *MockAlerter) Close() error {
	return nil
}
