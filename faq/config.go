package faq

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the full FAQ configuration file structure
type Config struct {
	// EmbeddingModel specifies which embedding model to use
	// Used to detect when model changes and trigger re-embedding
	EmbeddingModel string `yaml:"embedding_model"`

	// SimilarityThreshold is the minimum cosine similarity score to trigger a response
	// Default: 0.75 (recommended range: 0.70-0.85)
	SimilarityThreshold float64 `yaml:"similarity_threshold"`

	// DefaultCooldownSeconds is the default cooldown for entries that don't specify one
	DefaultCooldownSeconds int `yaml:"default_cooldown_seconds"`

	// Entries is the list of FAQ entries
	Entries []EntryConfig `yaml:"entries"`
}

// EntryConfig represents a single FAQ entry in the config file
type EntryConfig struct {
	// Question is the canonical question/trigger phrase for semantic matching
	Question string `yaml:"question"`

	// Response is the cached response or special token (e.g., FETCH_LATEST_VIDEO)
	Response string `yaml:"response"`

	// Category is optional grouping (youtube, social, schedule, events, content, etc.)
	Category string `yaml:"category,omitempty"`

	// IsActive indicates whether this FAQ entry is enabled
	// Defaults to true if not specified
	IsActive *bool `yaml:"is_active,omitempty"`

	// CooldownSeconds overrides the default cooldown for this specific entry
	// If not specified, uses DefaultCooldownSeconds from the config
	CooldownSeconds *int `yaml:"cooldown_seconds,omitempty"`
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		EmbeddingModel:         "text-embedding-3-small",
		SimilarityThreshold:    0.75,
		DefaultCooldownSeconds: 300,
		Entries:                []EntryConfig{},
	}
}

// LoadConfig loads FAQ configuration from a YAML file
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		return nil, fmt.Errorf("config path cannot be empty")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read FAQ config file %s: %w", configPath, err)
	}

	// Start with defaults
	config := DefaultConfig()

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse FAQ config YAML: %w", err)
	}

	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid FAQ config: %w", err)
	}

	// Apply defaults to entries
	applyEntryDefaults(config)

	return config, nil
}

// validateConfig ensures required fields are present and values are sensible
func validateConfig(config *Config) error {
	if config.EmbeddingModel == "" {
		return fmt.Errorf("embedding_model is required")
	}

	if config.SimilarityThreshold <= 0 || config.SimilarityThreshold > 1 {
		return fmt.Errorf("similarity_threshold must be between 0 and 1 (exclusive), got %f", config.SimilarityThreshold)
	}

	if config.DefaultCooldownSeconds < 0 {
		return fmt.Errorf("default_cooldown_seconds must be non-negative, got %d", config.DefaultCooldownSeconds)
	}

	if len(config.Entries) == 0 {
		return fmt.Errorf("at least one FAQ entry is required")
	}

	for i, entry := range config.Entries {
		if entry.Question == "" {
			return fmt.Errorf("entry %d: question is required", i)
		}
		if entry.Response == "" {
			return fmt.Errorf("entry %d: response is required", i)
		}
		if entry.CooldownSeconds != nil && *entry.CooldownSeconds < 0 {
			return fmt.Errorf("entry %d: cooldown_seconds must be non-negative", i)
		}
	}

	return nil
}

// applyEntryDefaults sets default values for entry fields that weren't specified
func applyEntryDefaults(config *Config) {
	for i := range config.Entries {
		entry := &config.Entries[i]

		// Default is_active to true
		if entry.IsActive == nil {
			isActive := true
			entry.IsActive = &isActive
		}

		// Use default cooldown if not specified
		if entry.CooldownSeconds == nil {
			entry.CooldownSeconds = &config.DefaultCooldownSeconds
		}
	}
}

// GetActiveCooldown returns the effective cooldown for an entry
func (e *EntryConfig) GetActiveCooldown(defaultCooldown int) int {
	if e.CooldownSeconds != nil {
		return *e.CooldownSeconds
	}
	return defaultCooldown
}

// IsEntryActive returns whether the entry is active (true if nil or true)
func (e *EntryConfig) IsEntryActive() bool {
	return e.IsActive == nil || *e.IsActive
}
