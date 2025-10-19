# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

**IAM_PEDRO** is a multi-platform AI chatbot for Twitch and Discord that integrates with LLMs via llama.cpp, includes web search capabilities, database persistence, and metrics monitoring. Pedro serves as an AI assistant for SoyPeteTech's streaming community.

## Build and Development Commands

### Building
```bash
# Build all packages
go build ./...

# Build specific platforms
go build -o bin/discord ./cli/discord
go build -o bin/twitch ./cli/twitch

# Run Discord bot locally
go run ./cli/discord -model "your-model-name" -errorLevel debug

# Run Twitch bot locally
go run ./cli/twitch -model "your-model-name" -errorLevel debug
```

### Testing
```bash
# Run all tests with coverage
go test ./... -v -cover -covermode=atomic

# Run specific package tests
go test ./ai/twitchchat/ -v
go test ./duckduckgo/ -v
```

### Linting
```bash
# Run golangci-lint (installed as tool)
go tool -modfile=golangci-lint.mod golangci-lint run

# Update golangci-lint
go get -tool -modfile=golangci-lint.mod github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
```

### Database Operations
```bash
# Run migrations (using goose)
goose -dir database/migrations postgres $POSTGRES_URL up

# Create new migration
goose -dir database/migrations create migration_name sql
```

### Docker
```bash
# Build containers
docker build -f discordBot.Dockerfile -t pedro-discord .
docker build -f twitchBot.Dockerfile -t pedro-twitch .

# Run with required environment variables
docker run -e LLAMA_CPP_PATH="http://127.0.0.1:8080" \
  -e POSTGRES_URL="" -e TWITCH_ID="" -e TWITCH_SECRET="" \
  -e POSTGRES_VECTOR_URL="" pedro-twitch
```

### Deployment
```bash
# Build and deploy both services (uses current git commit as tag)
./deployment/build-and-deploy.sh

# Build specific service with custom tag
./deployment/build-and-deploy.sh v1.2.3 discord
./deployment/build-and-deploy.sh v1.2.3 twitch

# After building, copy to remote host
scp pedro-discord-<TAG>.tar.gz deployment/deploy-*.sh remote-deploy.sh user@100.81.89.62:~/

# On remote host, deploy the containers
ssh user@100.81.89.62
./remote-deploy.sh

# Check service status
sudo systemctl status pedro-discord
sudo systemctl status pedro-twitch

# View logs
sudo journalctl -u pedro-discord -f
sudo journalctl -u pedro-twitch -f
```

## Architecture

### Core Components

**Two Separate Entry Points**
The project has two independent CLI applications in `cli/`:
- **Discord bot** (`cli/discord/discord.go`): Handles Discord slash commands and interactions
- **Twitch bot** (`cli/twitch/twitch.go`): Connects to Twitch IRC and responds to chat messages

Each bot runs as a separate process with its own systemd service in production.

**AI Layer (`/ai/`)**
- `chatter.go` - Core interface defining `Chatter` contract for all bot implementations
- `twitchchat/` - Twitch-specific LLM client with async web search integration
  - Maintains rolling chat history (max 10 messages)
  - Uses system prompt from `ai.PedroPrompt`
  - Implements `SingleMessageResponse()` for one-off chat replies
- `discordchat/` - Discord-specific LLM client with 20 questions game support
  - Supports `/askPedro` for single questions
  - Supports `/stumpPedro` for interactive 20 questions game
  - Different interface than Twitch due to Discord's threading model
- Pedro's personality is defined in `ai.PedroPrompt` constant

**Platform Integration**
- `/discord/` - Discord bot with slash commands (`/askPedro`, `/stumpPedro`, `/helpPedro`)
  - `setup.go`: Discord session setup and command registration
  - `commands.go`: Command definitions (`AddCommands()`)
  - `ask.go`: `/askPedro` handler
  - `stump_pedro.go`: `/stumpPedro` handler for 20 questions
- `/twitch/` - Twitch IRC client with OAuth authentication and message handling
  - `connection.go`: IRC setup and lifecycle (`SetupTwitchIRC`, `ConnectIRC`)
  - `handleMessage.go`: Message parsing and routing to LLM
  - `auth.go`: OAuth token management for Twitch API
  - Hardcoded channel: `peteTwitchChannel = "soypetetech"`
- `/types/` - Shared data structures including `TwitchMessage` and `WebSearchRequest`

**External Services**
- `/duckduckgo/` - Web search client for real-time information retrieval
- LLM integration via langchain-go connecting to llama.cpp (not OpenAI)

**Infrastructure**
- `/database/` - PostgreSQL with embedded migrations, supports vector embeddings
  - Uses `sqlx` and `lib/pq` for Postgres connectivity
  - Uses `goose` for embedded migrations (in `database/migrations/`)
  - **Important**: Database migrations run automatically on startup in `NewPostgres()`
  - **Warning**: `database/connect-postgres.go` contains a `TODO: do not commit` with a `goose.DownTo()` call that should be removed before production merges
  - Two writer interfaces:
    - `ChatResponseWriter`: For Twitch message storage
    - `DiscordWriter`: For Discord message storage
- `/logging/` - Structured JSON logging with slog
  - Supports log levels: debug, info, warn, error
  - Configured via `-errorLevel` CLI flag
- `/metrics/` - Prometheus metrics + expvar + pprof
  - Discord bot exposes metrics on port 6060
  - Twitch bot exposes metrics on port 6061
  - Prometheus server runs on separate host (100.125.196.1:9090)

### Data Flow

1. **Message Reception**: Discord slash commands or Twitch chat messages
2. **AI Processing**: Messages go through platform-specific LLM clients
3. **Web Search Integration**: If Pedro responds with "execute web search", triggers async DuckDuckGo lookup
4. **Response Handling**: Immediate response + async follow-up with search results
5. **Persistence**: All interactions stored in PostgreSQL with chat history management

### Key Environment Variables

```bash
# Required for both bots
LLAMA_CPP_PATH="http://127.0.0.1:8080"  # llama.cpp server endpoint
POSTGRES_URL=postgres://...              # Main database connection
POSTGRES_VECTOR_URL=""                   # Vector database for embeddings

# Discord bot
DISCORD_SECRET=...                       # Discord bot token

# Twitch bot
TWITCH_ID=...                           # Twitch OAuth client ID
TWITCH_SECRET=...                       # Twitch OAuth client secret
```

## Important Implementation Details

### Web Search Flow
- Pedro can trigger web searches by responding with "execute web search [query]"
- Immediate response: "one second and I will look that up for you soypet2Thinking"
- Async function performs DuckDuckGo search and generates informed follow-up response
- Chat history is preserved and passed to the async search function

### Database Schema
Key tables: `twitch_chat`, `bot_response`, `discord_ask_pedro`, `discord_twenty_questions_games`

### LLM Integration
- Uses langchain-go, not direct OpenAI API
- Connects to local llama.cpp server (development) or containerized deployment
- OpenAI-compatible API pattern via langchain-go:
  1. Set fake `OPENAI_API_KEY` environment variable (required by library)
  2. Use `LLAMA_CPP_PATH` environment variable for the actual LLM endpoint
  3. The setup functions automatically append `/v1` to the path if needed
  4. Pass model name via `-model` CLI flag (must match model loaded in llama.cpp)
- Context window management: chat history limited to 10 messages
- Temperature: 0.7, Max length: 500 characters

### Platform-Specific Features
- **Twitch**: Real-time chat integration, responds to mentions of "pedro", "Pedro", "llm", "LLM", "bot"
  - Specific trigger patterns match (implemented in `twitch/handleMessage.go`)
- **Discord**: Thread-based conversations, 20 questions game, slash command interface

### Pedro's Personality
- Assistant for SoyPeteTech (Miriah Peterson), a software streamer in Utah
- Focuses on Golang and Data/AI content
- Uses custom emotes: soypet2Thinking, soypet2Dance, soypet2ConfusedPedro, etc.
- Explicitly avoids Java and JavaScript discussions
- Response limit: 500 characters, no newlines

### LLM Response Cleanup
- All LLM responses are cleaned via `ai.CleanResponse()` which:
  - Removes newlines (Twitch has strict message format requirements)
  - Strips instruction markers (`<|im_start|>`, `<|im_end|>`)
  - Removes leading `!` and `/` to prevent command injection

### Metrics and Monitoring
- Prometheus metrics exposed on ports 6060/6061
- Use `metrics.<MetricName>.Add(1)` to increment counters
- Key metrics: `TwitchConnectionCount`, `TwitchMessageRecievedCount`, `SuccessfulLLMGen`, `FailedLLMGen`, `EmptyLLMResponse`
- Tracks message counts, LLM generation success/failure, response times
- Structured logging with trace IDs for request correlation

## Testing
- Unit tests for core logic in `ai/helpers_test.go` and `ai/twitchchat/llm_test.go`
- Integration tests for message handling in `twitch/handleMessage_test.go`
- Use table-driven tests following Go conventions

## Future Development (From README)

The project is evolving toward structured agentic planning with:
- Step-based reasoning workflows
- DAG-style orchestration
- Prescribed planning actions: `api_call`, `db_query`, `web_search`, `reference_check`, `return_response`
- Live workflow feedback in Discord threads
