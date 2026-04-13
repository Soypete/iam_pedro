# Mem Palace Implementation Plan

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        TWITCH POD                                       │
│  ┌─────────────────────┐    ┌─────────────────────────────────────────┐ │
│  │   Twitch Bot        │    │          Mem Palace Sidecar            │ │
│  │   (main container)  │    │       (internal/mempalace/)            │ │
│  │                     │    │                                         │ │
│  │  ┌───────────────┐  │    │  ┌─────────────┐  ┌───────────────┐   │ │
│  │  │ IRC Consumer  │──┼────┼─▶│  Classifier │  │ Writer        │   │ │
│  │  │ (parallel)    │  │    │  │  (LLM)      │  │ (parallel)    │   │ │
│  │  └───────────────┘  │    │  └──────┬──────┘  └───────┬───────┘   │ │
│  │                     │    │         │                 │            │ │
│  │  ┌───────────────┐  │    │  ┌──────▼──────┐  ┌──────▼───────┐   │ │
│  │  │ Moderation    │  │    │  │Ontology     │  │  SQLite      │   │ │
│  │  │ Monitor       │  │    │  │Index        │  │  Store       │   │ │
│  │  └───────────────┘  │    │  │(cosine)     │  │(per-stream)  │   │ │
│  │                     │    │  └─────────────┘  └───────────────┘   │ │
│  │  ┌───────────────┐  │    │  ┌─────────────────────────────────┐  │ │
│  │  │ Main LLM      │◀─┼────┼──│  Tool: query_chat_history       │  │ │
│  │  │ Responder     │  │    │  └─────────────────────────────────┘  │ │
│  │  └───────────────┘  │    │                                         │ │
│  └─────────────────────┘    └─────────────────────────────────────────┘ │
│           │                              │                              │
│           │         ┌────────────────────┴────────────────────┐        │
│           │         │         Helix Polling (30s)             │        │
│           │         │  lifecycle/ - Stream.online/offline    │        │
│           │         └─────────────────────────────────────────┘        │
│           │                                                         │
└───────────┼───────────────────────────────────────────────────────────┘
            │ Shared Volume (PVC: pedro-palaces)
            │ /data/palaces/active/{stream_id}.sqlite
            │ /data/palaces/archive/{started_at}-{stream_id}.sqlite
```

## Package Layout

```
internal/mempalace/
├── ontology/
│   ├── loader.go        # Load TTL/JSON-LD ontology
│   ├── index.go         # In-memory cosine index (~100 terms)
│   └── testdata/
│       └── tbox_learning_software.ttl  # Vendored from professor_pedro
│
├── classifier/
│   └── classifier.go    # LLM-based classification to ontology classes
│                       # Reuses existing tool-calling pattern
│
├── store/
│   ├── store.go         # SQLite per-stream, DDL from ontology
│   ├── messages.go      # Message CRUD
│   └── queries.go       # Query by topic, time range
│
├── writer/
│   └── writer.go        # Third parallel consumer of IRC stream
│                       # Gated on lifecycle state
│
├── lifecycle/
│   ├── controller.go    # Helix polling (30s tick)
│   ├── events.go        # SessionStart/SessionEnd events
│   └── recovery.go      # Crash recovery on startup
│
├── tools/
│   └── query_chat_history.go  # Responder tool
│
├── archive/
│   └── archiver.go      # SessionEnd: move to archive, alert on failure
│
├── address/
│   └── detector.go      # Cosine-based "Pedro" address detection
│
└── mempalace.go         # Main entry, wires all components
```

## Key Interfaces

```go
// Lifecycle - stream session management
type Lifecycle interface {
    Start(ctx context.Context) error      // Begin Helix polling
    Stop() error                          // Stop polling
    Events() <-chan SessionEvent          // SessionStart/SessionEnd
    IsActive() bool                       // Current stream status
    StreamID() string                     // Current stream ID
}

// Classifier - message to ontology class
type Classifier interface {
    Classify(ctx context.Context, msg string) (string, error)  // Returns class name
}

// Store - per-stream SQLite
type Store interface {
    Init(streamID string) error
    WriteMessage(ctx context.Context, msg Message) error
    Query(ctx context.Context, opts QueryOpts) ([]Message, error)
    Close() error
}

// Writer - consumes IRC messages (parallel consumer)
type Writer interface {
    Write(ctx context.Context, msg types.TwitchMessage) error
    Start(ctx context.Context, wg *sync.WaitGroup) error
    Stop() error
}
```

## SQLite Schema (Generated from Ontology)

For each ontology class (e.g., `sw:LLMEngineering`, `sw:Go`, `sw:Infrastructure`):
```sql
CREATE TABLE IF NOT EXISTS messages_llm_engineering (
    id TEXT PRIMARY KEY,
    stream_id TEXT NOT NULL,
    username TEXT NOT NULL,
    message TEXT NOT NULL,
    timestamp TEXT NOT NULL,
    classification_confidence REAL
);

-- Plus fallback for unclassified
CREATE TABLE IF NOT EXISTS messages_raw (
    id TEXT PRIMARY KEY,
    stream_id TEXT NOT NULL,
    username TEXT NOT NULL,
    message TEXT NOT NULL,
    timestamp TEXT NOT NULL
);
```

## Implementation Steps

### Step 1: Ontology Loader + Cosine Index
- Vendor `TBOX_LEARNING_SOFTWARE.ttl` from professor_pedro
- Add missing classes: `TwitchBots`, `OpenSource`
- Parse TTL, extract `rdfs:label` + `skos:altLabel`
- Embed via existing FAQ embedding service (uses LLM endpoint)
- In-memory cosine index over ~100 term vectors

### Step 2: Classifier
- Reuse existing `llms.WithTools()` pattern
- Grammar/JSON output: `{ "topic": "LLMEngineering", "confidence": 0.95 }`
- Or simpler: tool with one function call per class

### Step 3: Per-Stream SQLite Store
- Path: `/data/palaces/active/{stream_id}.sqlite`
- DDL generated from ontology classes
- Survives pod restart

### Step 4: Writer (Parallel Consumer)
- Same pattern as moderation monitor
- Receives messages via channel from IRC
- Only active when lifecycle says stream is live

### Step 5: Lifecycle Controller
- Add to `twitch/helix/client.go`: `GetStreamStatus(ctx, userID) (*StreamStatus, error)`
- Polling: 30s interval
- Events channel for SessionStart/SessionEnd

### Step 6: Responder Tool
- Register with existing tool registry
- Signature: `query_chat_history(topic_class, time_range, query_text, limit)`

### Step 7: Address Detector
- Replace existing stringContains with cosine over exemplars
- Embedding model: same as ontology (all-MiniLM-L6-v2)
- Expose score distribution via Prometheus histogram

### Step 8: Archive
- On SessionEnd: close SQLite, move to `/archive/`
- On failure: alert (reusing existing alert system)

### Step 9: Prometheus Metrics
Add to `metrics/server.go`:
```go
var (
    MempalaceSessionActive = prometheus.NewGauge(...)
    MempalaceMessagesClassifiedTotal = prometheus.NewCounterVec(...)
    MempalaceClassificationLatency = prometheus.NewHistogram(...)
    MempalaceClassificationUnclassifiedTotal = prometheus.NewCounter(...)
    MempalaceSQLiteWriteLatency = prometheus.NewHistogram(...)
    MempalaceToolCallsTotal = prometheus.NewCounterVec(...)
    MempalacePedroAddressScore = prometheus.NewHistogram(...)
    MempalaceArchiveFailuresTotal = prometheus.NewCounterVec(...)
)
```

### Step 10: K8s Sidecar Modification
Modify `charts/pedro-bots/templates/twitch-deployment.yaml`:
- Add mempalace container as sidecar
- Share volume: emptyDir or same PVC
- Remove separate mempalace deployment

## Decisions Made

| Decision | Choice |
|----------|--------|
| Sidecar | Yes, in Twitch pod |
| Classifier model | Same as chat model (gpt-oss-20b) |
| Embedding model | all-MiniLM-L6-v2 via existing LLM endpoint |
| Archive path | Same PVC: `/data/palaces/archive/` |
| Existing palace_session.go | Remove |
| Helix client | Add to existing `twitch/helix/client.go` |

## Dependencies

- **Existing**: `github.com/tmc/langchaingo` (LLM client)
- **Existing**: `github.com/mattn/go-sqlite3` (if not already present)
- **New**: None required (embedding via existing FAQ service pattern)