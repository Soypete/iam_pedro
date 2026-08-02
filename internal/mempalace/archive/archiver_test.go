package archive

import (
	"testing"
)

func TestArchiverStruct(t *testing.T) {
	t.Run("Archiver has required fields", func(t *testing.T) {
		archiver := &Archiver{}
		_ = archiver.activeDir
		_ = archiver.archiveDir
	})
}

func TestNewArchiver(t *testing.T) {
	t.Run("uses default directories when empty", func(t *testing.T) {
		archiver := NewArchiver("", "")
		if archiver.activeDir != "/data/palaces/active" {
			t.Errorf("expected /data/palaces/active, got %s", archiver.activeDir)
		}
		if archiver.archiveDir != "/data/palaces/archive" {
			t.Errorf("expected /data/palaces/archive, got %s", archiver.archiveDir)
		}
	})

	t.Run("uses provided directories", func(t *testing.T) {
		archiver := NewArchiver("/custom/active", "/custom/archive")
		if archiver.activeDir != "/custom/active" {
			t.Errorf("expected /custom/active, got %s", archiver.activeDir)
		}
		if archiver.archiveDir != "/custom/archive" {
			t.Errorf("expected /custom/archive, got %s", archiver.archiveDir)
		}
	})
}

func TestArchivePath(t *testing.T) {
	t.Run("returns archive directory", func(t *testing.T) {
		archiver := NewArchiver("", "/my/archive")
		if archiver.ArchivePath() != "/my/archive" {
			t.Errorf("expected /my/archive, got %s", archiver.ArchivePath())
		}
	})
}
