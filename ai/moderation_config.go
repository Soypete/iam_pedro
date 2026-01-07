package ai

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ModerationConfig holds configuration for the moderation system
type ModerationConfig struct {
	Enabled bool `yaml:"enabled"`

	// Channels to moderate (empty means moderate all channels)
	Channels []string `yaml:"channels"`

	// Sensitivity level: conservative, moderate, aggressive
	SensitivityLevel string `yaml:"sensitivity_level"`

	// Allowed tools (whitelist of tools that can be used)
	AllowedTools []string `yaml:"allowed_tools"`

	// Rate limits
	RateLimits RateLimits `yaml:"rate_limits"`

	// Channel rules to include in LLM context
	ChannelRules []string `yaml:"channel_rules"`

	// Dry run mode - log actions but don't execute
	DryRun bool `yaml:"dry_run"`

	// Escalation thresholds
	Escalation EscalationConfig `yaml:"escalation"`
}

// RateLimits defines rate limits for moderation actions
type RateLimits struct {
	// Maximum actions per minute
	ActionsPerMinute int `yaml:"actions_per_minute"`

	// Maximum bans per hour
	BansPerHour int `yaml:"bans_per_hour"`

	// Maximum timeouts per user per hour
	TimeoutsPerUserPerHour int `yaml:"timeouts_per_user_per_hour"`
}

// EscalationConfig defines how to escalate repeat offenders
type EscalationConfig struct {
	// Number of warnings before timeout
	WarningsBeforeTimeout int `yaml:"warnings_before_timeout"`

	// Number of timeouts before ban
	TimeoutsBeforeBan int `yaml:"timeouts_before_ban"`

	// Timeout duration multiplier for repeat offenses
	TimeoutMultiplier float64 `yaml:"timeout_multiplier"`
}

// DefaultModerationConfig returns the default moderation configuration
func DefaultModerationConfig() *ModerationConfig {
	return &ModerationConfig{
		Enabled:          false,
		Channels:         []string{},
		SensitivityLevel: "moderate",
		AllowedTools: []string{
			"no_action",
			"warn_user",
			"timeout_user",
			"delete_message",
		},
		RateLimits: RateLimits{
			ActionsPerMinute:       10,
			BansPerHour:            5,
			TimeoutsPerUserPerHour: 3,
		},
		ChannelRules: []string{
			"Be respectful to all community members",
			"No spam or self-promotion",
			"No hate speech or harassment",
			"Keep discussions relevant to the stream",
		},
		DryRun: false,
		Escalation: EscalationConfig{
			WarningsBeforeTimeout: 2,
			TimeoutsBeforeBan:     3,
			TimeoutMultiplier:     2.0,
		},
	}
}

// LoadModerationConfig loads a moderation configuration from a YAML file
func LoadModerationConfig(path string) (*ModerationConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read moderation config file: %w", err)
	}

	config := DefaultModerationConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse moderation config: %w", err)
	}

	return config, nil
}

// IsToolAllowed checks if a tool is in the allowed list
func (c *ModerationConfig) IsToolAllowed(toolName string) bool {
	if len(c.AllowedTools) == 0 {
		return true // No whitelist means all tools allowed
	}

	for _, allowed := range c.AllowedTools {
		if allowed == toolName {
			return true
		}
	}
	return false
}

// IsChannelModerated checks if a channel should be moderated
func (c *ModerationConfig) IsChannelModerated(channelName string) bool {
	if len(c.Channels) == 0 {
		return true // No channel list means moderate all channels
	}

	for _, ch := range c.Channels {
		if ch == channelName {
			return true
		}
	}
	return false
}

// ModerationPrompt is the system prompt for the moderation LLM
var ModerationPrompt = `You are a Twitch chat moderator assistant for SoyPeteTech's channel. Your role is to evaluate chat messages and decide if moderation action is needed.

Channel Rules:
%s

Guidelines for moderation:
1. Be CONSERVATIVE - only act on CLEAR violations. When in doubt, use no_action.
2. For first-time minor violations, prefer warn_user over timeout_user.
3. Use timeout_user for moderate violations or repeated warnings (start with 60-300 seconds).
4. Only use ban_user for SEVERE violations like hate speech, harassment, spam bots, or severe repeated offenses.
5. Use delete_message when a single message violates rules but the user doesn't need a timeout.
6. NEVER moderate messages that are just off-topic, jokes, or friendly banter.
7. NEVER moderate messages from the streamer or other moderators.

When evaluating a message, consider:
- The message content and intent
- Whether it targets or harms others
- Whether it's spam or self-promotion
- The context of recent chat messages
- The user's history (if provided)

You MUST call exactly one tool for each message you evaluate. If no moderation is needed, call no_action with a brief reason.

Sensitivity Level: %s
- conservative: Only act on obvious, severe violations
- moderate: Act on clear violations, give benefit of doubt
- aggressive: Act on potential violations, err on side of caution`
