package ai

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultModerationConfig(t *testing.T) {
	config := DefaultModerationConfig()

	if config == nil {
		t.Fatal("DefaultModerationConfig returned nil")
	}

	// Check default values
	if config.Enabled {
		t.Error("default config should have enabled=false")
	}

	if config.SensitivityLevel != "moderate" {
		t.Errorf("default sensitivity level should be 'moderate', got %s", config.SensitivityLevel)
	}

	if len(config.AllowedTools) == 0 {
		t.Error("default config should have some allowed tools")
	}

	if config.RateLimits.ActionsPerMinute <= 0 {
		t.Error("default rate limits should have positive actions per minute")
	}

	if len(config.ChannelRules) == 0 {
		t.Error("default config should have channel rules")
	}
}

func TestIsToolAllowed(t *testing.T) {
	tests := []struct {
		name         string
		allowedTools []string
		toolName     string
		want         bool
	}{
		{
			name:         "allowed tool in list",
			allowedTools: []string{"no_action", "timeout_user", "ban_user"},
			toolName:     "timeout_user",
			want:         true,
		},
		{
			name:         "tool not in list",
			allowedTools: []string{"no_action", "timeout_user"},
			toolName:     "ban_user",
			want:         false,
		},
		{
			name:         "empty list allows all",
			allowedTools: []string{},
			toolName:     "any_tool",
			want:         true,
		},
		{
			name:         "nil list allows all",
			allowedTools: nil,
			toolName:     "any_tool",
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ModerationConfig{
				AllowedTools: tt.allowedTools,
			}

			if got := config.IsToolAllowed(tt.toolName); got != tt.want {
				t.Errorf("IsToolAllowed(%q) = %v, want %v", tt.toolName, got, tt.want)
			}
		})
	}
}

func TestIsChannelModerated(t *testing.T) {
	tests := []struct {
		name        string
		channels    []string
		channelName string
		want        bool
	}{
		{
			name:        "channel in list",
			channels:    []string{"soypetetech", "otherchannel"},
			channelName: "soypetetech",
			want:        true,
		},
		{
			name:        "channel not in list",
			channels:    []string{"soypetetech"},
			channelName: "otherchannel",
			want:        false,
		},
		{
			name:        "empty list moderates all",
			channels:    []string{},
			channelName: "anychannel",
			want:        true,
		},
		{
			name:        "nil list moderates all",
			channels:    nil,
			channelName: "anychannel",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ModerationConfig{
				Channels: tt.channels,
			}

			if got := config.IsChannelModerated(tt.channelName); got != tt.want {
				t.Errorf("IsChannelModerated(%q) = %v, want %v", tt.channelName, got, tt.want)
			}
		})
	}
}

func TestLoadModerationConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `
enabled: true
channels:
  - testchannel
sensitivity_level: conservative
allowed_tools:
  - no_action
  - warn_user
rate_limits:
  actions_per_minute: 5
  bans_per_hour: 2
  timeouts_per_user_per_hour: 2
channel_rules:
  - Be nice
  - No spam
dry_run: true
escalation:
  warnings_before_timeout: 3
  timeouts_before_ban: 5
  timeout_multiplier: 3.0
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	config, err := LoadModerationConfig(configPath)
	if err != nil {
		t.Fatalf("LoadModerationConfig failed: %v", err)
	}

	// Verify loaded values
	if !config.Enabled {
		t.Error("expected enabled=true")
	}

	if len(config.Channels) != 1 || config.Channels[0] != "testchannel" {
		t.Errorf("unexpected channels: %v", config.Channels)
	}

	if config.SensitivityLevel != "conservative" {
		t.Errorf("expected sensitivity_level='conservative', got %s", config.SensitivityLevel)
	}

	if len(config.AllowedTools) != 2 {
		t.Errorf("expected 2 allowed tools, got %d", len(config.AllowedTools))
	}

	if config.RateLimits.ActionsPerMinute != 5 {
		t.Errorf("expected actions_per_minute=5, got %d", config.RateLimits.ActionsPerMinute)
	}

	if !config.DryRun {
		t.Error("expected dry_run=true")
	}

	if config.Escalation.WarningsBeforeTimeout != 3 {
		t.Errorf("expected warnings_before_timeout=3, got %d", config.Escalation.WarningsBeforeTimeout)
	}
}

func TestLoadModerationConfig_FileNotFound(t *testing.T) {
	_, err := LoadModerationConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadModerationConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err = LoadModerationConfig(configPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestModerationPrompt(t *testing.T) {
	if ModerationPrompt == "" {
		t.Error("ModerationPrompt should not be empty")
	}

	// Check that the prompt contains placeholder markers
	if !strings.Contains(ModerationPrompt, "%s") {
		t.Error("ModerationPrompt should contain format placeholders")
	}
}
