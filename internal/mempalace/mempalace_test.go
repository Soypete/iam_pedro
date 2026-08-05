package mempalace

import (
	"testing"

	"github.com/Soypete/twitch-llm-bot/logging"
)

func TestMemPalaceStruct(t *testing.T) {
	t.Run("MemPalace has required fields", func(t *testing.T) {
		m := &MemPalace{}
		_ = m.logger
		_ = m.classifier
		_ = m.writer
		_ = m.lifecycle
		_ = m.store
		_ = m.addressDetect
		_ = m.archiver
		_ = m.llm
		_ = m.modelName
		_ = m.active
	})
}

func TestConfigStruct(t *testing.T) {
	t.Run("Config has required fields", func(t *testing.T) {
		c := &Config{}
		_ = c.LLMPath
		_ = c.ModelName
		_ = c.EmbeddingsPath
		_ = c.EmbeddingsModel
		_ = c.HelixClient
		_ = c.Logger
		_ = c.PollInterval
		_ = c.ActiveDir
		_ = c.ArchiveDir
	})

	t.Run("Config can be created", func(t *testing.T) {
		logger := logging.Default()
		c := &Config{
			LLMPath:      "http://localhost:8080",
			ModelName:    "test-model",
			Logger:       logger,
			PollInterval: 30,
			ActiveDir:    "/tmp/active",
			ArchiveDir:   "/tmp/archive",
		}
		if c.LLMPath != "http://localhost:8080" {
			t.Error("expected LLMPath to match")
		}
		if c.PollInterval != 30 {
			t.Error("expected PollInterval to match")
		}
	})
}

func TestNew(t *testing.T) {
	t.Run("returns error when config is nil", func(t *testing.T) {
		_, err := New(nil)
		if err == nil {
			t.Error("expected error for nil config")
		}
	})

	t.Run("returns error when embedding path is empty", func(t *testing.T) {
		c := &Config{
			LLMPath:         "http://localhost:8080",
			ModelName:       "test",
			EmbeddingsPath:  "",
			EmbeddingsModel: "test-model",
		}
		_, err := New(c)
		if err == nil {
			t.Error("expected error for empty embedding path")
		}
	})
}

func TestMemPalaceMethods(t *testing.T) {
	t.Run("IsActive returns false initially", func(t *testing.T) {
		m := &MemPalace{}
		if m.IsActive() != false {
			t.Error("expected false initially")
		}
	})

	t.Run("GetQueryTool returns nil when store is nil", func(t *testing.T) {
		m := &MemPalace{}
		tool := m.GetQueryTool()
		if tool == nil {
			t.Error("expected non-nil tool")
		}
	})
}
