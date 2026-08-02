package twitchirc

import (
	"testing"
)

func TestSanitizeForEntity(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase conversion",
			input:    "Hello World",
			expected: "hello world",
		},
		{
			name:     "removes special characters",
			input:    "Hello@World#123!",
			expected: "helloworld123",
		},
		{
			name:     "limits to 6 words",
			input:    "one two three four five six seven eight",
			expected: "one two three four five six",
		},
		{
			name:     "preserves numbers",
			input:    "test123",
			expected: "test123",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeForEntity(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeForEntity(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPalaceSessionKeywordRoom(t *testing.T) {
	session := &PalaceSession{}

	tests := []struct {
		name     string
		username string
		message  string
		expected string
	}{
		{
			name:     "question detected",
			username: "user",
			message:  "How do I use Go?",
			expected: "questions",
		},
		{
			name:     "thanks detected",
			username: "user",
			message:  "Thanks Pedro!",
			expected: "answers",
		},
		{
			name:     "off topic detected",
			username: "user",
			message:  "Nice weather today",
			expected: "off_topic",
		},
		{
			name:     "default to general",
			username: "user",
			message:  "Hello everyone",
			expected: "general",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := session.keywordRoom(tt.message)
			if result != tt.expected {
				t.Errorf("keywordRoom(%q) = %q; want %q", tt.message, result, tt.expected)
			}
		})
	}
}

func TestOutputLines(t *testing.T) {
	t.Run("splits multiline string", func(t *testing.T) {
		input := "line1\nline2\nline3"
		lines := outputLines(input)
		if len(lines) != 3 {
			t.Errorf("expected 3 lines, got %d", len(lines))
		}
	})

	t.Run("handles empty string", func(t *testing.T) {
		lines := outputLines("")
		if len(lines) != 0 {
			t.Errorf("expected 0 lines, got %d", len(lines))
		}
	})

	t.Run("handles no newlines", func(t *testing.T) {
		lines := outputLines("single line")
		if len(lines) != 1 {
			t.Errorf("expected 1 line, got %d", len(lines))
		}
	})
}

func TestContains(t *testing.T) {
	t.Run("finds substring", func(t *testing.T) {
		if !contains("hello world", []string{"world"}) {
			t.Error("expected to find 'world'")
		}
	})

	t.Run("returns false for missing substring", func(t *testing.T) {
		if contains("hello", []string{"world"}) {
			t.Error("expected not to find 'world'")
		}
	})

	t.Run("checks multiple substrings", func(t *testing.T) {
		if !contains("hello world", []string{"foo", "world"}) {
			t.Error("expected to find one of the substrings")
		}
	})
}

func TestContainsSubstring(t *testing.T) {
	t.Run("finds substring at start", func(t *testing.T) {
		if !containsSubstring("hello world", "hello") {
			t.Error("expected to find 'hello'")
		}
	})

	t.Run("finds substring at end", func(t *testing.T) {
		if !containsSubstring("hello world", "world") {
			t.Error("expected to find 'world'")
		}
	})

	t.Run("finds substring in middle", func(t *testing.T) {
		if !containsSubstring("hello beautiful world", "beautiful") {
			t.Error("expected to find 'beautiful'")
		}
	})

	t.Run("returns false for missing substring", func(t *testing.T) {
		if containsSubstring("hello", "xyz") {
			t.Error("expected not to find 'xyz'")
		}
	})
}

func TestTrimPrefix(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		prefix   string
		expected string
	}{
		{"removes prefix", "hello world", "hello", " world"},
		{"returns original if no prefix", "hello world", "xyz", "hello world"},
		{"handles empty string", "", "prefix", ""},
		{"handles empty prefix", "hello", "", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trimPrefix(tt.s, tt.prefix)
			if result != tt.expected {
				t.Errorf("trimPrefix(%q, %q) = %q; want %q", tt.s, tt.prefix, result, tt.expected)
			}
		})
	}
}
