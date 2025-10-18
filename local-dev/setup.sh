#!/bin/bash

set -e

echo "=== Pedro Local Development Setup ==="
echo ""

# Check prerequisites
echo "Checking prerequisites..."

if ! command -v ollama &> /dev/null; then
    echo "❌ Ollama is not installed!"
    echo ""
    echo "Please install Ollama first:"
    echo "  macOS:   brew install ollama"
    echo "  Linux:   curl -fsSL https://ollama.com/install.sh | sh"
    echo "  Manual:  https://ollama.com/download"
    exit 1
fi
echo "✅ Ollama is installed"

if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed!"
    echo "Please install Docker Desktop: https://www.docker.com/products/docker-desktop"
    exit 1
fi
echo "✅ Docker is installed"

if ! docker compose version &> /dev/null; then
    echo "❌ Docker Compose is not installed or too old!"
    echo "Please install Docker Compose v2+: https://docs.docker.com/compose/install/"
    exit 1
fi
echo "✅ Docker Compose is installed"

echo ""
echo "Checking if Ollama is running..."
if ! curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    echo "⚠️  Ollama is not running!"
    echo ""
    echo "Starting Ollama in the background..."
    echo "Note: On macOS, Ollama may start automatically. If not, run 'ollama serve' in another terminal."

    # Try to start Ollama (works on Linux, may not work on macOS)
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        nohup ollama serve > /tmp/ollama.log 2>&1 &
        sleep 2
    else
        echo ""
        echo "Please start Ollama manually in another terminal:"
        echo "  ollama serve"
        echo ""
        read -p "Press Enter when Ollama is running..."
    fi
fi

# Verify Ollama is accessible
if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    echo "✅ Ollama is running"
else
    echo "❌ Cannot connect to Ollama at http://localhost:11434"
    echo "Please make sure 'ollama serve' is running in another terminal"
    exit 1
fi

echo ""
echo "Detecting hardware configuration..."

# Auto-detect system RAM for model recommendation
if [[ "$OSTYPE" == "darwin"* ]]; then
    TOTAL_RAM_GB=$(sysctl -n hw.memsize | awk '{print int($1/1024/1024/1024)}')
else
    TOTAL_RAM_GB=$(free -g | awk '/^Mem:/{print $2}')
fi

echo "System RAM: ${TOTAL_RAM_GB}GB"

# Recommend model based on RAM
if [ "$TOTAL_RAM_GB" -ge 80 ]; then
    RECOMMENDED_MODEL="qwen2.5-coder:72b-instruct"
    PERFORMANCE_TIER="studio (M3 Ultra / Mac Studio)"
elif [ "$TOTAL_RAM_GB" -ge 48 ]; then
    RECOMMENDED_MODEL="qwen2.5-coder:32b-instruct"
    PERFORMANCE_TIER="laptop (M1 Max / MacBook Pro)"
else
    RECOMMENDED_MODEL="qwen2.5-coder:7b-instruct"
    PERFORMANCE_TIER="basic"
fi

echo "🔍 Detected performance tier: ${PERFORMANCE_TIER}"
echo "📦 Recommended model: ${RECOMMENDED_MODEL}"
echo ""
echo "Pulling ${RECOMMENDED_MODEL}..."
echo "This may take a few minutes depending on your internet connection..."
echo ""

ollama pull "$RECOMMENDED_MODEL"

echo ""
echo "✅ Model pulled successfully!"
echo ""

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo "Creating .env file from template..."
    cp .env.template .env
    echo "⚠️  Please edit .env and add your configuration values"
fi

# Create prod.env if it doesn't exist
if [ ! -f ../prod.env ]; then
    echo "Creating ../prod.env file from template..."
    cp prod.env.template ../prod.env
    echo "⚠️  Please edit ../prod.env with your secrets"
    echo ""
    echo "Options:"
    echo "  1. Use 1Password references: op://vault/item/field"
    echo "  2. Use raw values: your_actual_token_here"
fi

echo ""
echo "=== Setup Complete! ==="
echo ""
echo "Next steps:"
echo "1. Edit .env and ../prod.env with your configuration"
echo "2. Make sure 'ollama serve' is running in another terminal"
echo "3. Run: ./run.sh [discord|twitch|both] [--tier=studio|laptop|auto]"
echo ""
echo "Available models in Ollama:"
ollama list
echo ""
echo "Performance Tiers (auto-detected by run.sh):"
echo "  studio  - M3 Ultra / Mac Studio (80GB+) → 72B model, 128k context"
echo "  laptop  - M1 Max / MacBook Pro (48GB+) → 32B model, 32k context"
echo "  basic   - Standard hardware (<48GB)    → 7B model, 8k context"
echo ""
echo "Manual model management:"
echo "  ollama pull qwen2.5-coder:72b-instruct  # Studio tier"
echo "  ollama pull qwen2.5-coder:32b-instruct  # Laptop tier"
echo "  ollama pull qwen2.5-coder:7b-instruct   # Basic tier (default)"
