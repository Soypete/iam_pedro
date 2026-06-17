# Mem Palace

Per-stream chat memory system for iam_pedro that provides long-term conversational context across live streams.

## Overview

Mem Palace enables Pedro to maintain contextual chat history during live streams and query that history when responding to questions. It's designed to run only during live streams, with each stream session having its own isolated SQLite database.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Twitch IRC                               │
│                    (peteTwitchChannel)                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────┐  │
│  │  Responder   │    │  Moderator   │    │   Mem Palace     │  │
│  │  (existing)  │    │  (existing)  │    │   (new writer)   │  │
│  └──────────────┘    └──────────────┘    └──────────────────┘  │
│         │                   │                     │             │
│         └───────────────────┼─────────────────────┘             │
│                             ▼                                   │
│                    ┌────────────────┐                          │
│                    │  Message       │                          │
│                    │  Classifier    │                          │
│                    │  (LLM-based)   │                          │
│                    └───────┬────────┘                          │
│                            ▼                                    │
│                   ┌────────────────┐                          │
│                   │  Topic Index   │                          │
│                   │  (Cosine sim)  │                          │
│                   └───────┬────────┘                          │
│                            ▼                                    │
│                  ┌────────────────────┐                       │
│                  │  SQLite per stream │                       │
│                  │  Session           │                       │
│                  └────────────────────┘                       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Components

### Ontology (`ontology/`)
- Parses SKOS thesaurus TTL files for topic taxonomy
- Builds in-memory cosine similarity index over topic labels
- Default: `testdata/twitch_topics.ttl` (~25 topics: Go, Python, LLM Engineering, etc.)

### Classifier (`classifier/`)
- Uses LLM to classify each chat message against ontology topics
- Returns topic + confidence score
- Embedding-based topic matching via cosine similarity

### Store (`store/`)
- One SQLite database per stream session
- Schema derived from ontology topics
- DDL: one table per topic class
- Message fields: id, stream_id, username, message, timestamp, confidence

### Writer (`writer/`)
- Third parallel IRC consumer (after responder + moderator)
- Receives all messages, classifies, writes to store
- Non-blocking - doesn't affect chat response latency

### Lifecycle (`lifecycle/`)
- Polls Twitch Helix API every 30s for stream status
- Detects stream start → creates new session
- Detects stream end → archives SQLite to `/data/palaces/archive`
- Publishes events via channel to other components

### Address Detection (`address/`)
- Determines if message is directed at Pedro (not just mention)
- Uses cosine similarity against "address terms" (hey, @pedro, etc.)
- Replaces simpler string contains check

### Archive (`archive/`)
- Moves SQLite from active to archive directory on session end
- Naming: `{stream_id}_{start_time}.db`

### Tools (`tools/`)
- `query_chat_history` LLM tool definition
- Enables Pedro to search past messages by topic or keywords

## Integration

### CLI Flags
```bash
./bin/twitch -model "gpt-oss-20b" \
  -enableMemPalace \
  -memPalaceActiveDir "/data/palaces/active" \
  -memPalaceArchiveDir "/data/palaces/archive"
```

### Environment Variables
- `MEMPALACE_DATA_DIR`: Base directory for palace data (default: `/data/palaces`)
- `MEMPALACE_ONTOLOGY_PATH`: Path to TTL file (default: `/app/internal/mempalace/ontology/testdata/twitch_topics.ttl`)

### K8s Deployment
- Volume mount: `/data/palaces` (read-write)
- PVC: `pedro-palaces` (1Gi default)
- Enable via `twitchBot.args` in values.yaml

## How It Works

1. **Stream Starts**: Lifecycle detects via Helix API → creates new session directory
2. **Message Arrives**: Writer receives via IRC → classifier assigns topic
3. **Classification**: LLM + cosine index determine topic + confidence
4. **Storage**: Message written to topic-specific table in session SQLite
5. **Query**: When Pedro receives question, tool can search historical context
6. **Stream Ends**: Lifecycle detects → archive moves SQLite to archive dir

## Design Decisions

- **No global vector store**: In-memory cosine index over ~100 ontology terms only
- **Per-session SQLite**: Isolation between streams, no cross-session contamination
- **Parallel writer**: Non-blocking classification/writing
- **LLM-based classification**: More accurate than keyword matching
- **Archive on end**: Preserves historical data for future analysis

## Related

- Existing `twitch/session_registry.go` - simpler palace system (separate)
- Existing `needsResponseChat()` - simple "pedro" string contains (replaced by address detector)