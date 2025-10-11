#!/bin/bash

set -e

echo "=== Starting Pedro Local Development Environment ==="
echo ""

# Check if Ollama is running
if ! curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    echo "❌ Ollama is not running!"
    echo ""
    echo "Please start Ollama in another terminal:"
    echo "  ollama serve"
    echo ""
    exit 1
fi

echo "✅ Ollama is running"
echo ""

# Check which service to run
SERVICE=${1:-discord}

case "$SERVICE" in
    discord)
        echo "Starting Discord bot with PostgreSQL..."
        docker compose up -d postgres pedro-discord
        ;;
    twitch)
        echo "Starting Twitch bot with PostgreSQL..."
        docker compose --profile twitch up -d postgres pedro-twitch
        ;;
    both)
        echo "Starting both Discord and Twitch bots with PostgreSQL..."
        docker compose --profile twitch up -d
        ;;
    *)
        echo "Usage: $0 [discord|twitch|both]"
        echo ""
        echo "Examples:"
        echo "  ./run.sh discord  # Start Discord bot only (default)"
        echo "  ./run.sh twitch   # Start Twitch bot only"
        echo "  ./run.sh both     # Start both bots"
        exit 1
        ;;
esac

echo ""
echo "Waiting for services to start..."
sleep 3

echo ""
echo "=== Services Status ==="
docker compose ps

echo ""
echo "=== View Logs ==="
echo "  All services:     docker compose logs -f"
echo "  Discord only:     docker compose logs -f pedro-discord"
echo "  Twitch only:      docker compose logs -f pedro-twitch"
echo "  Database only:    docker compose logs -f postgres"
echo ""
echo "=== Metrics Endpoints ==="
echo "  Discord:          http://localhost:6060/metrics"
if [[ "$SERVICE" == "twitch" || "$SERVICE" == "both" ]]; then
    echo "  Twitch:           http://localhost:6061/metrics"
    echo "  Twitch OAuth:     http://localhost:3000/oauth/redirect"
fi
echo ""
echo "=== Ollama ==="
echo "  API:              http://localhost:11434"
echo "  Models:           ollama list"
echo "  Pull model:       ollama pull qwen2.5-coder:7b-instruct"
echo ""
echo "=== Stop Services ==="
echo "  Run: ./stop.sh"
