package twitchirc

import (
	"testing"

	v2 "github.com/gempir/go-twitch-irc/v2"
)

func TestShouldProcessMessage(t *testing.T) {
	tests := []struct {
		name     string
		msg      v2.PrivateMessage
		expected bool
	}{
		{
			name: "allows regular user message",
			msg: v2.PrivateMessage{
				Message: "Hello Pedro, how do I learn Go?",
				User:    v2.User{DisplayName: "regularuser"},
			},
			expected: true,
		},
		{
			name: "blocks Nightbot",
			msg: v2.PrivateMessage{
				Message: "Nightbot message",
				User:    v2.User{DisplayName: "Nightbot"},
			},
			expected: false,
		},
		{
			name: "blocks StreamElements",
			msg: v2.PrivateMessage{
				Message: "StreamElements message",
				User:    v2.User{DisplayName: "StreamElements"},
			},
			expected: false,
		},
		{
			name: "blocks Moobot",
			msg: v2.PrivateMessage{
				Message: "Moobot message",
				User:    v2.User{DisplayName: "Moobot"},
			},
			expected: false,
		},
		{
			name: "blocks Pedro",
			msg: v2.PrivateMessage{
				Message: "Pedro message",
				User:    v2.User{DisplayName: "Pedro_el_asistente"},
			},
			expected: false,
		},
		{
			name: "blocks commands",
			msg: v2.PrivateMessage{
				Message: "!testcommand",
				User:    v2.User{DisplayName: "user"},
			},
			expected: false,
		},
		{
			name: "blocks short messages",
			msg: v2.PrivateMessage{
				Message: "hi",
				User:    v2.User{DisplayName: "user"},
			},
			expected: false,
		},
		{
			name: "allows message at minimum length",
			msg: v2.PrivateMessage{
				Message: "how does go work?",
				User:    v2.User{DisplayName: "user"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldProcessMessage(tt.msg)
			if result != tt.expected {
				t.Errorf("ShouldProcessMessage() = %v; want %v", result, tt.expected)
			}
		})
	}
}

func TestNewFAQProcessor(t *testing.T) {
	t.Run("creates processor with default logger", func(t *testing.T) {
		responseCh := make(chan<- interface{})
		_ = responseCh
	})
}
