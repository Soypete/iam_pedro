#!/bin/bash

set -e

# Remote build and deploy script for target host
TAG=${1:-$(git rev-parse --short HEAD 2>/dev/null || echo "latest")}
SERVICE=${2:-"discord"}  # discord, twitch, or both

echo "Building and deploying Pedro containers on remote host"
echo "Tag: $TAG"
echo "Service: $SERVICE"

# Ensure we have Docker/Podman
if ! command -v docker &> /dev/null && ! command -v podman &> /dev/null; then
    echo "Error: Neither docker nor podman found. Please install Docker."
    exit 1
fi

# Use docker or podman
CONTAINER_CMD="docker"
if command -v podman &> /dev/null && ! command -v docker &> /dev/null; then
    CONTAINER_CMD="podman"
fi

echo "Using container runtime: $CONTAINER_CMD"

# Build Discord container
if [[ "$SERVICE" == "discord" || "$SERVICE" == "both" ]]; then
    echo "Building Discord container..."
    $CONTAINER_CMD build -f cli/discord/discordBot.Dockerfile -t pedro-discord:$TAG .
    
    echo "Discord container built successfully!"
fi

# Build Twitch container  
if [[ "$SERVICE" == "twitch" || "$SERVICE" == "both" ]]; then
    echo "Building Twitch container..."
    $CONTAINER_CMD build -f cli/twitch/twitchBot.Dockerfile -t pedro-twitch:$TAG .
    
    echo "Twitch container built successfully!"
fi

# Ensure directories exist
sudo mkdir -p /opt/pedro

# Check for environment file
if [ ! -f /opt/pedro/prod.env ]; then
    echo "Creating template environment file at /opt/pedro/prod.env"
    sudo tee /opt/pedro/prod.env > /dev/null <<EOF
# Pedro Bot Environment Configuration
# Copy this file and fill in your actual values

# Discord Configuration
DISCORD_TOKEN=your_discord_token_here

# Twitch Configuration  
TWITCH_TOKEN=your_twitch_token_here
TWITCH_CHANNEL=your_twitch_channel_here

# Database Configuration
DATABASE_URL=your_postgres_database_url_here

# LLM Configuration
OPENAI_API_KEY=your_openai_api_key_here

# 1Password Configuration (if using)
OP_CONNECT_HOST=your_1password_connect_host
OP_CONNECT_TOKEN=your_1password_connect_token
EOF
    echo "WARNING: Please edit /opt/pedro/prod.env with your actual configuration values!"
    echo "The deployment will fail without proper environment variables."
fi

# Deploy Discord service
if [[ "$SERVICE" == "discord" || "$SERVICE" == "both" ]]; then
    echo "Deploying Discord service..."
    chmod +x deployment/deploy-discord.sh
    ./deployment/deploy-discord.sh $TAG pedro-discord
fi

# Deploy Twitch service
if [[ "$SERVICE" == "twitch" || "$SERVICE" == "both" ]]; then
    echo "Deploying Twitch service..."
    chmod +x deployment/deploy-twitch.sh
    ./deployment/deploy-twitch.sh $TAG pedro-twitch
fi

echo ""
echo "Deployment complete!"
echo ""
echo "Next steps:"
echo "1. Edit /opt/pedro/prod.env with your actual configuration"
echo "2. Restart services: sudo systemctl restart pedro-discord pedro-twitch"
echo ""
echo "Monitor services:"
if [[ "$SERVICE" == "discord" || "$SERVICE" == "both" ]]; then
    echo "  Discord: sudo systemctl status pedro-discord"
    echo "  Discord metrics: http://localhost:6060/metrics"
fi
if [[ "$SERVICE" == "twitch" || "$SERVICE" == "both" ]]; then
    echo "  Twitch: sudo systemctl status pedro-twitch"  
    echo "  Twitch metrics: http://localhost:6061/metrics"
fi
echo ""
echo "View logs:"
if [[ "$SERVICE" == "discord" || "$SERVICE" == "both" ]]; then
    echo "  sudo journalctl -u pedro-discord -f"
fi
if [[ "$SERVICE" == "twitch" || "$SERVICE" == "both" ]]; then
    echo "  sudo journalctl -u pedro-twitch -f"
fi