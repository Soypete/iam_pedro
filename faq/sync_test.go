package faq

import (
	"testing"
)

func TestSyncResult(t *testing.T) {
	t.Run("SyncResult has default values", func(t *testing.T) {
		result := &SyncResult{
			EntriesProcessed: 0,
			EntriesCreated:   0,
			EntriesUpdated:   0,
			EntriesDeleted:   0,
			Errors:           nil,
			Duration:         0,
		}

		if result.EntriesProcessed != 0 {
			t.Errorf("expected 0, got %d", result.EntriesProcessed)
		}
		if result.Errors != nil {
			t.Error("expected nil errors")
		}
	})

	t.Run("SyncResult can track errors", func(t *testing.T) {
		result := &SyncResult{}
		result.Errors = append(result.Errors, nil)

		if len(result.Errors) != 1 {
			t.Errorf("expected 1 error, got %d", len(result.Errors))
		}
	})
}

func TestSyncerStruct(t *testing.T) {
	t.Run("Syncer has required fields", func(t *testing.T) {
		syncer := &Syncer{}
		_ = syncer.db
		_ = syncer.embeddingService
		_ = syncer.logger
	})
}

func TestFAQEntryStruct(t *testing.T) {
	t.Run("FAQEntry has required fields", func(t *testing.T) {
		entry := FAQEntry{
			Question:        "What is Go?",
			Response:        "Go is a programming language.",
			Category:        func(s string) *string { return &s }("programming"),
			IsActive:        true,
			CooldownSeconds: 300,
		}

		if entry.Question != "What is Go?" {
			t.Errorf("expected question, got %s", entry.Question)
		}
		if !entry.IsActive {
			t.Error("expected IsActive to be true")
		}
	})
}
