# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is "IAM_PEDRO" - a Go-based Twitch chat bot that integrates with Discord, uses LLM responses via llama.cpp, and stores chat history in PostgreSQL with vector embeddings. The bot (Pedro) responds to chat messages, provides helpful information, and supports Discord slash commands.

## Common Development Commands

### Building and Running
```bash
# Build the project
go build ./...

# Run tests
go test ./...

# Run specific tests
go test ./ai/helpers_test.go
go test ./twitch/handleMessage_test.go

# Build and run Twitch bot
go run ./cli/twitch/twitch.go -model <model_name> -errorLevel <debug|info|warn|error>

# Build and run Discord bot  
go run ./cli/discord/discord.go

# Check code formatting
go fmt ./...

# Vet code for issues
go vet ./...

# Tidy dependencies
go mod tidy
```

### Docker Operations
```bash
# Build Twitch bot container
docker build -f cli/twitch/twitchBot.Dockerfile -t twitch-llm-bot .

# Build Discord bot container  
docker build -f cli/discord/discordBot.Dockerfile -t discord-llm-bot .

# Run with required environment variables
docker run -e LLAMA_CPP_PATH="" -e POSTGRES_URL="" -e TWITCH_ID="" -e TWITCH_SECRET="" -e POSTGRES_VECTOR_URL="" <image>
```

## Architecture

### Core Components

**AI Layer (`ai/`)**
- `chatter.go`: Defines the `Chatter` interface for bot responses
- `twitchchat/`: Twitch-specific LLM integration using langchain-go
- `discordchat/`: Discord-specific AI handlers (20 questions, ask commands)
- `helpers.go`: Shared AI utilities

**Database Layer (`database/`)**
- `connect-postgres.go`: PostgreSQL connection and migration management using goose
- `message.go`: Core message storage and retrieval
- `responses.go`: Bot response tracking
- `discord_messages.go`: Discord-specific message handling
- `migrations/`: SQL schema migrations

**Platform Integrations**
- `twitch/`: Twitch IRC client, authentication, and message handling
- `discord/`: Discord bot commands and setup
- `cli/`: Entry points for both Twitch and Discord bots

**Infrastructure**
- `metrics/`: Prometheus metrics server on port 6060 with expvar integration
- `logging/`: Structured logging throughout the application

### Key Dependencies
- `github.com/tmc/langchaingo`: LLM integration
- `github.com/gempir/go-twitch-irc/v2`: Twitch chat client
- `github.com/bwmarrin/discordgo`: Discord API client
- `github.com/jmoiron/sqlx` + `github.com/lib/pq`: PostgreSQL integration
- `github.com/pressly/goose/v3`: Database migrations
- `github.com/prometheus/client_golang`: Metrics collection

### Configuration
- Uses environment variables for configuration (no config files)
- Required env vars: `POSTGRES_URL`, `TWITCH_ID`, `TWITCH_SECRET`, `LLAMA_CPP_PATH`, `POSTGRES_VECTOR_URL`
- LLM integration expects llama.cpp server running on `127.0.0.1:8080`

### Bot Behavior
- Pedro responds to direct mentions and specific prompts
- Uses approved Twitch emotes: `soypet2Thinking`, `soypet2Dance`, `soypet2ConfusedPedro`, etc.
- 500 character response limit, no newlines allowed
- Stores all chat in vector database for context-aware responses
- Discord commands: `/askPedro`, `/stumpPedro` (20 questions), `/helpPedro`

### Testing
- Unit tests exist for `ai/helpers_test.go` and `twitch/handleMessage_test.go`
- Tests can be run individually or all together with `go test`

### Development Notes
- The codebase is actively developed with TODOs for features like embeddings, stream title integration, and configuration management
- Database migrations auto-run on startup
- Metrics available at `:6060/metrics` for monitoring
- Both Twitch and Discord bots can run simultaneously but are separate entry points