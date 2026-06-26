package mempalace

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Soypete/twitch-llm-bot/faq"
	"github.com/Soypete/twitch-llm-bot/internal/mempalace/address"
	"github.com/Soypete/twitch-llm-bot/internal/mempalace/archive"
	"github.com/Soypete/twitch-llm-bot/internal/mempalace/classifier"
	"github.com/Soypete/twitch-llm-bot/internal/mempalace/lifecycle"
	"github.com/Soypete/twitch-llm-bot/internal/mempalace/ontology"
	"github.com/Soypete/twitch-llm-bot/internal/mempalace/store"
	"github.com/Soypete/twitch-llm-bot/internal/mempalace/tools"
	"github.com/Soypete/twitch-llm-bot/internal/mempalace/writer"
	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/Soypete/twitch-llm-bot/twitch/helix"
	v2 "github.com/gempir/go-twitch-irc/v2"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type MemPalace struct {
	logger        *logging.Logger
	helixClient   *helix.Client
	classifier    *classifier.Classifier
	writer        *writer.Writer
	lifecycle     *lifecycle.Controller
	store         *store.Store
	addressDetect *address.Detector
	archiver      *archive.Archiver

	llm       llms.Model
	modelName string

	mu     sync.RWMutex
	active bool
}

type Config struct {
	// LLMPath / ModelName are the chat server (used by the classifier LLM).
	LLMPath   string
	ModelName string

	// EmbeddingsPath / EmbeddingsModel are the dedicated embeddings server. The
	// chat server runs MTP, which is incompatible with the embeddings graph, so
	// embeddings live on a separate server (e.g. a sidecar on localhost:8081).
	EmbeddingsPath  string
	EmbeddingsModel string

	HelixClient  *helix.Client
	Logger       *logging.Logger
	PollInterval int

	ActiveDir  string
	ArchiveDir string
}

func New(config *Config) (*MemPalace, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	logger := config.Logger
	if logger == nil {
		logger = logging.Default()
	}

	ontologyPath := os.Getenv("MEMPALACE_ONTOLOGY_PATH")
	if ontologyPath == "" {
		ontologyPath = "/app/internal/mempalace/ontology/testdata/twitch_topics.ttl"
	}

	// Embeddings run on a dedicated server, not the MTP chat server. Require it
	// explicitly — reusing the chat path would hit a server that no longer serves
	// /v1/embeddings and fail at runtime instead of startup.
	if config.EmbeddingsPath == "" {
		return nil, fmt.Errorf("EmbeddingsPath is required (set EMBEDDINGS_PATH); the chat server does not serve embeddings")
	}
	embeddingsModel := config.EmbeddingsModel
	if embeddingsModel == "" {
		embeddingsModel = "nomic-embed-text"
	}
	embedder, err := faq.NewEmbeddingService(config.EmbeddingsPath, embeddingsModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	index, err := ontology.NewIndex(embedder)
	if err != nil {
		return nil, fmt.Errorf("failed to create ontology index: %w", err)
	}

	if err := index.LoadTTL(context.Background(), ontologyPath); err != nil {
		return nil, fmt.Errorf("failed to load ontology: %w", err)
	}

	classes := index.GetClasses()

	llm, err := openai.New([]openai.Option{
		openai.WithBaseURL(config.LLMPath),
		openai.WithModel(config.ModelName),
	}...)
	if err != nil {
		return nil, fmt.Errorf("failed to create classifier LLM: %w", err)
	}

	classifr := classifier.NewClassifier(llm, config.ModelName, classes)

	addrDetect, err := address.NewDetector(embedder, 0.6)
	if err != nil {
		return nil, fmt.Errorf("failed to create address detector: %w", err)
	}

	activeDir := config.ActiveDir
	if activeDir == "" {
		activeDir = "/data/palaces/active"
	}
	archiveDir := config.ArchiveDir
	if archiveDir == "" {
		archiveDir = "/data/palaces/archive"
	}

	archiver := archive.NewArchiver(activeDir, archiveDir)

	pollInterval := time.Duration(config.PollInterval)
	if pollInterval == 0 {
		pollInterval = 30 * time.Second
	}

	lc := lifecycle.NewController(config.HelixClient, pollInterval)

	wr := writer.NewWriter(classifr, lc, logger)

	return &MemPalace{
		logger:        logger,
		helixClient:   config.HelixClient,
		classifier:    classifr,
		writer:        wr,
		lifecycle:     lc,
		addressDetect: addrDetect,
		archiver:      archiver,
		llm:           llm,
		modelName:     config.ModelName,
	}, nil
}

func (m *MemPalace) Start(ctx context.Context, wg *sync.WaitGroup) error {
	if err := m.lifecycle.Start(ctx); err != nil {
		return fmt.Errorf("failed to start lifecycle controller: %w", err)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-m.lifecycle.Events():
				m.handleSessionEvent(ctx, event)
			}
		}
	}()

	m.mu.Lock()
	m.active = true
	m.mu.Unlock()

	return nil
}

func (m *MemPalace) handleSessionEvent(ctx context.Context, event lifecycle.SessionEvent) {
	switch event.Type {
	case lifecycle.EventSessionStart:
		m.logger.Info("starting mempalace session", "streamID", event.StreamID)

		s := store.NewStore()
		if err := s.Init(event.StreamID, m.classifier.GetClasses()); err != nil {
			m.logger.Error("failed to init store", "error", err)
			return
		}

		m.store = s
		m.writer.SetStore(s)

	case lifecycle.EventSessionEnd:
		m.logger.Info("ending mempalace session", "streamID", event.StreamID)

		if m.store != nil {
			_ = m.store.Close()
			m.store = nil
		}

		_ = m.archiver.Archive(event.StreamID, m.lifecycle.GetStartedAt())
	}
}

func (m *MemPalace) MessageChannel() chan<- v2.PrivateMessage {
	return m.writer.MessageChannel()
}

func (m *MemPalace) IsAddressed(msg string) (bool, float32) {
	return m.addressDetect.IsAddressed(msg)
}

func (m *MemPalace) GetQueryTool() *tools.QueryChatHistoryTool {
	return tools.NewQueryChatHistoryTool(m.store, nil, nil)
}

func (m *MemPalace) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.lifecycle.Stop(); err != nil {
		return err
	}

	if m.store != nil {
		_ = m.store.Close()
	}

	m.active = false
	return nil
}

func (m *MemPalace) IsActive() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.active
}
