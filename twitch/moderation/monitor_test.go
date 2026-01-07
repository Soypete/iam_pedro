package moderation

import (
	"testing"

	"github.com/Soypete/twitch-llm-bot/ai"
	"github.com/Soypete/twitch-llm-bot/types"
)

func TestNeedsEvaluation(t *testing.T) {
	// Create a monitor with a default config for testing
	m := &Monitor{
		config: ai.DefaultModerationConfig(),
	}

	tests := []struct {
		name    string
		message string
		want    bool
	}{
		{
			name:    "short message",
			message: "hi",
			want:    false,
		},
		{
			name:    "normal message",
			message: "Hello everyone!",
			want:    false,
		},
		{
			name:    "emote only",
			message: "soypet2Dance",
			want:    false,
		},
		{
			name:    "http link",
			message: "check out http://example.com for info",
			want:    true,
		},
		{
			name:    "https link",
			message: "https://spam.site/free-stuff",
			want:    true,
		},
		{
			name:    "excessive caps",
			message: "THIS IS ALL CAPS AND VERY LONG MESSAGE",
			want:    true,
		},
		{
			name:    "repeated characters",
			message: "hellooooooo everyone",
			want:    true,
		},
		{
			name:    "at everyone",
			message: "hey @everyone check this out",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.needsEvaluation(tt.message); got != tt.want {
				t.Errorf("needsEvaluation(%q) = %v, want %v", tt.message, got, tt.want)
			}
		})
	}
}

func TestShouldSkipUser(t *testing.T) {
	m := &Monitor{}

	tests := []struct {
		name     string
		username string
		want     bool
	}{
		{
			name:     "Nightbot should be skipped",
			username: "Nightbot",
			want:     true,
		},
		{
			name:     "nightbot lowercase should be skipped",
			username: "nightbot",
			want:     true,
		},
		{
			name:     "StreamElements should be skipped",
			username: "StreamElements",
			want:     true,
		},
		{
			name:     "Pedro should be skipped",
			username: "Pedro_el_asistente",
			want:     true,
		},
		{
			name:     "broadcaster should be skipped",
			username: "soypetetech",
			want:     true,
		},
		{
			name:     "regular user should not be skipped",
			username: "regularuser123",
			want:     false,
		},
		{
			name:     "another regular user",
			username: "chatviewer",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.shouldSkipUser(tt.username); got != tt.want {
				t.Errorf("shouldSkipUser(%q) = %v, want %v", tt.username, got, tt.want)
			}
		})
	}
}

func TestHasRepeatedChars(t *testing.T) {
	tests := []struct {
		name string
		s    string
		n    int
		want bool
	}{
		{
			name: "no repeats",
			s:    "hello",
			n:    5,
			want: false,
		},
		{
			name: "has 5 repeated chars",
			s:    "hellooooo",
			n:    5,
			want: true,
		},
		{
			name: "exactly 5 repeated chars",
			s:    "aaaaa",
			n:    5,
			want: true,
		},
		{
			name: "4 repeated chars looking for 5",
			s:    "aaaa",
			n:    5,
			want: false,
		},
		{
			name: "short string",
			s:    "hi",
			n:    5,
			want: false,
		},
		{
			name: "repeated at start",
			s:    "aaaaahello",
			n:    5,
			want: true,
		},
		{
			name: "repeated at end",
			s:    "helloaaaaa",
			n:    5,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasRepeatedChars(tt.s, tt.n); got != tt.want {
				t.Errorf("hasRepeatedChars(%q, %d) = %v, want %v", tt.s, tt.n, got, tt.want)
			}
		})
	}
}

func TestAddRecentMessage(t *testing.T) {
	m := &Monitor{
		recentMsgs:    make([]types.TwitchMessage, 0, 5),
		maxRecentMsgs: 5,
	}

	// Add messages up to max
	for i := 0; i < 5; i++ {
		m.addRecentMessage(types.TwitchMessage{
			Username: "user",
			Text:     "message",
		})
	}

	if len(m.recentMsgs) != 5 {
		t.Errorf("expected 5 messages, got %d", len(m.recentMsgs))
	}

	// Add one more, should maintain max
	m.addRecentMessage(types.TwitchMessage{
		Username: "newuser",
		Text:     "new message",
	})

	if len(m.recentMsgs) != 5 {
		t.Errorf("expected 5 messages after overflow, got %d", len(m.recentMsgs))
	}

	// The oldest message should have been removed
	if m.recentMsgs[4].Username != "newuser" {
		t.Error("newest message should be at the end")
	}
}

func TestGetRecentMessages(t *testing.T) {
	m := &Monitor{
		recentMsgs:    make([]types.TwitchMessage, 0, 5),
		maxRecentMsgs: 5,
	}

	// Add some messages
	for i := 0; i < 3; i++ {
		m.addRecentMessage(types.TwitchMessage{
			Username: "user",
			Text:     "message",
		})
	}

	msgs := m.getRecentMessages()

	if len(msgs) != 3 {
		t.Errorf("expected 3 messages, got %d", len(msgs))
	}

	// Verify it's a copy, not the original slice
	msgs[0].Username = "modified"
	if m.recentMsgs[0].Username == "modified" {
		t.Error("getRecentMessages should return a copy")
	}
}

func TestCheckRateLimit(t *testing.T) {
	m := &Monitor{
		config: &ai.ModerationConfig{
			RateLimits: ai.RateLimits{
				ActionsPerMinute: 3,
			},
		},
	}

	// First few calls should succeed
	for i := 0; i < 3; i++ {
		if !m.checkRateLimit() {
			t.Errorf("call %d should have succeeded", i+1)
		}
	}

	// Fourth call should fail due to rate limit
	if m.checkRateLimit() {
		t.Error("fourth call should have been rate limited")
	}
}

func TestGetAvailableTools(t *testing.T) {
	tests := []struct {
		name         string
		allowedTools []string
		wantMinLen   int
	}{
		{
			name:         "empty allowed list returns all core tools",
			allowedTools: []string{},
			wantMinLen:   10, // Core tools count
		},
		{
			name:         "whitelist filters tools",
			allowedTools: []string{"no_action", "timeout_user"},
			wantMinLen:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Monitor{
				config: &ai.ModerationConfig{
					AllowedTools: tt.allowedTools,
				},
			}

			tools := m.getAvailableTools()

			if len(tools) < tt.wantMinLen {
				t.Errorf("expected at least %d tools, got %d", tt.wantMinLen, len(tools))
			}

			// no_action should always be present
			hasNoAction := false
			for _, tool := range tools {
				if tool.Function.Name == "no_action" {
					hasNoAction = true
					break
				}
			}
			if !hasNoAction {
				t.Error("no_action tool should always be present")
			}
		})
	}
}
