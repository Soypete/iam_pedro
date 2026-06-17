// Package archive moves Mem Palace session databases from active to archive
// directory when a stream ends.
//
// Archived databases are named: {stream_id}_{start_time}.db
// They remain available for future analysis or debugging.
package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Soypete/twitch-llm-bot/metrics"
)

type Archiver struct {
	activeDir  string
	archiveDir string
}

func NewArchiver(activeDir, archiveDir string) *Archiver {
	if activeDir == "" {
		activeDir = "/data/palaces/active"
	}
	if archiveDir == "" {
		archiveDir = "/data/palaces/archive"
	}

	return &Archiver{
		activeDir:  activeDir,
		archiveDir: archiveDir,
	}
}

func (a *Archiver) Archive(streamID string, startedAt time.Time) error {
	sourcePath := filepath.Join(a.activeDir, fmt.Sprintf("%s.sqlite", streamID))

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		metrics.MempalaceArchiveFailuresTotal.WithLabelValues("file_not_found").Add(1)
		return fmt.Errorf("source file does not exist: %s", sourcePath)
	}

	if err := os.MkdirAll(a.archiveDir, 0755); err != nil {
		metrics.MempalaceArchiveFailuresTotal.WithLabelValues("mkdir_failed").Add(1)
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	filename := fmt.Sprintf("%s-%s.sqlite",
		startedAt.Format("2006-01-02-1504"),
		streamID,
	)
	destPath := filepath.Join(a.archiveDir, filename)

	if err := os.Rename(sourcePath, destPath); err != nil {
		metrics.MempalaceArchiveFailuresTotal.WithLabelValues("rename_failed").Add(1)
		return fmt.Errorf("failed to archive file: %w", err)
	}

	return nil
}

func (a *Archiver) ArchivePath() string {
	return a.archiveDir
}
