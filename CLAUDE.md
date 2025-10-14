# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

IAM_PEDRO is a dual-platform LLM-powered chat bot that operates on both Twitch and Discord. The bot uses llama.cpp as the LLM backend (accessed via OpenAI-compatible API), langchain-go for LLM interactions, and Postgres for message persistence. The project is written in Go.

## Common Commands

### Development
```bash
# Build Discord bot
go build -o bin/discord ./cli/discord

# Build Twitch bot
go build -o bin/twitch ./cli/twitch

# Run Discord bot locally
go run ./cli/discord -model "your-model-name" -errorLevel debug

# Run Twitch bot locally
go run ./cli/twitch -model "your-model-name" -errorLevel debug

# Run tests
go test ./...

# Run specific package tests
go test ./ai/twitchchat/...

# Lint
golangci-lint run
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

## Architecture Overview

### Two Separate Entry Points
The project has two independent CLI applications in `cli/`:
- **Discord bot** (`cli/discord/discord.go`): Handles Discord slash commands and interactions
- **Twitch bot** (`cli/twitch/twitch.go`): Connects to Twitch IRC and responds to chat messages

Each bot runs as a separate process with its own systemd service in production.

### Platform-Specific LLM Clients
The `ai/` package contains platform-specific implementations:
- **`ai/twitchchat/`**: Twitch-specific LLM client implementing the `ai.Chatter` interface
  - Maintains rolling chat history (max 10 messages)
  - Uses system prompt from `ai.PedroPrompt`
  - Implements `SingleMessageResponse()` for one-off chat replies
- **`ai/discordchat/`**: Discord-specific LLM client implementing the `LLM` interface
  - Supports `/askPedro` for single questions
  - Supports `/stumpPedro` for interactive 20 questions game
  - Different interface than Twitch due to Discord's threading model

### Database Layer (`database/`)
- Uses `sqlx` and `lib/pq` for Postgres connectivity
- Uses `goose` for embedded migrations (in `database/migrations/`)
- **Important**: Database migrations run automatically on startup in `NewPostgres()`
- Two writer interfaces:
  - `ChatResponseWriter`: For Twitch message storage
  - `DiscordWriter`: For Discord message storage

### Platform Integration Packages
- **`twitch/`**: Handles Twitch IRC connection, OAuth, and message handling
  - `connection.go`: IRC setup and lifecycle (`SetupTwitchIRC`, `ConnectIRC`)
  - `handleMessage.go`: Message parsing and routing to LLM
  - `auth.go`: OAuth token management for Twitch API
  - Hardcoded channel: `peteTwitchChannel = "soypetetech"`

- **`discord/`**: Handles Discord bot lifecycle and slash commands
  - `setup.go`: Discord session setup and command registration
  - `commands.go`: Command definitions (`AddCommands()`)
  - `ask.go`: `/askPedro` handler
  - `stump_pedro.go`: `/stumpPedro` handler for 20 questions

### Shared Components
- **`types/`**: Shared data structures
  - `twitch.go`: `TwitchMessage` struct
  - `discord.go`: Discord-specific types

- **`logging/`**: Structured logging wrapper
  - Supports log levels: debug, info, warn, error
  - Configured via `-errorLevel` CLI flag

- **`metrics/`**: Prometheus metrics server
  - Discord bot exposes metrics on port 6060
  - Twitch bot exposes metrics on port 6061
  - Prometheus server runs on separate host (100.125.196.1:9090)

### LLM Integration Pattern
Both bots use the OpenAI-compatible API pattern via langchain-go:
1. Set fake `OPENAI_API_KEY` environment variable (required by library)
2. Use `LLAMA_CPP_PATH` environment variable for the actual LLM endpoint
3. The setup functions automatically append `/v1` to the path if needed
4. Pass model name via `-model` CLI flag (must match model loaded in llama.cpp)

### Key Environment Variables
```bash
# Required for both bots
LLAMA_CPP_PATH=http://127.0.0.1:8080  # llama.cpp server endpoint
POSTGRES_URL=postgres://...            # Database connection string

# Discord bot
DISCORD_SECRET=...                     # Discord bot token

# Twitch bot
TWITCH_ID=...                          # Twitch OAuth client ID
TWITCH_SECRET=...                      # Twitch OAuth client secret
```

## Important Patterns and Conventions

### Error Handling
- Use `fmt.Errorf` with `%w` verb for error wrapping
- Log errors with structured fields before returning: `logger.Error("msg", "key", value)`
- Fatal errors in main functions should call `os.Exit(1)` after logging

### Database Migrations
- All migrations are embedded in the binary using `//go:embed migrations/*.sql`
- **Warning**: `database/connect-postgres.go` contains a `TODO: do not commit` with a `goose.DownTo()` call that should be removed before production merges
- Migrations run automatically on startup

### LLM Response Cleanup
- All LLM responses are cleaned via `ai.CleanResponse()` which:
  - Removes newlines (Twitch has strict message format requirements)
  - Strips instruction markers (`<|im_start|>`, `<|im_end|>`)
  - Removes leading `!` and `/` to prevent command injection

### Message Routing
- **Twitch**: The bot responds when:
  - Directly addressed by name
  - Specific trigger patterns match (implemented in `twitch/handleMessage.go`)
- **Discord**: The bot only responds to registered slash commands

### Prometheus Metrics
- Use `metrics.<MetricName>.Add(1)` to increment counters
- Key metrics: `TwitchConnectionCount`, `TwitchMessageRecievedCount`, `SuccessfulLLMGen`, `FailedLLMGen`, `EmptyLLMResponse`

## Testing
- Unit tests for core logic in `ai/helpers_test.go` and `ai/twitchchat/llm_test.go`
- Integration tests for message handling in `twitch/handleMessage_test.go`
- Use table-driven tests following Go conventions
