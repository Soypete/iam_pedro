package faq

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantErr   bool
		errMsg    string
		checkFunc func(t *testing.T, config *Config)
	}{
		{
			name: "valid config",
			content: `
embedding_model: "text-embedding-3-small"
similarity_threshold: 0.75
default_cooldown_seconds: 300
entries:
  - question: "What's your YouTube channel?"
    response: "https://youtube.com/@soypetetech"
    category: "social"
    is_active: true
`,
			wantErr: false,
			checkFunc: func(t *testing.T, config *Config) {
				assert.Equal(t, "text-embedding-3-small", config.EmbeddingModel)
				assert.Equal(t, 0.75, config.SimilarityThreshold)
				assert.Equal(t, 300, config.DefaultCooldownSeconds)
				assert.Len(t, config.Entries, 1)
				assert.Equal(t, "What's your YouTube channel?", config.Entries[0].Question)
				assert.True(t, config.Entries[0].IsEntryActive())
			},
		},
		{
			name: "default values applied",
			content: `
embedding_model: "text-embedding-3-small"
similarity_threshold: 0.8
default_cooldown_seconds: 600
entries:
  - question: "Test question"
    response: "Test response"
`,
			wantErr: false,
			checkFunc: func(t *testing.T, config *Config) {
				// is_active defaults to true
				assert.True(t, config.Entries[0].IsEntryActive())
				// cooldown defaults to default_cooldown_seconds
				assert.Equal(t, 600, config.Entries[0].GetActiveCooldown(0))
			},
		},
		{
			name: "missing embedding_model uses default",
			content: `
similarity_threshold: 0.75
default_cooldown_seconds: 300
entries:
  - question: "Test"
    response: "Response"
`,
			wantErr: false,
			checkFunc: func(t *testing.T, config *Config) {
				// Should use the default embedding model
				assert.Equal(t, "text-embedding-3-small", config.EmbeddingModel)
			},
		},
		{
			name: "invalid threshold (too high)",
			content: `
embedding_model: "test"
similarity_threshold: 1.5
default_cooldown_seconds: 300
entries:
  - question: "Test"
    response: "Response"
`,
			wantErr: true,
			errMsg:  "similarity_threshold must be between 0 and 1",
		},
		{
			name: "invalid threshold (zero)",
			content: `
embedding_model: "test"
similarity_threshold: 0
default_cooldown_seconds: 300
entries:
  - question: "Test"
    response: "Response"
`,
			wantErr: true,
			errMsg:  "similarity_threshold must be between 0 and 1",
		},
		{
			name: "no entries",
			content: `
embedding_model: "test"
similarity_threshold: 0.75
default_cooldown_seconds: 300
entries: []
`,
			wantErr: true,
			errMsg:  "at least one FAQ entry is required",
		},
		{
			name: "entry missing question",
			content: `
embedding_model: "test"
similarity_threshold: 0.75
default_cooldown_seconds: 300
entries:
  - response: "Response"
`,
			wantErr: true,
			errMsg:  "question is required",
		},
		{
			name: "entry missing response",
			content: `
embedding_model: "test"
similarity_threshold: 0.75
default_cooldown_seconds: 300
entries:
  - question: "Question"
`,
			wantErr: true,
			errMsg:  "response is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file with config content
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			err := os.WriteFile(configPath, []byte(tt.content), 0644)
			require.NoError(t, err)

			config, err := LoadConfig(configPath)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, config)

			if tt.checkFunc != nil {
				tt.checkFunc(t, config)
			}
		})
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read FAQ config file")
}

func TestLoadConfig_EmptyPath(t *testing.T) {
	_, err := LoadConfig("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config path cannot be empty")
}

func TestEntryConfig_IsEntryActive(t *testing.T) {
	tests := []struct {
		name     string
		isActive *bool
		want     bool
	}{
		{
			name:     "nil defaults to true",
			isActive: nil,
			want:     true,
		},
		{
			name:     "explicit true",
			isActive: boolPtr(true),
			want:     true,
		},
		{
			name:     "explicit false",
			isActive: boolPtr(false),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := EntryConfig{IsActive: tt.isActive}
			assert.Equal(t, tt.want, entry.IsEntryActive())
		})
	}
}

func TestEntryConfig_GetActiveCooldown(t *testing.T) {
	tests := []struct {
		name           string
		cooldownSec    *int
		defaultCooldown int
		want           int
	}{
		{
			name:           "nil uses default",
			cooldownSec:    nil,
			defaultCooldown: 300,
			want:           300,
		},
		{
			name:           "explicit value overrides default",
			cooldownSec:    intPtr(600),
			defaultCooldown: 300,
			want:           600,
		},
		{
			name:           "zero is valid",
			cooldownSec:    intPtr(0),
			defaultCooldown: 300,
			want:           0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := EntryConfig{CooldownSeconds: tt.cooldownSec}
			assert.Equal(t, tt.want, entry.GetActiveCooldown(tt.defaultCooldown))
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.Equal(t, "text-embedding-3-small", config.EmbeddingModel)
	assert.Equal(t, 0.75, config.SimilarityThreshold)
	assert.Equal(t, 300, config.DefaultCooldownSeconds)
	assert.Empty(t, config.Entries)
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}
