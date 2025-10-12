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
go build ./cli/twitch/    # Twitch bot
go build ./cli/discord/   # Discord bot
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

## Architecture

### Core Components

**AI Layer (`/ai/`)**
- `chatter.go` - Core interface defining `Chatter` contract for all bot implementations
- `twitchchat/` - Twitch-specific LLM client with async web search integration
- `discordchat/` - Discord-specific LLM client with 20 questions game support
- Pedro's personality is defined in `ai.PedroPrompt` constant

**Platform Integration**
- `/discord/` - Discord bot with slash commands (`/askPedro`, `/stumpPedro`, `/helpPedro`)
- `/twitch/` - Twitch IRC client with OAuth authentication and message handling
- `/types/` - Shared data structures including `TwitchMessage` and `WebSearchRequest`

**External Services**
- `/duckduckgo/` - Web search client for real-time information retrieval
- LLM integration via langchain-go connecting to llama.cpp (not OpenAI)

**Infrastructure**
- `/database/` - PostgreSQL with embedded migrations, supports vector embeddings
- `/logging/` - Structured JSON logging with slog
- `/metrics/` - Prometheus metrics + expvar + pprof on port 6060

### Data Flow

1. **Message Reception**: Discord slash commands or Twitch chat messages
2. **AI Processing**: Messages go through platform-specific LLM clients
3. **Web Search Integration**: If Pedro responds with "execute web search", triggers async DuckDuckGo lookup
4. **Response Handling**: Immediate response + async follow-up with search results
5. **Persistence**: All interactions stored in PostgreSQL with chat history management

### Key Environment Variables

```bash
LLAMA_CPP_PATH="http://127.0.0.1:8080"  # llama.cpp server endpoint
POSTGRES_URL=""                          # Main database connection
POSTGRES_VECTOR_URL=""                   # Vector database for embeddings
TWITCH_ID=""                            # Twitch OAuth client ID
TWITCH_SECRET=""                        # Twitch OAuth client secret
DISCORD_TOKEN=""                        # Discord bot token
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
- Context window management: chat history limited to 10 messages
- Temperature: 0.7, Max length: 500 characters

### Platform-Specific Features
- **Twitch**: Real-time chat integration, responds to mentions of "pedro", "Pedro", "llm", "LLM", "bot"
- **Discord**: Thread-based conversations, 20 questions game, slash command interface

### Pedro's Personality
- Assistant for SoyPeteTech (Miriah Peterson), a software streamer in Utah
- Focuses on Golang and Data/AI content
- Uses custom emotes: soypet2Thinking, soypet2Dance, soypet2ConfusedPedro, etc.
- Explicitly avoids Java and JavaScript discussions
- Response limit: 500 characters, no newlines

### Metrics and Monitoring
- Prometheus metrics on port 6060
- Tracks message counts, LLM generation success/failure, response times
- Structured logging with trace IDs for request correlation

## Future Development (From README)

The project is evolving toward structured agentic planning with:
- Step-based reasoning workflows
- DAG-style orchestration
- Prescribed planning actions: `api_call`, `db_query`, `web_search`, `reference_check`, `return_response`
- Live workflow feedback in Discord threads