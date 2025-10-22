#!/bin/bash

set -e

# Simple Docker-based deployment script for Pedro bots
# No systemd required - uses Docker's built-in restart policies

SERVICE=${1:-"discord"}  # discord, twitch, or both
TAG=${2:-$(git rev-parse --short HEAD 2>/dev/null || echo "latest")}

echo "=== Pedro Bot Deployment ==="
echo "Service: $SERVICE"
echo "Tag: $TAG"
echo ""

# Configuration file paths
SERVICE_ENV="/opt/pedro/service.env"
PROD_ENV="/opt/pedro/prod.env"

# Check prerequisites
if ! command -v docker &> /dev/null; then
    echo "ERROR: Docker/Podman not found. Please install docker or podman."
    exit 1
fi

# Ensure config directory exists
sudo mkdir -p /opt/pedro

# Check for service.env
if [ ! -f "$SERVICE_ENV" ]; then
    echo "Creating template $SERVICE_ENV..."
    sudo tee "$SERVICE_ENV" > /dev/null <<'EOF'
# 1Password Service Account Token (required for op CLI in container)
OP_SERVICE_ACCOUNT_TOKEN=ops_YOUR_ACTUAL_TOKEN_HERE

# Twitch Client ID (not a secret, safe to store in plain text)
TWITCH_ID=your_twitch_client_id

# OAuth redirect host for remote authentication
OAUTH_REDIRECT_HOST=100.81.89.62:3000

# LLM Service URL (not a secret, can be plain text)
LLAMA_CPP_PATH=https://pedro-gpu.tail6fbc5.ts.net

# LLM Model name - should match the model name in your vLLM/llama.cpp server
MODEL=your_model_name_here
EOF
    echo "ERROR: Please edit $SERVICE_ENV with your actual values!"
    exit 1
fi

# Check for prod.env
if [ ! -f "$PROD_ENV" ]; then
    echo "Creating template $PROD_ENV..."
    sudo tee "$PROD_ENV" > /dev/null <<'EOF'
# Discord Bot
DISCORD_TOKEN=op://vault/discord-bot/token

# Twitch Bot - client secret for OAuth
TWITCH_SECRET=op://vault/twitch-bot/client-secret

# Twitch Bot - OPTIONAL: uncomment after first OAuth to skip flow
# TWITCH_TOKEN=op://vault/twitch-bot/access-token

# Database
DATABASE_URL=op://vault/postgres/connection-url
EOF
    echo "ERROR: Please edit $PROD_ENV with your 1Password references!"
    exit 1
fi

# Load service environment
source "$SERVICE_ENV"

# Function to deploy Discord
deploy_discord() {
    echo "=== Deploying Discord Bot ==="

    # Stop and remove existing container
    docker stop pedro-discord 2>/dev/null || true
    docker rm pedro-discord 2>/dev/null || true

    # Build image
    echo "Building Discord container..."
    docker build -f cli/discord/discordBot.Dockerfile -t localhost/pedro-discord:$TAG .

    # Run container
    echo "Starting Discord container..."
    docker run -d \
        --name pedro-discord \
        --restart unless-stopped \
        -p 6060:6060 \
        -v "$PROD_ENV:/app/prod.env:ro" \
        -e OP_SERVICE_ACCOUNT_TOKEN="$OP_SERVICE_ACCOUNT_TOKEN" \
        -e TWITCH_ID="$TWITCH_ID" \
        -e LLAMA_CPP_PATH="${LLAMA_CPP_PATH:-https://pedro-gpu.tail6fbc5.ts.net}" \
        -e MODEL="$MODEL" \
        localhost/pedro-discord:$TAG

    echo "✅ Discord bot deployed!"
    echo "   Container: pedro-discord"
    echo "   Metrics: http://localhost:6060/metrics"
    echo "   Logs: docker logs -f pedro-discord"
}

# Function to deploy Twitch
deploy_twitch() {
    echo "=== Deploying Twitch Bot ==="

    # Stop and remove existing container
    docker stop pedro-twitch 2>/dev/null || true
    docker rm pedro-twitch 2>/dev/null || true

    # Build image
    echo "Building Twitch container..."
    docker build -f cli/twitch/twitchBot.Dockerfile -t localhost/pedro-twitch:$TAG .

    # Run container
    echo "Starting Twitch container..."
    docker run -d \
        --name pedro-twitch \
        --restart unless-stopped \
        -p 6061:6060 \
        -p 3000:3000 \
        -v "$PROD_ENV:/app/prod.env:ro" \
        -e OP_SERVICE_ACCOUNT_TOKEN="$OP_SERVICE_ACCOUNT_TOKEN" \
        -e TWITCH_ID="$TWITCH_ID" \
        -e OAUTH_REDIRECT_HOST="$OAUTH_REDIRECT_HOST" \
        -e LLAMA_CPP_PATH="${LLAMA_CPP_PATH:-https://pedro-gpu.tail6fbc5.ts.net}" \
        -e MODEL="$MODEL" \
        localhost/pedro-twitch:$TAG

    echo "✅ Twitch bot deployed!"
    echo "   Container: pedro-twitch"
    echo "   Metrics: http://localhost:6061/metrics"
    echo "   OAuth: http://$OAUTH_REDIRECT_HOST/oauth/redirect"
    echo "   Logs: docker logs -f pedro-twitch"
    echo ""
    echo "   NOTE: Watch logs for OAuth URL on first run:"
    echo "   docker logs -f pedro-twitch"
}

# Deploy based on service selection
case "$SERVICE" in
    discord)
        deploy_discord
        ;;
    twitch)
        deploy_twitch
        ;;
    both)
        deploy_discord
        echo ""
        deploy_twitch
        ;;
    *)
        echo "ERROR: Invalid service '$SERVICE'"
        echo "Usage: $0 [discord|twitch|both] [tag]"
        exit 1
        ;;
esac

echo ""
echo "=== Deployment Complete ==="
echo ""
echo "Manage containers:"
echo "  docker ps                    # List running containers"
echo "  docker logs -f <name>        # View logs"
echo "  docker stop <name>           # Stop container"
echo "  docker start <name>          # Start container"
echo "  docker restart <name>        # Restart container"
echo ""
echo "Containers will automatically restart on failure and system reboot."
