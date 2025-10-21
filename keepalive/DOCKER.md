# KeepAlive Docker Deployment

## Quick Start

### Build
```bash
docker build -f cli/keepalive/keepaliveBot.Dockerfile -t pedro-keepalive .
```

### Run
```bash
docker run pedro-keepalive
```

The container automatically loads configuration from `prod.env` using 1Password CLI.

## Configuration

### Required 1Password Secrets

**No new secrets needed!** The keepalive service reuses the existing `DISCORD_SECRET` from your Discord bot configuration.

The channel (#pedrogpt) and user (@soypete) are hardcoded in the service.

### Optional Environment Variables

You can add these to `prod.env` to override defaults:

```bash
# Service URLs (defaults shown)
DISCORD_BOT_URL="http://discord-bot:6060/healthz"  # Always monitored
TWITCH_BOT_URL=""                                    # Optional, not monitored by default
LLAMA_CPP_PATH="http://vllm:8080"                   # Auto-monitored if set (adds /health)

# Timing configuration
CHECK_INTERVAL="60"      # Health check every 60 seconds
ALERT_INTERVAL="3600"    # Repeat alerts every 3600 seconds (1 hour)

# Logging
LOG_LEVEL="info"         # debug, info, warn, error
```

## Docker Compose Example

```yaml
version: '3.8'

services:
  discord-bot:
    image: pedro-discord
    ports:
      - "6060:6060"
    # ... other config

  twitch-bot:
    image: pedro-twitch
    ports:
      - "6061:6060"
    # ... other config

  keepalive:
    image: pedro-keepalive
    depends_on:
      - discord-bot
      - twitch-bot
    environment:
      - DISCORD_BOT_URL=http://discord-bot:6060/healthz
      - TWITCH_BOT_URL=http://twitch-bot:6060/healthz
    volumes:
      - ./prod.env:/app/prod.env:ro
```

## Kubernetes Example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: keepalive
spec:
  replicas: 1
  selector:
    matchLabels:
      app: keepalive
  template:
    metadata:
      labels:
        app: keepalive
    spec:
      containers:
      - name: keepalive
        image: pedro-keepalive
        env:
        - name: DISCORD_BOT_URL
          value: "http://discord-bot:6060/healthz"
        - name: TWITCH_BOT_URL
          value: "http://twitch-bot:6060/healthz"
        - name: CHECK_INTERVAL
          value: "60"
        - name: ALERT_INTERVAL
          value: "3600"
        volumeMounts:
        - name: config
          mountPath: /app/prod.env
          subPath: prod.env
          readOnly: true
      volumes:
      - name: config
        secret:
          secretName: pedro-config
```

## Networking Notes

- The keepalive service needs network access to the Discord and Twitch bots
- Default service names in Docker Compose: `discord-bot`, `twitch-bot`
- Health endpoints must be accessible on port 6060
- Outbound HTTPS access required for Discord webhook alerts

## Monitoring the Monitor

Check keepalive logs to ensure it's running:

```bash
# Docker
docker logs -f pedro-keepalive

# Docker Compose
docker-compose logs -f keepalive

# Kubernetes
kubectl logs -f deployment/keepalive
```

Expected log output:
```
{"level":"info","msg":"Starting KeepAlive service","check_interval":60,"alert_interval":3600,"monitored_services":2}
{"level":"debug","msg":"health check request succeeded","url":"http://discord-bot:6060/healthz"}
{"level":"debug","msg":"health check request succeeded","url":"http://twitch-bot:6060/healthz"}
```

## Troubleshooting

### Container won't start
```bash
# Check if prod.env is accessible
docker run --rm pedro-keepalive ls -la /app/prod.env

# Test 1Password CLI
docker run --rm pedro-keepalive op --version
```

### Health checks failing
```bash
# Test from within the keepalive container
docker exec -it <container-id> wget -O- http://discord-bot:6060/healthz
docker exec -it <container-id> wget -O- http://twitch-bot:6060/healthz
```

### Alerts not sending
```bash
# Check Discord token is loaded
docker exec -it <container-id> env | grep DISCORD_SECRET

# Check logs for channel discovery
docker logs <container-id> | grep "Discord alerter initialized"
```
