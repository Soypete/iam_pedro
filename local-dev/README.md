# Pedro Local Development Guide

Run Pedro bots locally with Ollama for testing and development without needing access to production GPU resources.

## Architecture

Local development stack:
- **Ollama**: Runs on host at `http://localhost:11434` (provides OpenAI-compatible API)
- **PostgreSQL**: Docker container for database
- **Pedro Bots**: Docker containers connecting to Ollama via `host.docker.internal`

## Prerequisites

### Required Software

1. **Ollama** (runs models locally)
   ```bash
   # macOS
   brew install ollama

   # Linux
   curl -fsSL https://ollama.com/install.sh | sh

   # Or download from: https://ollama.com/download
   ```

2. **Docker Desktop** (for containers)
   - macOS/Windows: https://www.docker.com/products/docker-desktop
   - Linux: Install Docker Engine + Docker Compose

3. **1Password CLI** (optional - for secrets management)
   ```bash
   # macOS
   brew install --cask 1password-cli

   # Or download from: https://developer.1password.com/docs/cli/get-started
   ```

### System Requirements

Choose a model based on your available RAM:

| Model | RAM Required | Speed | Quality | Recommended For |
|-------|--------------|-------|---------|-----------------|
| `qwen2.5-coder:3b-instruct` | 4-6 GB | Fast | Good | Quick testing, low-RAM systems |
| `qwen2.5-coder:7b-instruct` | 8-12 GB | Medium | Better | **Default, balanced** |
| `qwen2.5-coder:14b-instruct` | 16-24 GB | Slow | Best | High-end systems, production-like |

**Recommended**: 16GB+ RAM for comfortable development with 7B model

## Quick Start

### 1. Initial Setup

```bash
cd local-dev
./setup.sh
```

This will:
- ✅ Check prerequisites (Ollama, Docker)
- ✅ Pull the Qwen model (default: 7b-instruct)
- ✅ Create configuration templates

### 2. Configure Secrets

You have two options:

#### Option A: Using 1Password (Recommended)

```bash
# Set your 1Password service account token
export OP_SERVICE_ACCOUNT_TOKEN="ops_your_token_here"

# Edit .env
nano .env
# Set: OP_SERVICE_ACCOUNT_TOKEN=ops_your_token_here

# Use 1Password references in prod.env
cp prod.env.template ../prod.env
nano ../prod.env
# Keep the op:// references, they'll be injected at runtime
```

#### Option B: Using Raw Environment Variables

```bash
# Edit prod.env with actual values
cp prod.env.template ../prod.env
nano ../prod.env

# Replace op:// references with actual tokens:
# DISCORD_TOKEN=your_actual_discord_token
# TWITCH_SECRET=your_actual_twitch_secret
# TWITCH_TOKEN=your_actual_twitch_token

# Edit .env (leave OP_SERVICE_ACCOUNT_TOKEN empty)
nano .env
```

### 3. Start Ollama

In a **separate terminal**, run:

```bash
ollama serve
```

Keep this running while you develop.

### 4. Run the Bots

```bash
# Start Discord bot only (default)
./run.sh discord

# Start Twitch bot only
./run.sh twitch

# Start both bots
./run.sh both
```

### 5. View Logs

```bash
# All services
docker compose logs -f

# Discord bot only
docker compose logs -f pedro-discord

# Twitch bot only
docker compose logs -f pedro-twitch

# Database only
docker compose logs -f postgres
```

### 6. Stop Everything

```bash
./stop.sh
```

## Development Workflow

### Testing Changes

1. **Edit code** in your editor
2. **Rebuild containers**:
   ```bash
   docker compose build pedro-discord
   # or
   docker compose build pedro-twitch
   ```
3. **Restart services**:
   ```bash
   docker compose restart pedro-discord
   # or use ./run.sh again
   ```
4. **View logs**:
   ```bash
   docker compose logs -f pedro-discord
   ```

### Switching Models

```bash
# Pull a different model
ollama pull qwen2.5-coder:3b-instruct

# Edit docker-compose.yml and change the -model flag
nano docker-compose.yml
# Change: -model qwen2.5-coder:7b-instruct
# To:     -model qwen2.5-coder:3b-instruct

# Restart
docker compose restart pedro-discord
```

### Using Different Models

List available models:
```bash
ollama list
```

Pull additional models:
```bash
# Smaller model (faster, less RAM)
ollama pull qwen2.5-coder:3b-instruct

# Larger model (slower, more accurate)
ollama pull qwen2.5-coder:14b-instruct

# Other models
ollama pull llama3.2:3b
ollama pull codellama:7b
```

## Endpoints

When running locally:

| Service | Endpoint | Description |
|---------|----------|-------------|
| Ollama API | http://localhost:11434 | Model inference |
| Discord Metrics | http://localhost:6060/metrics | Prometheus metrics |
| Twitch Metrics | http://localhost:6061/metrics | Prometheus metrics |
| Twitch OAuth | http://localhost:3000/oauth/redirect | OAuth callback |
| PostgreSQL | localhost:5432 | Database (user: pedro, db: pedro_dev) |

## Troubleshooting

### Ollama Connection Issues

**Error**: "failed to connect to Ollama"

```bash
# Check if Ollama is running
curl http://localhost:11434/api/tags

# If not, start it
ollama serve

# Check if model is pulled
ollama list
```

### Docker Can't Connect to Ollama

**Error**: "connection refused" to host.docker.internal

**Solution**: Make sure Ollama is running on the host, not in Docker:
```bash
# This should work OUTSIDE of Docker
curl http://localhost:11434/api/tags

# Docker should be able to reach it via host.docker.internal
docker run --rm curlimages/curl:latest curl http://host.docker.internal:11434/api/tags
```

### Database Connection Issues

**Error**: "failed to connect to postgres"

```bash
# Check if postgres is healthy
docker compose ps

# Restart postgres
docker compose restart postgres

# Check logs
docker compose logs postgres
```

### Out of Memory

**Error**: Container killed or system freezing

**Solution**: Use a smaller model
```bash
ollama pull qwen2.5-coder:3b-instruct
# Update docker-compose.yml to use this model
```

### 1Password Issues

**Error**: "op: command not found" or authentication failures

**Solutions**:
1. **If using 1Password**: Make sure `OP_SERVICE_ACCOUNT_TOKEN` is set in `.env`
2. **If NOT using 1Password**: Use raw env vars in `prod.env` instead of `op://` references

## Database Management

### Connect to Database

```bash
# Using psql
docker compose exec postgres psql -U pedro -d pedro_dev

# Using any PostgreSQL client
# Host: localhost
# Port: 5432
# User: pedro
# Password: pedro_local_dev
# Database: pedro_dev
```

### Reset Database

```bash
# Stop and remove volumes
docker compose down -v

# Start fresh
./run.sh
```

### Backup/Restore

```bash
# Backup
docker compose exec postgres pg_dump -U pedro pedro_dev > backup.sql

# Restore
docker compose exec -T postgres psql -U pedro -d pedro_dev < backup.sql
```

## Differences from Production

| Aspect | Local Dev | Production |
|--------|-----------|------------|
| LLM Service | Ollama (localhost) | vLLM (pedro-gpu.tail6fbc5.ts.net) |
| Model | qwen2.5-coder:7b | Qwen/Qwen2.5-Coder-14B-Instruct-AWQ |
| Database | Local PostgreSQL | Production PostgreSQL |
| Secrets | 1Password or raw env | 1Password service account |
| Restart Policy | Manual (dev mode) | Auto-restart (unless-stopped) |

## Tips

1. **Keep Ollama running**: Start `ollama serve` once and leave it running
2. **Use smaller models**: 3B or 7B models are fine for most testing
3. **Watch logs**: Use `docker compose logs -f` to see what's happening
4. **Rebuild after code changes**: Run `docker compose build` after modifying code
5. **Clean up**: Run `docker compose down -v` to remove everything and start fresh

## Next Steps

Once local development is working:
1. Test your changes locally
2. Commit to your branch
3. Deploy to production using `deployment/` scripts
4. Production uses vLLM with larger models for better quality

## Additional Resources

- Ollama: https://ollama.com/library/qwen2.5-coder
- Docker Compose: https://docs.docker.com/compose/
- 1Password CLI: https://developer.1password.com/docs/cli/
