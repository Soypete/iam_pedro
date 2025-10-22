#!/bin/bash

# Simple test script to run Pedro bots locally with Docker
# Usage: ./test-local.sh [discord|twitch]

set -e

SERVICE=${1:-"discord"}

echo "=== Testing Pedro $SERVICE Bot Locally ==="
echo ""

# Check for required environment variables
if [ -z "$MODEL" ]; then
    echo "ERROR: MODEL environment variable not set"
    echo "Example: export MODEL='hugging-quants/Meta-Llama-3.1-8B-Instruct-AWQ-INT4'"
    exit 1
fi

if [ -z "$LLAMA_CPP_PATH" ]; then
    echo "ERROR: LLAMA_CPP_PATH environment variable not set"
    echo "Example: export LLAMA_CPP_PATH='http://100.121.229.114:8000'"
    exit 1
fi

# Stop existing container if running
docker stop pedro-$SERVICE-test 2>/dev/null || true
docker rm pedro-$SERVICE-test 2>/dev/null || true

# Build the image
echo "Building $SERVICE container..."
docker build -f cli/$SERVICE/${SERVICE}Bot.Dockerfile -t pedro-$SERVICE-test .

# Run based on service type
if [ "$SERVICE" = "discord" ]; then
    if [ -z "$DISCORD_TOKEN" ]; then
        echo "ERROR: DISCORD_TOKEN environment variable not set"
        exit 1
    fi

    if [ -z "$POSTGRES_URL" ]; then
        echo "ERROR: POSTGRES_URL environment variable not set"
        exit 1
    fi

    echo "Starting Discord bot..."
    docker run --rm -it \
        --name pedro-discord-test \
        -p 6060:6060 \
        -e DISCORD_TOKEN="$DISCORD_TOKEN" \
        -e POSTGRES_URL="$POSTGRES_URL" \
        -e POSTGRES_VECTOR_URL="${POSTGRES_VECTOR_URL:-}" \
        -e LLAMA_CPP_PATH="$LLAMA_CPP_PATH" \
        -e MODEL="$MODEL" \
        -e OPENAI_API_KEY="test" \
        pedro-discord-test -errorLevel debug

elif [ "$SERVICE" = "twitch" ]; then
    if [ -z "$TWITCH_ID" ]; then
        echo "ERROR: TWITCH_ID environment variable not set"
        exit 1
    fi

    if [ -z "$TWITCH_SECRET" ]; then
        echo "ERROR: TWITCH_SECRET environment variable not set"
        exit 1
    fi

    if [ -z "$POSTGRES_URL" ]; then
        echo "ERROR: POSTGRES_URL environment variable not set"
        exit 1
    fi

    echo "Starting Twitch bot..."
    docker run --rm -it \
        --name pedro-twitch-test \
        -p 6061:6060 \
        -p 3000:3000 \
        -e TWITCH_ID="$TWITCH_ID" \
        -e TWITCH_SECRET="$TWITCH_SECRET" \
        -e POSTGRES_URL="$POSTGRES_URL" \
        -e POSTGRES_VECTOR_URL="${POSTGRES_VECTOR_URL:-}" \
        -e LLAMA_CPP_PATH="$LLAMA_CPP_PATH" \
        -e MODEL="$MODEL" \
        -e OPENAI_API_KEY="test" \
        pedro-twitch-test -errorLevel debug

else
    echo "ERROR: Invalid service '$SERVICE'"
    echo "Usage: $0 [discord|twitch]"
    exit 1
fi
