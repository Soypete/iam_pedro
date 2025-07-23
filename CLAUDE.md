# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

IAM_PEDRO is a Twitch/Discord chatbot built in Go that uses llama.cpp for LLM inference, langchain-go for AI orchestration, and PostgreSQL with vector embeddings for chat history storage. The bot provides AI-powered responses to chat messages and supports both Twitch IRC and Discord slash commands.

## Architecture

The codebase is organized into several key modules:

- **CLI Applications**: Two main entry points in `cli/discord/` and `cli/twitch/` that run separate bot instances
- **AI Module** (`ai/`): Contains LLM integration logic with separate handlers for Discord (`discordchat/`) and Twitch (`twitchchat/`)
- **Database** (`database/`): PostgreSQL connection, migrations, and data models for chat history and bot responses
- **Platform Integrations**: 
  - `discord/`: Discord bot setup, slash commands (/askPedro, /stumpPedro, /helpPedro), and message handling
  - `twitch/`: Twitch IRC client, authentication, and chat message processing
- **Infrastructure**: 
  - `logging/`: Structured logging with configurable levels
  - `metrics/`: Prometheus metrics server (runs on port 6060)
  - `charts/`: Helm charts for Kubernetes deployment

## Development Commands

### Building
```bash
# Build Discord bot
go build -v -o pedro ./cli/discord

# Build Twitch bot  
go build -v -o pedro ./cli/twitch
```

### Testing
```bash
# Run all tests with coverage
go test ./... -v -cover -covermode=atomic

# Run tests for specific package
go test ./ai -v
go test ./twitch -v
```

### Code Quality
The project uses golangci-lint for linting. CI runs with `only-new-issues: true` and `skip-cache: true`.

### Running Locally
Both bots require these environment variables:
- `LLAMA_CPP_PATH`: URL to llama.cpp server (typically http://127.0.0.1:8080)
- `POSTGRES_URL`: PostgreSQL connection string
- `POSTGRES_VECTOR_URL`: PostgreSQL vector database connection
- `TWITCH_ID` and `TWITCH_SECRET`: Twitch API credentials
- `DISCORD_TOKEN`: Discord bot token

Command-line flags:
- `-model`: Specify LLM model (e.g., "meta-llama3.1")  
- `-errorLevel`: Set log level (debug, info, warn, error)

## Key Implementation Details

### LLM Integration
- Uses OpenAI-compatible API to connect to local llama.cpp server
- Sets `OPENAI_API_KEY=test` as placeholder since using local inference
- Response timeout can be up to 5 minutes based on model performance

### Database Schema
- Uses Goose migrations in `database/migrations/`
- Stores chat messages with embeddings for context-aware responses
- Tracks bot responses and prompt/chat relationships

### Deployment
- Dockerized with separate containers for Discord and Twitch bots
- Helm charts support independent scaling of each bot type  
- Kubernetes deployment uses image tags from GitHub Container Registry (`ghcr.io/soypete/iam_pedro`)

### Chat Features
- Records all chat in vector database with embeddings
- Provides context-aware responses based on chat history
- Supports 20 questions game mode via Discord threads
- Maintains helpful links table for stream-relevant information