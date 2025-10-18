# KeepAlive Service

The KeepAlive service monitors the health of Discord bot, Twitch bot, and optionally VLLM services. It performs periodic health checks and sends alerts to a Discord channel when services go offline.

## Features

- **Periodic Health Checks**: Checks service health every minute (configurable)
- **Exponential Backoff**: Retries failed health checks with exponential backoff (1s, 2s, 4s)
- **Discord Alerts**: Sends alerts to #pedrogpt channel and tags @soypete
- **Smart Alerting**:
  - Alerts after 3 consecutive failed health checks
  - Repeats alerts once per hour if service remains offline
  - Sends recovery notification when service comes back online
- **State Tracking**: Tracks last check time and last alert time in memory

## Prerequisites

- Discord bot token with permission to send messages
- Discord channel ID for #pedrogpt
- Discord user ID for @soypete
- Running Discord and Twitch bots with /healthz endpoints on port 6060

## Configuration

The service is configured via command-line flags:

```bash
./keepalive \
  -discord-bot-url="http://localhost:6060/healthz" \
  -twitch-bot-url="http://localhost:6061/healthz" \
  -vllm-url="http://localhost:8080/health" \
  -discord-token="YOUR_DISCORD_TOKEN" \
  -channel-id="CHANNEL_ID" \
  -user-id="USER_ID" \
  -check-interval=60 \
  -alert-interval=3600 \
  -log-level=info
```

### Flags

- `discord-bot-url`: Discord bot health endpoint (default: http://localhost:6060/healthz)
- `twitch-bot-url`: Twitch bot health endpoint (optional, not set by default)
- `discord-token`: Discord bot token for sending alerts (required, uses same token as Discord bot)
- `check-interval`: Health check interval in seconds (default: 60)
- `alert-interval`: Alert repeat interval in seconds (default: 3600 = 1 hour)
- `log-level`: Logging level: debug, info, warn, error (default: info)

**Notes**:
- The Discord channel (#pedrogpt) and user tag (@soypete) are hardcoded
- VLLM/llama.cpp is automatically monitored if `LLAMA_CPP_PATH` environment variable is set
- Twitch bot monitoring is optional (set `-twitch-bot-url` to enable)

## Environment Variables

The service uses the following environment variables from `prod.env`:

```bash
DISCORD_SECRET           # Discord bot token (already exists, reused for alerts)
LLAMA_CPP_PATH           # llama.cpp/VLLM server URL (already exists, auto-monitored if set)
DISCORD_BOT_URL          # Discord bot health endpoint (optional, default: http://discord-bot:6060/healthz)
TWITCH_BOT_URL           # Twitch bot health endpoint (optional, not monitored by default)
CHECK_INTERVAL           # Health check interval in seconds (optional, default: 60)
ALERT_INTERVAL           # Alert repeat interval in seconds (optional, default: 3600)
LOG_LEVEL                # Logging level (optional, default: info)
```

**No new secrets required!** The service reuses the existing `DISCORD_SECRET` from your Discord bot configuration.

These are loaded via 1Password CLI in production using `op run --env-file prod.env`.

## Building

### Local Build

```bash
go build ./cli/keepalive/
```

### Docker Build

```bash
docker build -f cli/keepalive/keepaliveBot.Dockerfile -t pedro-keepalive .
```

## Running

### Local

```bash
./keepalive \
  -discord-token="YOUR_DISCORD_SECRET"
```

The service will automatically find the #pedrogpt channel and tag @soypete in alerts.

### Docker (Production)

The Docker container uses 1Password CLI to load secrets from `prod.env`:

```bash
docker run pedro-keepalive
```

All configuration is loaded from `prod.env` via 1Password. The default values are:
- Discord/Twitch bot URLs: `http://discord-bot:6060/healthz` and `http://twitch-bot:6060/healthz`
- Check interval: 60 seconds
- Alert interval: 3600 seconds (1 hour)
- Log level: info

You can override defaults by setting environment variables in prod.env:
```bash
DISCORD_BOT_URL="http://custom-discord:6060/healthz"
TWITCH_BOT_URL="http://custom-twitch:6060/healthz"
VLLM_URL="http://vllm:8080/health"
CHECK_INTERVAL="120"
ALERT_INTERVAL="7200"
LOG_LEVEL="debug"
```

## Alert Behavior

### Initial Failure
- Service fails 1-2 times: Only logs warnings, no alerts
- Service fails 3 times consecutively: Sends first alert to Discord

### Ongoing Failure
- Continues checking every minute
- Sends repeat alerts once per hour while service remains down

### Recovery
- When service comes back online, sends a recovery notification

### Alert Format

```
@soypete **Alert:** Service Discord Bot is offline after 3 failed health checks
@soypete **Alert:** Service Discord Bot is still offline (consecutive failures: 15)
@soypete **Alert:** Service Discord Bot has recovered after 15 failed checks
```

## Architecture

### Components

- `service.go`: Core keepalive service with health check logic
- `discord_alerter.go`: Discord integration for sending alerts
- `cli/keepalive/main.go`: CLI entry point

### Health Check Flow

1. Every minute, check all configured services
2. For each service, perform HTTP GET to /healthz endpoint
3. If health check fails, retry with exponential backoff (3 attempts max)
4. Track consecutive failures in service state
5. Send alert after 3 failures or every hour if still down
6. Send recovery alert when service becomes healthy again

## Monitoring

The keepalive service itself logs all health checks and alerts using structured JSON logging. Monitor the logs to ensure the service is running properly.

```bash
# View logs
docker logs pedro-keepalive

# Follow logs
docker logs -f pedro-keepalive
```

## Troubleshooting

### No alerts being sent

1. Verify Discord token has proper permissions
2. Check channel ID is correct
3. Ensure bot is a member of the channel
4. Check logs for error messages

### False positives

1. Adjust `check-interval` to allow more time between checks
2. Verify network connectivity between services
3. Check if services are actually healthy via manual curl

### Missing health checks

1. Ensure Discord and Twitch bots are running
2. Verify /healthz endpoints are accessible
3. Check firewall rules allow connections
4. Test health endpoints manually: `curl http://localhost:6060/healthz`
